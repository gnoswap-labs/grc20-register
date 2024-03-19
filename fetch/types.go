package fetch

import (
	core_types "github.com/gnolang/gno/tm2/pkg/bft/rpc/core/types"

	clientTypes "github.com/gnoswap-labs/grc20-register/client/types"
	"github.com/gnoswap-labs/grc20-register/events"
)

// Client defines the interface for the node (client) communication
type Client interface {
	// GetLatestBlockNumber returns the latest block height from the chain
	GetLatestBlockNumber() (uint64, error)

	// GetBlock returns specified block
	GetBlock(uint64) (*core_types.ResultBlock, error)

	// GetBlockResults returns the results of executing the transactions
	// for the specified block
	GetBlockResults(uint64) (*core_types.ResultBlockResults, error)

	// CreateBatch creates a new client batch
	CreateBatch() clientTypes.Batch

	// GetAbciQuery returns the result of an ABCI query
	GetAbciQuery(path string, data []byte) (*core_types.ResultABCIQuery, error)
}

// Events is the events API
type Events interface {
	// SignalEvent signals a new event to the event manager
	SignalEvent(events.Event)
}
