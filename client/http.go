package client

import (
	"fmt"

	"github.com/gnolang/gno/tm2/pkg/amino"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/std"

	rpcClient "github.com/gnolang/gno/tm2/pkg/bft/rpc/client"
	core_types "github.com/gnolang/gno/tm2/pkg/bft/rpc/core/types"

	clientTypes "github.com/gnoswap-labs/grc20-register/client/types"
)

// Client is the TM2 HTTP client
type Client struct {
	client *rpcClient.HTTP
}

// NewClient creates a new TM2 HTTP client
func NewClient(remote string) *Client {
	return &Client{
		client: rpcClient.NewHTTP(remote, ""),
	}
}

// CreateBatch creates a new request batch
func (c *Client) CreateBatch() clientTypes.Batch {
	return &Batch{
		batch: c.client.NewBatch(),
	}
}

func (c *Client) GetLatestBlockNumber() (uint64, error) {
	status, err := c.client.Status()
	if err != nil {
		return 0, fmt.Errorf("unable to get chain status, %w", err)
	}

	return uint64(status.SyncInfo.LatestBlockHeight), nil
}

func (c *Client) GetBlock(blockNum uint64) (*core_types.ResultBlock, error) {
	bn := int64(blockNum)

	block, err := c.client.Block(&bn)
	if err != nil {
		return nil, fmt.Errorf("unable to get block, %w", err)
	}

	return block, nil
}

func (c *Client) GetBlockResults(blockNum uint64) (*core_types.ResultBlockResults, error) {
	bn := int64(blockNum)

	results, err := c.client.BlockResults(&bn)
	if err != nil {
		return nil, fmt.Errorf("unable to get block results, %w", err)
	}

	return results, nil
}

func (c *Client) GetAbciQuery(path string, data []byte) (*core_types.ResultABCIQuery, error) {
	return c.client.ABCIQuery(path, data)
}

func (c *Client) GetAccount(address crypto.Address) (std.Account, error) {
	path := fmt.Sprintf("auth/accounts/%s", address.String())

	queryResponse, err := c.client.ABCIQuery(path, []byte{})
	if err != nil {
		return nil, fmt.Errorf("unable to execute ABCI query, %w", err)
	}

	var queryData struct{ BaseAccount std.BaseAccount }

	if err := amino.UnmarshalJSON(queryResponse.Response.Data, &queryData); err != nil {
		return nil, err
	}

	return &queryData.BaseAccount, nil
}

func (c *Client) SendTransactionCommit(tx *std.Tx) (*core_types.ResultBroadcastTxCommit, error) {
	aminoTx, err := amino.Marshal(tx)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal transaction, %w", err)
	}

	return c.client.BroadcastTxCommit(aminoTx)
}
