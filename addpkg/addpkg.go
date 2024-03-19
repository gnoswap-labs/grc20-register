package addpkg

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/gnolang/gno/tm2/pkg/std"

	"github.com/gnoswap-labs/grc20-register/client"
	"github.com/gnoswap-labs/grc20-register/config"
	"github.com/gnoswap-labs/grc20-register/estimate"
	"github.com/gnoswap-labs/grc20-register/estimate/static"
	"github.com/gnoswap-labs/grc20-register/keyring"
	"github.com/gnoswap-labs/grc20-register/keyring/memory"

	"github.com/go-chi/chi"
)

var errNoFundedAccount = errors.New("no funded account found")

// AddPkg
type AddPkg struct {
	estimator estimate.Estimator // gas pricing estimations
	logger    *slog.Logger       // log feedback
	client    client.Client      // TM2 client
	keyring   keyring.Keyring    // the faucet keyring

	mux *chi.Mux // HTTP routing

	config         *config.Config     // faucet configuration
	prepareTxMsgFn PrepareTxMessageFn // transaction message creator
}

var noopLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func NewAddPkg(
	estimator estimate.Estimator,
	client client.Client,
	opts ...Option,
) (*AddPkg, error) {
	a := &AddPkg{
		estimator:      estimator,
		client:         client,
		logger:         noopLogger,
		config:         config.DefaultConfig(),
		prepareTxMsgFn: defaultPrepareTxMessage,

		mux: chi.NewMux(),
	}

	// Validate the configuration
	if err := config.ValidateConfig(a.config); err != nil {
		return nil, fmt.Errorf("invalid configuration, %w", err)
	}

	// Generate the in-memory keyring
	a.keyring = memory.New(a.config.Mnemonic, 1)

	return a, nil
}

func RegisterGrc20Token(pkgPath string) error {
	// Create a new AddPkg instance
	estimator := static.New(
		std.NewCoin("ugnot", 1000), // r3v4_xxx: HARDCODED
		10000000,
	)
	client := client.NewClient("http://localhost:26657") // r3v4_xxx: HARDCODED

	a, err := NewAddPkg(estimator, *client)
	if err != nil {
		return err
	}

	// Register the GRC20 token
	return a.registerGrc20Token(pkgPath)
}

func (a *AddPkg) registerGrc20Token(pkgPath string) error {
	// Find an account that has balance to cover the transfer
	fundAccount, err := a.findFundedAccount()
	if err != nil {
		return err
	}

	// Prepare the transaction
	template := "package token_register\n\nimport (\n\ttoken \"pkgPath\"\n\n\tpusers \"gno.land/p/demo/users\"\n\n\tpl \"gno.land/r/demo/pool\"\n\trr \"gno.land/r/demo/router\"\n\tsr \"gno.land/r/demo/staker\"\n)\n\ntype NewToken struct{}\n\nfunc (NewToken) Transfer() func(to pusers.AddressOrName, amount uint64) {\n\treturn token.Transfer\n}\n\nfunc (NewToken) TransferFrom() func(from, to pusers.AddressOrName, amount uint64) {\n\treturn token.TransferFrom\n}\n\nfunc (NewToken) BalanceOf() func(owner pusers.AddressOrName) uint64 {\n\treturn token.BalanceOf\n}\n\nfunc (NewToken) Approve() func(spender pusers.AddressOrName, amount uint64) {\n\treturn token.Approve\n}\n\nfunc init() {\n\tpl.RegisterGRC20Interface(\"pkgPath\", NewToken{})\n\n\trr.RegisterGRC20Interface(\"pkgPath\", NewToken{})\n\n\tsr.RegisterGRC20Interface(\"pkgPath\", NewToken{})\n\n\tprintln(99999999999999)\n}\n"
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
		chainID:       a.config.ChainID,
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
	// return broadcastTransaction(a.client, tx)

	// Braodcast the transaction sync
	return broadcastTransaction(a.client, tx)
}

// findFundedAccount finds an account
// whose balance is enough to cover the send amount
func (a *AddPkg) findFundedAccount() (std.Account, error) {
	// A funded account is an account that can
	// cover the initial transfer fee, as well
	// as the send amount
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
