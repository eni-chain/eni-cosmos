package baseapp

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/ethtx"
	"github.com/ethereum/go-ethereum/log"
	"sort"
	"sync"
)

type TxsGrouper interface {
	FilterEvmTxs(typedTx sdk.Tx, encodedTx []byte) (interface{}, error)
	DecodeEvmTxs(msgData interface{}, idx int, rawTx []byte) (*TxMeta, error)
	SortGlobalNonce() error
	GroupByAddressTxs() error
	GroupSequentialTxs() error
}

type TxGroup struct {
	evmTxMetas   []*TxMeta
	txDecoder    sdk.TxDecoder
	groups       map[string][]*TxMeta
	otherEntries []*sdk.DeliverTxEntry
	seqEntries   []*sdk.DeliverTxEntry
}

type TxMeta struct {
	RawTx []byte
	To    string
	Nonce uint64
	Data  []byte
	idx   int
}

func NewTxGroup(txDecoder sdk.TxDecoder) *TxGroup {
	return &TxGroup{
		evmTxMetas:   make([]*TxMeta, 0),
		txDecoder:    txDecoder,
		groups:       make(map[string][]*TxMeta),
		otherEntries: make([]*sdk.DeliverTxEntry, 0),
		seqEntries:   make([]*sdk.DeliverTxEntry, 0),
	}
}

func (app *BaseApp) GroupByTxs(ctx sdk.Context, txs [][]byte) (*TxGroup, error) {
	txGroup := NewTxGroup(app.TxDecode)
	wg := sync.WaitGroup{}
	// protected parallel append
	mu := sync.Mutex{}
	// record failed transactions
	failedTxs := make(map[int]string)

	for i, tx := range txs {
		wg.Add(1)
		go func(idx int, encodedTx []byte) {
			defer wg.Done()
			defer func() {
				if err := recover(); err != nil {
					mu.Lock()
					failedTxs[idx] = fmt.Sprintf("panic: %v", err)
					mu.Unlock()
					ctx.Logger().Error(fmt.Sprintf("panic during transaction decoding at index %d: %v", idx, err))
				}
			}()

			// Decode transaction
			typedTx, err := app.TxDecode(encodedTx)
			if err != nil {
				mu.Lock()
				failedTxs[idx] = fmt.Sprintf("decode error: %s", err)
				mu.Unlock()
				ctx.Logger().Error(fmt.Sprintf("error decoding transaction at index %d due to %s", idx, err))
				return
			}

			// Check if it's an EVM transaction
			if isEVM := IsEVMMessage(typedTx); isEVM {
				msgData, err := txGroup.FilterEvmTxs(typedTx, encodedTx)
				if err != nil {
					mu.Lock()
					failedTxs[idx] = fmt.Sprintf("filter error: %s", err)
					mu.Unlock()
					ctx.Logger().Error(fmt.Sprintf("error getting cached value from transaction at index %d", idx))
					return
				}

				txMeta, err := txGroup.DecodeEvmTxs(msgData, idx, encodedTx)
				if err != nil {
					mu.Lock()
					failedTxs[idx] = fmt.Sprintf("decode meta error: %s", err)
					mu.Unlock()
					ctx.Logger().Error(fmt.Sprintf("error getting tx meta and nonce from transaction at index %d due to %s", idx, err))
					return
				}

				mu.Lock()
				txGroup.evmTxMetas = append(txGroup.evmTxMetas, txMeta)
				mu.Unlock()
			} else {
				mu.Lock()
				txGroup.otherEntries = append(txGroup.otherEntries, &sdk.DeliverTxEntry{Tx: encodedTx, TxIndex: idx})
				mu.Unlock()
			}
		}(i, tx)
	}
	wg.Wait()

	// check failed transactions
	if len(failedTxs) > 0 {
		ctx.Logger().Error(fmt.Sprintf("Found %d failed transactions: %v", len(failedTxs), failedTxs))
	}

	// sort evm transactions by global nonce
	if err := txGroup.SortGlobalNonce(); err != nil {
		return nil, err
	}

	// group by address
	if err := txGroup.GroupByAddressTxs(); err != nil {
		return nil, err
	}

	// group sequential transactions
	if err := txGroup.GroupSequentialTxs(); err != nil {
		return nil, err
	}
	return txGroup, nil
}

