package addpkg

import (
	"errors"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/gnolang/gno/tm2/pkg/std"

	"github.com/gnoswap-labs/grc20-register/client"
	"github.com/gnoswap-labs/grc20-register/estimate"
	"github.com/gnoswap-labs/grc20-register/estimate/static"
	"github.com/gnoswap-labs/grc20-register/keyring"
	"github.com/gnoswap-labs/grc20-register/keyring/memory"

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
	estimator      estimate.Estimator // gas pricing estimations
	logger         *slog.Logger       // log feedback
	client         client.Client      // TM2 client
	keyring        keyring.Keyring    // the faucet keyring
	prepareTxMsgFn PrepareTxMessageFn // transaction message creator
}

// RegisterGrc20Token registers news grc20 token to pre-defined register contract
func RegisterGrc20Token(pkgPath string) error {
	// load envs
	gasFeeDenom := getEnv("GNO_GAS_FEE_DENOM", "ugnot")
	gasFeeAmountStr := getEnv("GNO_GAS_FEE_AMOUNT", "1000000")
	gasFeeAmount, err := strconv.ParseInt(gasFeeAmountStr, 10, 64)
	if err != nil {
		logger.Error("error parsing gas fee amount", "error", err.Error())
		return err
	}
	gasFeeWantedStr := getEnv("GNO_GAS_WANTED", "1000000000") // current max block gas after bump PR, https://github.com/gnolang/gno/pull/2065
	gasFeeWanted, err := strconv.ParseInt(gasFeeWantedStr, 10, 64)
	if err != nil {
		logger.Error("error parsing gas fee wanted", "error", err.Error())
		return err
	}

	gnoRpcUrl := getEnv("GNO_RPC_URL", "http://localhost:26657")
	gnoChainId := getEnv("GNO_CHAIN_ID", "dev")

	registerMnemonic := getEnv("GNO_REGISTER_MNEMONIC", "")

	// Create a new AddPkg instance
	estimator := static.New(
		std.NewCoin(gasFeeDenom, gasFeeAmount),
		gasFeeWanted,
	)
	client, err := client.NewClient(gnoRpcUrl)
	if err != nil {
		logger.Error("unable to create TM2 client", "error", err)
		return err
	}

	a := &AddPkg{
		estimator:      estimator,
		logger:         logger,
		client:         *client,
		keyring:        memory.New(registerMnemonic, 1),
		prepareTxMsgFn: defaultPrepareTxMessage,
	}

	// Register the GRC20 token
	return a.registerGrc20Token(pkgPath, gnoChainId)
}

func (a *AddPkg) registerGrc20Token(pkgPath, gnoChainId string) error {
	// Find an account that has balance to cover tx fee
	fundAccount, err := a.findFundedAccount()
	if err != nil {
		return err
	}

	// Prepare the transaction
	registerCode := strings.ReplaceAll(template, "pkgPath", pkgPath)

	pCfg := PrepareCfg{
		Creator: fundAccount.GetAddress(),
		PkgName: "token_register",
		PkgPath: pkgPath + "_gnoswap_register",
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
	return broadcastTransaction(a.client, tx)
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
		account, err := a.client.GetAccount(address)
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
