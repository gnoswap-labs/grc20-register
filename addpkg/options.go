package addpkg

import (
	"log/slog"

	"github.com/gnoswap-labs/grc20-register/config"
)

type Option func(f *AddPkg)

// WithLogger specifies the logger for the faucet
func WithLogger(l *slog.Logger) Option {
	return func(f *AddPkg) {
		f.logger = l
	}
}

// WithConfig specifies the config for the faucet
func WithConfig(c *config.Config) Option {
	return func(f *AddPkg) {
		f.config = c
	}
}

// WithPrepareTxMessageFn specifies the faucet
// transaction message constructor
func WithPrepareTxMessageFn(prepareTxMsgFn PrepareTxMessageFn) Option {
	return func(f *AddPkg) {
		f.prepareTxMsgFn = prepareTxMsgFn
	}
}
