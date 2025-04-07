package baseapp

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/evm/types"
	"github.com/cosmos/cosmos-sdk/x/evm/types/ethtx"
	"github.com/ethereum/go-ethereum/common"
	"sort"
	"sync"
)

// TxGroup manages transaction grouping and processing
type TxGroup struct {
	evmTxMetas    []*TxMeta
	txDecoder     sdk.TxDecoder
	groups        map[string][]*TxMeta // todo map can be replaced with a slice of slices
	associateTxs  [][]byte
	otherTxs      [][]byte
	sequentialTxs [][]byte
	txGroupDAG    *SimpleDag
}

// TxMeta represents transaction metadata
type TxMeta struct {
	RawTx       []byte
	From        common.Address
	To          *common.Address
	Nonce       uint64
	Data        []byte
	IsAssociate bool
	Index       int
}

type txResult struct {
	index int
	meta  *TxMeta
	entry *sdk.DeliverTxEntry
	err   error
}

// NewTxGroup creates a new TxGroup instance with pre-allocated slices.
// The capacity parameter represents the expected total number of transactions.
// associateTxs is allocated with capacity/4, assuming ~25% of transactions are AssociateTx.
func NewTxGroup(txDecoder sdk.TxDecoder, capacity int) *TxGroup {
	return &TxGroup{
		evmTxMetas:    make([]*TxMeta, 0, capacity),
		txDecoder:     txDecoder,
		groups:        make(map[string][]*TxMeta),
		associateTxs:  make([][]byte, 0, capacity),
		otherTxs:      make([][]byte, 0, capacity),
		sequentialTxs: make([][]byte, 0, capacity),
		txGroupDAG: &SimpleDag{
			txs: make([][]byte, 0, capacity),
			dag: make([]int64, 0),
		},
	}
}

