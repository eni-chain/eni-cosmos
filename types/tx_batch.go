package types

import (
	"cosmossdk.io/store/multiversion"
	abci "github.com/cometbft/cometbft/abci/types"
)

// DeliverTxEntry represents an individual transaction's request within a batch.
// This can be extended to include tx-level metadata
type DeliverTxEntry struct {
	//Request            abci.RequestDeliverTx
	SdkTx              Tx
	Checksum           [32]byte
	AbsoluteIndex      int
	EstimatedWritesets MappedWritesets
	TxTracer           TxTracer
	//SimpleDag          []int64
	Tx      []byte
	TxIndex int
}

// EstimatedWritesets represents an estimated writeset for a transaction mapped by storekey to the writeset estimate.
type MappedWritesets map[StoreKey]multiversion.WriteSet

// DeliverTxBatchRequest represents a request object for a batch of transactions.
// This can be extended to include request-level tracing or metadata
type DeliverTxBatchRequest struct {
	TxEntries []*DeliverTxEntry
	SimpleDag []int64
}

// DeliverTxResult represents an individual transaction's response within a batch.
// This can be extended to include tx-level tracing or metadata
type DeliverTxResult struct {
	Response abci.ResponseDeliverTx
}

// DeliverTxBatchResponse represents a response object for a batch of transactions.
// This can be extended to include response-level tracing or metadata
type DeliverTxBatchResponse struct {
	Results []*DeliverTxResult
}
