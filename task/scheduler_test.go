package tasks

import (
	"cosmossdk.io/log"
	"cosmossdk.io/store/multiversion"
	store "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/types"
	io "io"
	"math/rand"
	"reflect"
	"testing"
	"time"
	//"cosmossdk.io/api/tendermint/abci"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func Test_scheduler_ProcessAll(t *testing.T) {
	type fields struct {
		deliverTx          func(ctx sdk.Context, tx []byte) *abci.ExecTxResult
		workers            int
		multiVersionStores map[sdk.StoreKey]multiversion.MultiVersionStore
		allTasksMap        map[int]*deliverTxTask
		allTasks           []*deliverTxTask
		executeCh          chan func()
		validateCh         chan func()
		metrics            *schedulerMetrics
		synchronous        bool
		maxIncarnation     int
		loger              log.Logger
	}
	type args struct {
		ctx  types.Context
		reqs []*sdk.DeliverTxEntry
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*abci.ExecTxResult
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &scheduler{
				deliverTx:          tt.fields.deliverTx,
				workers:            tt.fields.workers,
				multiVersionStores: tt.fields.multiVersionStores,
				allTasksMap:        tt.fields.allTasksMap,
				allTasks:           tt.fields.allTasks,
				executeCh:          tt.fields.executeCh,
				validateCh:         tt.fields.validateCh,
				metrics:            tt.fields.metrics,
				synchronous:        tt.fields.synchronous,
				maxIncarnation:     tt.fields.maxIncarnation,
				loger:              tt.fields.loger,
			}
			got, err := s.ProcessAll(tt.args.ctx, tt.args.reqs)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ProcessAll() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Benchmark_scheduler_ProcessAll(b *testing.B) {
	benchmarks := []struct {
		name string
	}{
		{
			name: "Benchmark_scheduler_ProcessAll",
		},
	}
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {

			}
		})
	}
}

func mockDeliverTx(ctx sdk.Context, tx []byte) *abci.ExecTxResult {
	time.Sleep(100 * time.Microsecond)
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

func createTestRequests(n int) []*sdk.DeliverTxEntry {
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
		}
	}
	return reqs
}

func Benchmark_scheduler_ProcessAll_10000tx_16workers(b *testing.B) {
	const (
		txCount = 10000
		workers = 16
	)

	logger := log.NewNopLogger()
	scheduler := NewScheduler(workers, mockDeliverTx, logger)

	ctx := createTestContext()
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
