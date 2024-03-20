package addpkg

import (
	"github.com/gnolang/gno/gno.land/pkg/sdk/vm"

	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/gnoswap-labs/grc20-register/estimate"
)

// PrepareTxMessageFn is the callback method that
// constructs the addpkg transaction message
type PrepareTxMessageFn func(PrepareCfg) std.Msg

// PrepareCfg specifies the tx prepare configuration
type PrepareCfg struct {
	Creator crypto.Address // the creator address
	PkgName string
	PkgPath string
	Files   []*std.MemFile
}

// defaultPrepareTxMessage constructs the default
// addpkg transaction message
func defaultPrepareTxMessage(cfg PrepareCfg) std.Msg {
	// REF: https://github.com/gnolang/gno/blob/173d5a28fb7851c13a93be987451efd075c39a03/gno.land/pkg/sdk/vm/msgs.go#L14-L24
	msgAddPackage := vm.MsgAddPackage{
		Creator: cfg.Creator,
		Package: &std.MemPackage{
			Name:  cfg.PkgName,
			Path:  cfg.PkgPath,
			Files: cfg.Files,
		},
	}

	return msgAddPackage
}

// prepareTransaction prepares the transaction for signing
func prepareTransaction(
	estimator estimate.Estimator,
	msg std.Msg,
) *std.Tx {
	// Construct the transaction
	tx := &std.Tx{
		Msgs:       []std.Msg{msg},
		Signatures: nil,
	}

	// Prepare the gas fee
	gasFee := estimator.EstimateGasFee()
	gasWanted := estimator.EstimateGasWanted(tx)

	tx.Fee = std.NewFee(gasWanted, gasFee)

	return tx
}