// GroupByTxs processes and groups transactions
func (app *BaseApp) GroupByTxs(txs [][]byte) (*TxGroup, error) {
	txCount := len(txs)
	txGroup := NewTxGroup(app.TxDecode, txCount)

	results := make(chan txResult, txCount)
	var wg sync.WaitGroup

	// Process transactions concurrently
	for i, tx := range txs {
		wg.Add(1)
		go func(idx int, rawTx []byte) {
			defer wg.Done()
			result := processTransaction(txGroup, idx, rawTx)
			results <- result
		}(i, tx)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	failedTxs := make(map[int]string)
	for result := range results {
		if result.err != nil {
			failedTxs[result.index] = result.err.Error()
			continue
		}
		if result.meta != nil {
			txGroup.evmTxMetas = append(txGroup.evmTxMetas, result.meta)
		} else if result.entry != nil {
			txGroup.otherTxs = append(txGroup.otherTxs, result.entry.Tx)
		}
	}

	if len(failedTxs) > 0 {
		return nil, fmt.Errorf("failed to process %d transactions", len(failedTxs))
	}

	if err := txGroup.groupByAddress(); err != nil {
		return nil, err
	}
	if err := txGroup.getAssociateTxs(); err != nil {
		return nil, err
	}

	return txGroup, nil
}

// processTransaction handles individual transaction processing
func processTransaction(txGroup *TxGroup, idx int, rawTx []byte) txResult {
	var result txResult
	defer func() {
		if r := recover(); r != nil {
			result = txResult{index: idx, err: fmt.Errorf("panic: %v", r)}
		}
	}()

	typedTx, err := txGroup.txDecoder(rawTx)
	if err != nil {
		return txResult{index: idx, err: fmt.Errorf("decode error: %v", err)}
	}

	//if !IsEVMMessage(typedTx) || true {
	if !IsEVMMessage(typedTx) {
		return txResult{index: idx, entry: &sdk.DeliverTxEntry{Tx: rawTx, TxIndex: idx}}
	}

	msgData, sender, isAssociate, err := txGroup.filterEvmTx(typedTx)
	if err != nil {
		return txResult{index: idx, err: err}
	}

	meta, err := txGroup.decodeEvmTx(msgData, sender, idx, rawTx, isAssociate)
	if err != nil {
		return txResult{index: idx, err: err}
	}

	result = txResult{index: idx, meta: meta}
	return result
}

// filterEvmTx extracts EVM transaction data
func (t *TxGroup) filterEvmTx(tx sdk.Tx) (interface{}, common.Address, bool, error) {
	msg := MustGetEVMTransactionMessage(tx)
	txData, err := types.UnpackTxData(msg.Data)
	if err != nil {
		return nil, common.Address{}, false, fmt.Errorf("unpack tx data: %v", err)
	}

	sender, err := msg.GetEvmSender()
	if err != nil {
		return nil, common.Address{}, false, fmt.Errorf("get sender: %v", err)
	}

	cachedValue := msg.Data.GetCachedValue()
	if cachedValue == nil {
		return nil, common.Address{}, false, fmt.Errorf("nil cached value")
	}

	_, isAssociate := txData.(*ethtx.AssociateTx)
	return cachedValue, sender, isAssociate, nil
}

// decodeEvmTx creates TxMeta from EVM transaction data
func (t *TxGroup) decodeEvmTx(msgData interface{}, sender common.Address, idx int, rawTx []byte, isAssociate bool) (*TxMeta, error) {
	txMeta := &TxMeta{
		RawTx:       rawTx,
		From:        sender,
		IsAssociate: isAssociate,
		Index:       idx,
	}

	switch tx := msgData.(type) {
	case *ethtx.DynamicFeeTx:
		txMeta.To = tx.GetTo()
		txMeta.Nonce = tx.GetNonce()
		txMeta.Data = tx.GetData()
	case *ethtx.AccessListTx:
		txMeta.To = tx.GetTo()
		txMeta.Nonce = tx.GetNonce()
		txMeta.Data = tx.GetData()

	case *ethtx.BlobTx:
		txMeta.To = tx.GetTo()
		txMeta.Nonce = tx.GetNonce()
		txMeta.Data = tx.GetData()

	case *ethtx.LegacyTx:
		txMeta.To = tx.GetTo()
		txMeta.Nonce = tx.GetNonce()
		txMeta.Data = tx.GetData()

	case *ethtx.AssociateTx:
		txMeta.To = nil
		txMeta.Nonce = 0
		txMeta.Data = nil

	default:
		return nil, fmt.Errorf("unsupported transaction type: %T", tx)
	}

	return txMeta, nil
}

// groupByAddress groups EVM transactions by sender address
func (t *TxGroup) groupByAddress() error {
	for _, meta := range t.evmTxMetas {
		addr := meta.From.Hex()
		t.groups[addr] = append(t.groups[addr], meta)
	}

	for _, group := range t.groups {
		sort.Slice(group, func(i, j int) bool {
			return group[i].Nonce < group[j].Nonce
		})
	}
	return nil
}

// getAssociateTxs gets associate transactions and stores them in associateTxs
func (t *TxGroup) getAssociateTxs() error {
	for addr, group := range t.groups {
		var remaining []*TxMeta

		for i := 0; i < len(group); i++ {
			if group[i].IsAssociate {
				t.associateTxs = append(t.associateTxs, group[i].RawTx)
			} else {
				remaining = append(remaining, group[i])
			}
		}

		if len(remaining) > 0 {
			t.groups[addr] = remaining
		} else {
			delete(t.groups, addr)
		}
	}
	return nil
}

func (t *TxGroup) buildSampDag() {
	t.batchAssociateTxs()
	t.batchOtherTxs()
	t.batchAccountGroup()

}

func (t *TxGroup) batchAssociateTxs() {
	if len(t.associateTxs) == 0 {
		return
	}
	t.txGroupDAG.txs = append(t.txGroupDAG.txs, t.associateTxs...)
	t.txGroupDAG.dag = append(t.txGroupDAG.dag, int64(len(t.associateTxs)))
}

func (t *TxGroup) batchOtherTxs() {
	if len(t.otherTxs) == 0 {
		return
	}
	t.txGroupDAG.txs = append(t.txGroupDAG.txs, t.otherTxs...)
	t.txGroupDAG.dag = append(t.txGroupDAG.dag, int64(len(t.otherTxs)))
}

// batchAccountGroup account group by column
func (t *TxGroup) batchAccountGroup() {
	maxLen := 0
	for _, Txs := range t.groups {
		if len(Txs) > maxLen {
			maxLen = len(Txs)
		}
	}

	for i := 0; i < maxLen; i++ {
		group := make([][]byte, 0)
		for _, Txs := range t.groups {
			if i < len(Txs) {
				group = append(group, Txs[i].RawTx)
			}
		}

		if len(group) > 0 {
			t.txGroupDAG.txs = append(t.txGroupDAG.txs, group...)
			t.txGroupDAG.dag = append(t.txGroupDAG.dag, int64(len(group)))
		}
	}
}

// MustGetEVMTransactionMessage extracts EVM message from transaction
func MustGetEVMTransactionMessage(tx sdk.Tx) *types.MsgEVMTransaction {
	if len(tx.GetMsgs()) != 1 {
		panic("EVM transaction must have exactly 1 message")
	}
	msg, ok := tx.GetMsgs()[0].(*types.MsgEVMTransaction)
	if !ok {
		panic("not EVM message")
	}
	return msg
}

// IsEVMMessage checks if transaction contains EVM message
func IsEVMMessage(tx sdk.Tx) bool {
	msgs := tx.GetMsgs()
	if len(msgs) != 1 {
		return false
	}
	_, ok := msgs[0].(*types.MsgEVMTransaction)
	return ok
}
