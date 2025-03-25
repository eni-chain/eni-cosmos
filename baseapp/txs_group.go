package baseapp

import (
	"fmt"
	"sort"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/evm/types"
	"github.com/cosmos/cosmos-sdk/x/evm/types/ethtx"
	"github.com/ethereum/go-ethereum/common"
)

// TxGroup manages transaction grouping and processing
type TxGroup struct {
	evmTxMetas    []*TxMeta
	txDecoder     sdk.TxDecoder
	groups        map[string][]*TxMeta
	associateTxs  []*sdk.DeliverTxEntry
	otherTxs      []*sdk.DeliverTxEntry
	sequentialTxs []*sdk.DeliverTxEntry
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
		associateTxs:  make([]*sdk.DeliverTxEntry, 0, capacity/4),
		otherTxs:      make([]*sdk.DeliverTxEntry, 0, capacity),
		sequentialTxs: make([]*sdk.DeliverTxEntry, 0, capacity/2),
	}
}

// GroupByTxs processes and groups transactions
func (app *BaseApp) GroupByTxs(ctx sdk.Context, txs [][]byte) (*TxGroup, error) {
	txCount := len(txs)
	txGroup := NewTxGroup(app.TxDecode, txCount)

	results := make(chan txResult, txCount)
	var wg sync.WaitGroup

	// Process transactions concurrently
	for i, tx := range txs {
		wg.Add(1)
		go func(idx int, rawTx []byte) {
			defer wg.Done()
			result := processTransaction(ctx, txGroup, idx, rawTx)
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
			txGroup.otherTxs = append(txGroup.otherTxs, result.entry)
		}
	}

	if len(failedTxs) > 0 {
		ctx.Logger().Error("Failed transactions", "count", len(failedTxs), "details", failedTxs)
		return nil, fmt.Errorf("failed to process %d transactions", len(failedTxs))
	}

	if err := txGroup.groupByAddress(); err != nil {
		return nil, err
	}
	if err := txGroup.groupSequential(); err != nil {
		return nil, err
	}

	return txGroup, nil
}

// processTransaction handles individual transaction processing
func processTransaction(ctx sdk.Context, txGroup *TxGroup, idx int, rawTx []byte) txResult {
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

	msgData, sender, isAssociate, err := txGroup.filterEvmTx(ctx, typedTx)
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
func (t *TxGroup) filterEvmTx(ctx sdk.Context, tx sdk.Tx) (interface{}, common.Address, bool, error) {
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

// groupSequential organizes transactions into sequential groups with priority for associateTxs
func (t *TxGroup) groupSequential() error {
	for _, group := range t.groups {
		if len(group) == 1 {
			entry := &sdk.DeliverTxEntry{
				Tx:      group[0].RawTx,
				TxIndex: group[0].Index,
			}
			if group[0].IsAssociate {
				t.associateTxs = append(t.associateTxs, entry)
			} else {
				t.otherTxs = append(t.otherTxs, entry)
			}
			continue
		}

		for _, meta := range group {
			entry := &sdk.DeliverTxEntry{
				Tx:      meta.RawTx,
				TxIndex: meta.Index,
			}
			if meta.IsAssociate {
				t.associateTxs = append(t.associateTxs, entry)
			} else {
				t.sequentialTxs = append(t.sequentialTxs, entry)
			}
		}
	}
	return nil
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
