package tasks

import (
	"cosmossdk.io/log"
	"cosmossdk.io/store/multiversion"
	"cosmossdk.io/store/multiversion/occ"
	storetypes "cosmossdk.io/store/types"
	"github.com/cometbft/cometbft/abci/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/task/mocks"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/rand"
	"testing"
	"time"
)

func TestDeliverTxTask_Status(t *testing.T) {
	task := &deliverTxTask{
		Status:       statusPendingInt,
		Dependencies: make(map[int]struct{}),
	}

	// Test IsStatus
	assert.True(t, task.IsStatus(statusPendingInt), "should be pending")
	assert.False(t, task.IsStatus(statusExecutedInt), "should not be executed")

	// Test SetStatus
	task.SetStatus(statusExecutedInt)
	assert.True(t, task.IsStatus(statusExecutedInt), "should be executed")
	assert.False(t, task.IsStatus(statusPendingInt), "should not be pending")
}

func TestDeliverTxTask_AppendDependencies(t *testing.T) {
	task := &deliverTxTask{
		Dependencies: make(map[int]struct{}),
	}

	deps := []int{1, 2, 3}
	task.AppendDependencies(deps)

	assert.Len(t, task.Dependencies, 3, "should have 3 dependencies")
	for _, dep := range deps {
		_, exists := task.Dependencies[dep]
		assert.True(t, exists, "dependency %d should exist", dep)
	}
}

func TestDeliverTxTask_ResetAndIncrement(t *testing.T) {
	task := &deliverTxTask{
		Status:        statusExecutedInt,
		Response:      &types.ExecTxResult{},
		Abort:         &occ.Abort{},
		AbortCh:       make(chan occ.Abort),
		Incarnation:   1,
		VersionStores: make(map[sdk.StoreKey]*multiversion.VersionIndexedStore),
	}

	// Test Reset
	task.Reset()
	task.Status = statusPendingInt
	expected := task.Status
	assert.Equal(t, expected, task.Status, "status should be pending")
	assert.Nil(t, task.Response, "response should be nil")
	assert.Nil(t, task.Abort, "abort should be nil")
	assert.Nil(t, task.AbortCh, "abortCh should be nil")
	assert.Nil(t, task.VersionStores, "versionStores should be nil")
	assert.Equal(t, 1, task.Incarnation, "incarnation should not change")

	// Test Increment
	task.Increment()
	assert.Equal(t, 2, task.Incarnation, "incarnation should be 2")
}

