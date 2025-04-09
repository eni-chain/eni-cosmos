package tasks

import (
	"cosmossdk.io/log"
	store "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	io "io"
	"math/rand"
	"testing"
	"time"
	//"cosmossdk.io/api/tendermint/abci"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func mockDeliverTx(ctx sdk.Context, tx []byte) *abci.ExecTxResult {
	time.Sleep(500 * time.Microsecond)
	return &abci.ExecTxResult{
		Code:      0,
		GasWanted: 1000,
		GasUsed:   800,
	}
}

func createTestContext() sdk.Context {
	storeKey := sdk.NewKVStoreKey("test")
	multiStore := &mockMultiStore{
		stores: map[store.StoreKey]store.KVStore{
			storeKey: &mockKVStore{},
		},
	}
	return sdk.NewContext(multiStore, cmtproto.Header{Time: time.Now()}, false, log.NewNopLogger())
}

type mockMultiStore struct {
	stores map[store.StoreKey]store.KVStore
}

func (m *mockMultiStore) Write() {
	//TODO implement me
	panic("implement me")
}

func (m *mockMultiStore) GetStoreType() store.StoreType {

	//TODO implement me
	panic("implement me")
}

func (m *mockMultiStore) CacheWrap() store.CacheWrap {
	//TODO implement me
	panic("implement me")
}

func (m *mockMultiStore) CacheWrapWithTrace(w io.Writer, tc store.TraceContext) store.CacheWrap {
	//TODO implement me
	panic("implement me")
}

func (m *mockMultiStore) CacheMultiStoreWithVersion(version int64) (store.CacheMultiStore, error) {
	//TODO implement me
	panic("implement me")
}

func (m *mockMultiStore) GetStore(key store.StoreKey) store.Store {
	//TODO implement me
	panic("implement me")
}

func (m *mockMultiStore) TracingEnabled() bool {
	//TODO implement me
	panic("implement me")
}

func (m *mockMultiStore) SetTracer(w io.Writer) store.MultiStore {
	//TODO implement me
	panic("implement me")
}

func (m *mockMultiStore) SetTracingContext(context store.TraceContext) store.MultiStore {
	//TODO implement me
	panic("implement me")
}

func (m *mockMultiStore) LatestVersion() int64 {
	//TODO implement me
	panic("implement me")
}

func (m *mockMultiStore) GetKVStore(key store.StoreKey) store.KVStore {
	return m.stores[key]
}

func (m *mockMultiStore) CacheMultiStore() store.CacheMultiStore {
	return m
}

func (m *mockMultiStore) StoreKeys() []store.StoreKey {
	keys := make([]store.StoreKey, 0, len(m.stores))
	for k := range m.stores {
		keys = append(keys, k)
	}
	return keys
}

func (m *mockMultiStore) SetKVStores(_ func(store.StoreKey, store.KVStore) store.CacheWrap) store.MultiStore {
	return m
}

type mockKVStore struct {
	data map[string][]byte
}

func (m *mockKVStore) GetStoreType() store.StoreType {
	//TODO implement me
	panic("implement me")
}

func (m *mockKVStore) CacheWrap() store.CacheWrap {
	//TODO implement me
	panic("implement me")
}

func (m *mockKVStore) CacheWrapWithTrace(w io.Writer, tc store.TraceContext) store.CacheWrap {
	//TODO implement me
	panic("implement me")
}

func (m *mockKVStore) Iterator(start, end []byte) store.Iterator {
	//TODO implement me
	panic("implement me")
}

func (m *mockKVStore) ReverseIterator(start, end []byte) store.Iterator {
	//TODO implement me
	panic("implement me")
}

func (m *mockKVStore) Get(key []byte) []byte {
	if m.data == nil {
		return nil
	}
	return m.data[string(key)]
}

func (m *mockKVStore) Set(key, value []byte) {
	if m.data == nil {
		m.data = make(map[string][]byte)
	}
	m.data[string(key)] = value
}

func (m *mockKVStore) Has(key []byte) bool {
	if m.data == nil {
		return false
	}
	_, ok := m.data[string(key)]
	return ok
}

func (m *mockKVStore) Delete(key []byte) {
	if m.data != nil {
		delete(m.data, string(key))
	}
}

func createTestRequests(n int) *sdk.DeliverTxBatchRequest {
	batchReqs := &sdk.DeliverTxBatchRequest{
		TxEntries: make([]*sdk.DeliverTxEntry, 0, n),
	}

	reqs := make([]*sdk.DeliverTxEntry, n)
	for i := 0; i < n; i++ {
		txBytes := make([]byte, 32)
		_, err := rand.Read(txBytes)
		if err != nil {
			panic(err)
		}

		reqs[i] = &sdk.DeliverTxEntry{
			Tx:            txBytes,
			AbsoluteIndex: i,
			Checksum:      [32]byte{},
			SdkTx:         nil,
			TxTracer:      nil,
			//SimpleDag:     []int64{50000, 50000},
		}
	}
	batchReqs.TxEntries = reqs
	batchReqs.SimpleDag = []int64{50000, 50000}
	return batchReqs
}

func Benchmark_scheduler_ProcessAll_10000tx_16workers(b *testing.B) {
	const (
		txCount = 100000
		workers = 16
	)

	logger := log.NewNopLogger()
	scheduler := NewScheduler(workers, mockDeliverTx, logger)

	ctx := createTestContext()
	ctx.WithParallelExec(true)
	ctx.WithSimpleDag(true)
	reqs := createTestRequests(txCount)
	for i := 0; i < b.N; i++ {
		_, err := scheduler.ProcessAll(ctx, reqs)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.StopTimer()
	b.ReportMetric(float64(txCount)/b.Elapsed().Seconds(), "tx/s")
}
