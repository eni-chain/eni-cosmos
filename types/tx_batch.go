package types

// DeliverTxEntry represents an individual transaction's request within a batch.
type DeliverTxEntry struct {
	Tx      []byte
	TxIndex int
}

// DeliverTxBatchRequest represents a request object for a batch of transactions.
// This can be extended to include request-level tracing or metadata
type DeliverTxBatchRequest struct {
	OtherEntries []*DeliverTxEntry
	SeqEntries   []*DeliverTxEntry
}