func TestScheduler_ProcessAll(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock dependencies
	mockMultiVersionStore := mocks.NewMockMultiVersionStore(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockMultiStore := mocks.NewMockMultiStore(ctrl)
	mockCacheMultiStore := mocks.NewMockCacheMultiStore(ctrl)

	// Setup scheduler
	deliverTxFunc := func(ctx sdk.Context, tx []byte) *types.ExecTxResult {
		return &types.ExecTxResult{Code: 0, Data: tx}
	}
	scheduler := NewScheduler(2, deliverTxFunc, mockLogger).(*scheduler)

	// Setup tasks
	reqs := &sdk.DeliverTxBatchRequest{
		TxEntries: []*sdk.DeliverTxEntry{
			{
				Tx:            []byte("tx1"),
				AbsoluteIndex: 0,
				Checksum:      [32]byte{},
			},
			{
				Tx:            []byte("tx2"),
				AbsoluteIndex: 1,
				Checksum:      [32]byte{},
			},
		},
	}

	// Mock MultiVersionStore behavior
	storeKey := storetypes.NewKVStoreKey("test")
	scheduler.multiVersionStores = map[sdk.StoreKey]multiversion.MultiVersionStore{
		storeKey: mockMultiVersionStore,
	}

	// Mock context
	ctx := sdk.NewContext(mockMultiStore, cmtproto.Header{}, false, mockLogger)

	// Expectations
	mockMultiStore.EXPECT().StoreKeys().Return([]sdk.StoreKey{storeKey}).AnyTimes()
	mockMultiStore.EXPECT().CacheMultiStore().Return(mockCacheMultiStore).AnyTimes()
	mockCacheMultiStore.EXPECT().SetKVStores(gomock.Any()).Return(mockCacheMultiStore).AnyTimes()
	mockMultiVersionStore.EXPECT().VersionedIndexedStore(gomock.Any(), gomock.Any(), gomock.Any()).Return(&multiversion.VersionIndexedStore{}).AnyTimes()
	mockMultiVersionStore.EXPECT().ValidateTransactionState(gomock.Any()).Return(true, []int{}).AnyTimes()
	mockMultiVersionStore.EXPECT().WriteLatestToStore().Times(1)
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	// Run ProcessAll
	results, err := scheduler.ProcessAll(ctx, reqs)
	require.NoError(t, err, "ProcessAll should succeed")
	assert.Len(t, results, 2, "should return 2 results")
	assert.Equal(t, []byte("tx1"), results[0].Data, "first result data should match")
	assert.Equal(t, []byte("tx2"), results[1].Data, "second result data should match")

	// Verify metrics
	assert.Equal(t, 0, scheduler.metrics.retries, "no retries expected")
	assert.Equal(t, 0, scheduler.metrics.maxIncarnation, "no incarnations expected")
}

func TestScheduler_ExecuteTask(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock dependencies
	mockMultiVersionStore := mocks.NewMockMultiVersionStore(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockMultiStore := mocks.NewMockMultiStore(ctrl)
	mockCacheMultiStore := mocks.NewMockCacheMultiStore(ctrl)

	// Setup scheduler
	deliverTxFunc := func(ctx sdk.Context, tx []byte) *types.ExecTxResult {
		return &types.ExecTxResult{Code: 0, Data: tx}
	}
	scheduler := NewScheduler(1, deliverTxFunc, mockLogger).(*scheduler)
	scheduler.multiVersionStores = map[sdk.StoreKey]multiversion.MultiVersionStore{
		storetypes.NewKVStoreKey("test"): mockMultiVersionStore,
	}

	// Setup task
	task := &deliverTxTask{
		Tx:            []byte("test-tx"),
		AbsoluteIndex: 0,
		Incarnation:   0,
		Status:        statusPendingInt,
		Dependencies:  make(map[int]struct{}),
	}

	// Mock context
	ctx := sdk.NewContext(mockMultiStore, cmtproto.Header{}, false, mockLogger)

	// Expectations
	mockMultiStore.EXPECT().CacheMultiStore().Return(mockCacheMultiStore)
	mockCacheMultiStore.EXPECT().SetKVStores(gomock.Any()).Return(mockCacheMultiStore)
	mockMultiVersionStore.EXPECT().VersionedIndexedStore(0, 0, gomock.Any()).Return(&multiversion.VersionIndexedStore{})
	//mockMultiVersionStore.EXPECT().WriteToMultiVersionStore().Times(0) // VersionIndexedStore method, not mocked here
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	// Run executeTask
	scheduler.executeTask(task, ctx)

	// Verify
	assert.Equal(t, statusExecutedInt, task.Status, "task should be executed")
	assert.NotNil(t, task.Response, "response should be set")
	assert.Equal(t, []byte("test-tx"), task.Response.Data, "response data should match")
}

func TestScheduler_ExecuteTask_Abort(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock dependencies
	mockMultiVersionStore := mocks.NewMockMultiVersionStore(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockMultiStore := mocks.NewMockMultiStore(ctrl)
	mockCacheMultiStore := mocks.NewMockCacheMultiStore(ctrl)

	// Setup scheduler
	deliverTxFunc := func(ctx sdk.Context, tx []byte) *types.ExecTxResult {
		return &types.ExecTxResult{Code: 0, Data: tx}
	}
	scheduler := NewScheduler(1, deliverTxFunc, mockLogger).(*scheduler)
	scheduler.multiVersionStores = map[sdk.StoreKey]multiversion.MultiVersionStore{
		storetypes.NewKVStoreKey("test"): mockMultiVersionStore,
	}

	// Setup task
	task := &deliverTxTask{
		Tx:            []byte("test-tx"),
		AbsoluteIndex: 0,
		Incarnation:   0,
		Status:        statusPendingInt,
		Dependencies:  make(map[int]struct{}),
	}

	// Mock context
	ctx := sdk.NewContext(mockMultiStore, cmtproto.Header{}, false, mockLogger)

	// Simulate abort
	abortCh := make(chan occ.Abort, 1)
	abortCh <- occ.Abort{DependentTxIdx: 1}
	close(abortCh)

	// Expectations
	mockMultiStore.EXPECT().CacheMultiStore().Return(mockCacheMultiStore)
	mockCacheMultiStore.EXPECT().SetKVStores(gomock.Any()).Return(mockCacheMultiStore)
	mockMultiVersionStore.EXPECT().VersionedIndexedStore(0, 0, gomock.Any()).Return(&multiversion.VersionIndexedStore{})
	//mockMultiVersionStore.EXPECT().WriteEstimatesToMultiVersionStore().Times(0) // VersionIndexedStore method
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	// Run executeTask
	scheduler.executeTask(task, ctx)

	// Verify
	assert.Equal(t, statusAbortedInt, task.Status, "task should be aborted")
	assert.Nil(t, task.Response, "response should be nil")
	assert.NotNil(t, task.Abort, "abort should be set")
	assert.Equal(t, 1, task.Abort.DependentTxIdx, "abort dependent index should be 1")
	assert.Len(t, task.Dependencies, 1, "should have 1 dependency")
}

func TestScheduler_ValidateTask(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock dependencies
	mockMultiVersionStore := mocks.NewMockMultiVersionStore(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockMultiStore := mocks.NewMockMultiStore(ctrl)

	// Setup scheduler
	scheduler := NewScheduler(1, nil, mockLogger).(*scheduler)
	scheduler.multiVersionStores = map[sdk.StoreKey]multiversion.MultiVersionStore{
		storetypes.NewKVStoreKey("test"): mockMultiVersionStore,
	}
	scheduler.allTasksMap = make(map[int]*deliverTxTask)

	// Setup tasks
	task := &deliverTxTask{
		AbsoluteIndex: 0,
		Incarnation:   0,
		Status:        statusExecutedInt,
		Dependencies:  make(map[int]struct{}),
	}
	scheduler.allTasksMap[0] = task

	// Mock context
	ctx := sdk.NewContext(mockMultiStore, cmtproto.Header{}, false, mockLogger)

	// Expectations
	mockMultiVersionStore.EXPECT().ValidateTransactionState(0).Return(true, []int{})
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	// Run validateTask
	result := scheduler.validateTask(ctx, task)

	// Verify
	assert.True(t, result, "task should be valid")
	assert.Equal(t, statusValidatedInt, task.Status, "task should be validated")
}

func mockDeliverTx(ctx sdk.Context, tx []byte) *abci.ExecTxResult {
	//time.Sleep(500 * time.Microsecond)
	time.Sleep(1500 * time.Microsecond)
	return &abci.ExecTxResult{
		Code:      0,
		GasWanted: 1000,
		GasUsed:   800,
	}
}

func createTestContext() sdk.Context {
	//storeKey := sdk.NewKVStoreKey("test")
	////multiStore := &mockMultiStore{
	////	stores: map[store.StoreKey]store.KVStore{
	////		storeKey: &mockKVStore{},
	////	},
	////}
	//multiStore := &mocks.MockMultiStore{
	//	KVStores: map[store.StoreKey]store.KVStore{
	//		storeKey: &mocks.MockKVStore{},
	//	},
	//}
	var t *testing.T
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	multiStore := mocks.NewMockMultiStore(ctrl)
	return sdk.NewContext(multiStore, cmtproto.Header{Time: time.Now()}, false, log.NewNopLogger())
}

//
//type mockMultiStore struct {
//	stores map[store.StoreKey]store.KVStore
//}
//
//func (m *mockMultiStore) Write() {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (m *mockMultiStore) GetStoreType() store.StoreType {
//
//	//TODO implement me
//	panic("implement me")
//}
//
//func (m *mockMultiStore) CacheWrap() store.CacheWrap {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (m *mockMultiStore) CacheWrapWithTrace(w io.Writer, tc store.TraceContext) store.CacheWrap {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (m *mockMultiStore) CacheMultiStoreWithVersion(version int64) (store.CacheMultiStore, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (m *mockMultiStore) GetStore(key store.StoreKey) store.Store {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (m *mockMultiStore) TracingEnabled() bool {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (m *mockMultiStore) SetTracer(w io.Writer) store.MultiStore {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (m *mockMultiStore) SetTracingContext(context store.TraceContext) store.MultiStore {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (m *mockMultiStore) LatestVersion() int64 {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (m *mockMultiStore) GetKVStore(key store.StoreKey) store.KVStore {
//	return m.stores[key]
//}
//
//func (m *mockMultiStore) CacheMultiStore() store.CacheMultiStore {
//	return m
//}
//
//func (m *mockMultiStore) StoreKeys() []store.StoreKey {
//	keys := make([]store.StoreKey, 0, len(m.stores))
//	for k := range m.stores {
//		keys = append(keys, k)
//	}
//	return keys
//}
//
//func (m *mockMultiStore) SetKVStores(_ func(store.StoreKey, store.KVStore) store.CacheWrap) store.MultiStore {
//	return m
//}
//
//type mockKVStore struct {
//	data map[string][]byte
//}
//
//func (m *mockKVStore) GetStoreType() store.StoreType {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (m *mockKVStore) CacheWrap() store.CacheWrap {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (m *mockKVStore) CacheWrapWithTrace(w io.Writer, tc store.TraceContext) store.CacheWrap {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (m *mockKVStore) Iterator(start, end []byte) store.Iterator {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (m *mockKVStore) ReverseIterator(start, end []byte) store.Iterator {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (m *mockKVStore) Get(key []byte) []byte {
//	if m.data == nil {
//		return nil
//	}
//	return m.data[string(key)]
//}
//
//func (m *mockKVStore) Set(key, value []byte) {
//	if m.data == nil {
//		m.data = make(map[string][]byte)
//	}
//	m.data[string(key)] = value
//}
//
//func (m *mockKVStore) Has(key []byte) bool {
//	if m.data == nil {
//		return false
//	}
//	_, ok := m.data[string(key)]
//	return ok
//}
//
//func (m *mockKVStore) Delete(key []byte) {
//	if m.data != nil {
//		delete(m.data, string(key))
//	}
//}

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
