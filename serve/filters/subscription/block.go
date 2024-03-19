package subscription

import (
	"github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnoswap-labs/grc20-register/serve/conns"
	"github.com/gnoswap-labs/grc20-register/serve/encode"
	"github.com/gnoswap-labs/grc20-register/serve/spec"
)

const (
	NewHeadsEvent = "newHeads"
)

// BlockSubscription is the new-heads type
// subscription
type BlockSubscription struct {
	*baseSubscription
}

func NewBlockSubscription(conn conns.WSConnection) *BlockSubscription {
	return &BlockSubscription{
		baseSubscription: newBaseSubscription(conn),
	}
}

func (b *BlockSubscription) WriteResponse(id string, block *types.Block) error {
	encodedBlock, err := encode.PrepareValue(block.Header)
	if err != nil {
		return err
	}

	return b.conn.WriteData(spec.NewJSONSubscribeResponse(id, encodedBlock))
}
