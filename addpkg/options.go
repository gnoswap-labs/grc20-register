package addpkg

import (
	"log/slog"
)

type Option func(f *AddPkg)

// WithLogger specifies the logger for the faucet
func WithLogger(l *slog.Logger) Option {
	return func(f *AddPkg) {
		f.logger = l
	}
}

// WithPrepareTxMessageFn specifies the faucet
// transaction message constructor
func WithPrepareTxMessageFn(prepareTxMsgFn PrepareTxMessageFn) Option {
	return func(f *AddPkg) {
		f.prepareTxMsgFn = prepareTxMsgFn
	}
}
