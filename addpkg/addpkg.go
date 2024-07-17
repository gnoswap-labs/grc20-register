package addpkg

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/gnolang/gno/tm2/pkg/std"

	"github.com/gnolang/faucet/estimate"
	"github.com/gnolang/faucet/estimate/static"
	"github.com/gnolang/faucet/keyring"
	"github.com/gnolang/faucet/keyring/memory"

	faucetClient "github.com/gnolang/faucet/client/http"
	rpcClient "github.com/gnolang/gno/tm2/pkg/bft/rpc/client"
	"github.com/gnolang/tx-indexer/client"

	_ "github.com/joho/godotenv/autoload"

	_ "embed"
)

//go:embed template.txt
var template string // register contract template

// Errors
var (
	errNoFundedAccount = errors.New("no funded account found")
)

var logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

// AddPkg
type AddPkg struct {
	estimator      estimate.Estimator  // gas pricing estimations
	logger         *slog.Logger        // log feedback
	client         client.Client       // TM2 client
	faucetClient   faucetClient.Client // the faucet client
	rpcClient      rpcClient.RPCClient // the rpc client
	keyring        keyring.Keyring     // the faucet keyring
	prepareTxMsgFn PrepareTxMessageFn  // transaction message creator
}

// RegisterGrc20Token registers news grc20 token to pre-defined register contract
func RegisterGrc20Token(pkgPath string) error {
	gnoRpcUrl := getEnv("GNO_RPC_URL", "http://localhost:26657")
	client, err := client.NewClient(gnoRpcUrl)
	if err != nil {
		logger.Error("unable to create TM2 client", "error", err)
		return err
	}

	rClient, err := rpcClient.NewHTTPClient(gnoRpcUrl)
	if err != nil {
		logger.Error("unable to create rpc client", "error", err)
		return err
	}

	registered, err := checkIfTokenRegistered(rClient, pkgPath)
	if err != nil {
		return err
	}
	if registered {
		return fmt.Errorf("token already registered: %s", pkgPath)
	}

	// load envs
	gasFeeDenom := getEnv("GNO_GAS_FEE_DENOM", "ugnot")
	gasFeeAmountStr := getEnv("GNO_GAS_FEE_AMOUNT", "1000000")
	gasFeeAmount, err := strconv.ParseInt(gasFeeAmountStr, 10, 64)
	if err != nil {
		logger.Error("error parsing gas fee amount", "error", err.Error())
		return err
	}
	gasFeeWantedStr := getEnv("GNO_GAS_WANTED", "100000000") // current max block gas after bump PR, https://github.com/gnolang/gno/pull/2065
	gasFeeWanted, err := strconv.ParseInt(gasFeeWantedStr, 10, 64)
	if err != nil {
		logger.Error("error parsing gas fee wanted", "error", err.Error())
		return err
	}

	// Create a new AddPkg instance
	estimator := static.New(
		std.NewCoin(gasFeeDenom, gasFeeAmount),
		gasFeeWanted,
	)

	// faucet client
	fClient, err := faucetClient.NewClient(gnoRpcUrl)
	if err != nil {
		logger.Error("unable to create faucet client", "error", err)
		return err
	}

	registerMnemonic := getEnv("GNO_REGISTER_MNEMONIC", "")
	a := &AddPkg{
		estimator:      estimator,
		logger:         logger,
		client:         *client,
		faucetClient:   *fClient,
		rpcClient:      *rClient,
		keyring:        memory.New(registerMnemonic, 1),
		prepareTxMsgFn: defaultPrepareTxMessage,
	}

	// Register the GRC20 token
	gnoChainId := getEnv("GNO_CHAIN_ID", "dev")
	return a.registerGrc20Token(pkgPath, gnoChainId) // #87 func
}

func (a *AddPkg) registerGrc20Token(pkgPath, gnoChainId string) error {
	// Find an account that has balance to cover tx fee
	fundAccount, err := a.findFundedAccount()
	if err != nil {
		return err
	}

	// Prepare the transaction
	registerCode := strings.ReplaceAll(template, "pkgPath", pkgPath)

	_removeCommon := strings.Replace(pkgPath, "gno.land/r/", "", 1)
	pathToRegister := "gno.land/r/g1er355fkjksqpdtwmhf5penwa82p0rhqxkkyhk5/" + _removeCommon

	/*
		orig:					gno.land/r/gnoswap/gns
		remove:				gnoswap/gns
		toRegister: 	gno.land/r/g1er355fkjksqpdtwmhf5penwa82p0rhqxkkyhk5/gnoswap/gns
	*/

	pCfg := PrepareCfg{
		Creator: fundAccount.GetAddress(),
		PkgName: "token_register",
		PkgPath: pathToRegister,
		Files: []*std.MemFile{
			{
				Name: "register.gno",
				Body: registerCode,
			},
		},
	}
	tx := prepareTransaction(a.estimator, a.prepareTxMsgFn(pCfg))

	// Sign the transaction
	sCfg := signCfg{
		chainID:       gnoChainId,
		accountNumber: fundAccount.GetAccountNumber(),
		sequence:      fundAccount.GetSequence(),
	}

	if err := signTransaction(
		tx,
		a.keyring.GetKey(fundAccount.GetAddress()),
		sCfg,
	); err != nil {
		return err
	}

	// Broadcast the transaction
	_, err = a.faucetClient.SendTransactionCommit(tx)
	return err
}

// findFundedAccount finds an account
// whose balance is enough to cover tx fee
func (a *AddPkg) findFundedAccount() (std.Account, error) {
	// A funded account is an account that can
	// cover the initial addpkg fee
	estimatedFee := a.estimator.EstimateGasFee()
	requiredFunds := std.NewCoins(estimatedFee)

	for _, address := range a.keyring.GetAddresses() {
		// Fetch the account
		account, err := a.faucetClient.GetAccount(address)
		if err != nil {
			a.logger.Error(
				"unable to fetch account",
				"address",
				address.String(),
				"error",
				err,
			)

			continue
		}

		// Fetch the balance
		balance := account.GetCoins()

		// Make sure there are enough funds
		if balance.IsAllLT(requiredFunds) {
			a.logger.Error(
				"account cannot serve requests",
				"address",
				address.String(),
				"balance",
				balance.String(),
				"amount",
				requiredFunds,
			)

			continue
		}

		return account, nil
	}

	return nil, errNoFundedAccount
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	if fallback == "" {
		panic("missing required env variable: " + key)
	}

	return fallback
}

func checkIfTokenRegistered(client *rpcClient.RPCClient, pkgPath string) (bool, error) {
	poolContract := getEnv("POOL_CONTRACT_PATH", "gno.land/r/gnoswap/pool")
	payload := fmt.Sprintf("%s.GetRegisteredTokens()", poolContract)

	res, err := client.ABCIQuery("vm/qeval", []byte(payload))
	if err != nil {
		logger.Error("unable to fetch registered tokens", "error", err)
		return false, err
	}
	tokens := string(res.Response.Data)

	registered := strings.Contains(tokens, pkgPath)
	if registered {
		logger.Info("token already registered", "pkgPath", pkgPath)
		return true, nil
	}

	return false, nil
}