// FilterEvmTxs filter evm transactions and cache value
func (t *TxGroup) FilterEvmTxs(typedTx sdk.Tx, encodedTx []byte) (interface{}, error) {
	msg := MustGetEVMTransactionMessage(typedTx)
	cachedValue := msg.Data.GetCachedValue()
	if cachedValue == nil {
		return nil, fmt.Errorf("error getting cached value")
	}
	return cachedValue, nil
}

// DecodeEvmTxs decode evm transactions and get tx meta and nonce
func (t *TxGroup) DecodeEvmTxs(msgData interface{}, idx int, rawTx []byte) (*TxMeta, error) {
	txMeta := &TxMeta{
		RawTx: rawTx,
		idx:   idx,
	}
	switch tx := msgData.(type) {
	case *ethtx.DynamicFeeTx:
		txMeta.To = tx.GetTo().Hex()
		txMeta.Nonce = tx.GetNonce()
		txMeta.Data = tx.GetData()
	case *ethtx.AccessListTx:
		txMeta.To = tx.GetTo().Hex()
		txMeta.Nonce = tx.GetNonce()
		txMeta.Data = tx.GetData()

	case *ethtx.BlobTx:
		txMeta.To = tx.GetTo().Hex()
		txMeta.Nonce = tx.GetNonce()
		txMeta.Data = tx.GetData()

	case *ethtx.LegacyTx:
		txMeta.To = tx.GetTo().Hex()
		txMeta.Nonce = tx.GetNonce()
		txMeta.Data = tx.GetData()

	default:
		return nil, fmt.Errorf("unsupported transaction type: %T", tx)
	}

	return txMeta, nil
}

// SortGlobalNonce sort evm transactions by global nonce
func (t *TxGroup) SortGlobalNonce() error {
	sort.Slice(t.evmTxMetas, func(i, j int) bool {
		return t.evmTxMetas[i].Nonce < t.evmTxMetas[j].Nonce
	})
	return nil
}

// GroupByAddressTxs group evm transactions by address and sort by nonce
func (t *TxGroup) GroupByAddressTxs() error {
	groups := make(map[string][]*TxMeta)
	for _, meta := range t.evmTxMetas {
		groups[meta.To] = append(groups[meta.To], meta)
	}

	// Sort by nonce in each group
	for to := range groups {
		sort.Slice(groups[to], func(i, j int) bool {
			return groups[to][i].Nonce < groups[to][j].Nonce
		})
	}
	t.groups = groups
	return nil
}

// GroupSequentialTxs group sequential evm transactions
func (t *TxGroup) GroupSequentialTxs() error {
	// Store grouped transactions into serialGroups (assuming that each group requires serial processing)
	for _, group := range t.groups {
		groupTxs := make([]*sdk.DeliverTxEntry, 0)
		// if group has only one transaction, it can be processed in parallel
		if len(group) == 1 {
			t.otherEntries = append(t.otherEntries, &sdk.DeliverTxEntry{
				Tx:      group[0].RawTx,
				TxIndex: group[0].idx,
			})
		}
		// group sequential transactions
		if len(group) > 1 {
			for _, meta := range group {
				groupTxs = append(groupTxs, &sdk.DeliverTxEntry{
					Tx:      meta.RawTx,
					TxIndex: meta.idx,
				})
			}
			t.seqEntries = append(t.seqEntries, groupTxs...)
		}
	}

	return nil
}

func MustGetEVMTransactionMessage(tx sdk.Tx) *ethtx.MsgEVMTransaction {
	if len(tx.GetMsgs()) != 1 {
		panic("EVM transaction must have exactly 1 message")
	}
	msg, ok := tx.GetMsgs()[0].(*ethtx.MsgEVMTransaction)
	if !ok {
		panic("not EVM message")
	}
	return msg
}

func IsEVMMessage(tx sdk.Tx) bool {
	hasEVMMsg := false
	for _, msg := range tx.GetMsgs() {
		switch msg.(type) {
		case *ethtx.MsgEVMTransaction:
			hasEVMMsg = true
		default:
			continue
		}
	}

	if hasEVMMsg && len(tx.GetMsgs()) != 1 {
		log.Error("EVM tx must have exactly one message")
		return false
	}

	return hasEVMMsg
}
