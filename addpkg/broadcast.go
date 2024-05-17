package addpkg

import (
	"fmt"

	"github.com/gnolang/gno/tm2/pkg/amino"

	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/gnoswap-labs/grc20-register/client"

	rpcClient "github.com/gnolang/gno/tm2/pkg/bft/rpc/client"

	core_types "github.com/gnolang/gno/tm2/pkg/bft/rpc/core/types"
)

// Client is the TM2 HTTP client
type Client struct {
	client *rpcClient.RPCClient
}

// broadcastTransaction broadcasts the transaction using a COMMIT send
func broadcastTransaction(client client.Client, tx *std.Tx) error {
	// Send the transaction.
	// NOTE: Commit sends are temporary. Once
	// there is support for event indexing, this
	// call will change to a sync send
	response, err := client.SendTransactionCommit(tx)
	if err != nil {
		return fmt.Errorf("unable to send transaction, %w", err)
	}

	// Check the errors
	if response.CheckTx.IsErr() {
		return fmt.Errorf("transaction failed initial validation, %w", response.CheckTx.Error)
	}

	if response.DeliverTx.IsErr() {
		return fmt.Errorf("transaction failed during execution, %w", response.DeliverTx.Error)
	}

	return nil
}

func (c *Client) SendTransactionCommit(tx *std.Tx) (*core_types.ResultBroadcastTxCommit, error) {
	aminoTx, err := amino.Marshal(tx)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal transaction, %w", err)
	}

	return c.client.BroadcastTxCommit(aminoTx)
}
