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

	"github.com/joho/godotenv"
)

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

func init() {
	err := godotenv.Load("addpkg/.env")
	if err != nil {
		logger.Error("Error loading .env file", "error", err.Error())
		os.Exit(-1)
	}
}

// REGISTER CONTRACT TEMPLATE
// below is template for registering grc20 tokens to gnoswap
// REF: https://github.com/gnoswap-labs/gnoswap/blob/97b7b4124328338ff379286c3a7b35de560c42e9/pool/token_register.gno#L51-L70
var registerContractTemplate = "package token_register\n\nimport (\n\ttoken \"pkgPath\"\n\n\tpusers \"gno.land/p/demo/users\"\n\n\tpl \"gno.land/r/demo/pool\"\n\trr \"gno.land/r/demo/router\"\n\tsr \"gno.land/r/demo/staker\"\n)\n\ntype NewToken struct{}\n\nfunc (NewToken) Transfer() func(to pusers.AddressOrName, amount uint64) {\n\treturn token.Transfer\n}\n\nfunc (NewToken) TransferFrom() func(from, to pusers.AddressOrName, amount uint64) {\n\treturn token.TransferFrom\n}\n\nfunc (NewToken) BalanceOf() func(owner pusers.AddressOrName) uint64 {\n\treturn token.BalanceOf\n}\n\nfunc (NewToken) Approve() func(spender pusers.AddressOrName, amount uint64) {\n\treturn token.Approve\n}\n\nfunc init() {\n\tpl.RegisterGRC20Interface(\"pkgPath\", NewToken{})\n\n\trr.RegisterGRC20Interface(\"pkgPath\", NewToken{})\n\n\tsr.RegisterGRC20Interface(\"pkgPath\", NewToken{})\n\n}\n"

// RegisterGrc20Token registers news grc20 token to pre-defined register contract
// - which is defined #48 `registerContractTemplate`
func RegisterGrc20Token(pkgPath string) error {
	// load envs
	gasFeeDenom := os.Getenv("GNO_GAS_FEE_DENOM")
	gasFeeAmountStr := os.Getenv("GNO_GAS_FEE_AMOUNT")
	gasFeeAmount, err := strconv.ParseInt(gasFeeAmountStr, 10, 64)
	if err != nil {
		logger.Error("error parsing gas fee amount", "error", err.Error())
		return err
	}
	gasFeeWantedStr := os.Getenv("GNO_GAS_WANTED")
	gasFeeWanted, err := strconv.ParseInt(gasFeeWantedStr, 10, 64)
	if err != nil {
		logger.Error("error parsing gas fee wanted", "error", err.Error())
		return err
	}

	gnoRpcUrl := os.Getenv("GNO_RPC_URL")
	gnoChainId := os.Getenv("GNO_CHAIN_ID")

	registerMnemonic := os.Getenv("GNO_REGISTER_MNEMONIC")

	// Create a new AddPkg instance
	estimator := static.New(
		std.NewCoin(gasFeeDenom, gasFeeAmount),
		gasFeeWanted,
	)
	client := client.NewClient(gnoRpcUrl)

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
	registerCode := strings.ReplaceAll(registerContractTemplate, "pkgPath", pkgPath)

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
