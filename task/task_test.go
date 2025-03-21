package tasks

import (
	"context"
	"cosmossdk.io/log"
	"cosmossdk.io/store/cachekv"
	"cosmossdk.io/store/cachemulti"
	"cosmossdk.io/store/dbadapter"
	"cosmossdk.io/store/types"
	"errors"
	"fmt"
	abci "github.com/cometbft/cometbft/abci/types"
	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"math/rand"
	"net/http"
	"runtime"
	"testing"
	"time"
)

type mockDeliverTxFunc func(ctx sdk.Context, tx []byte) *abci.ExecTxResult

var testStoreKey = types.NewKVStoreKey("mock")
var itemKey = []byte("key")

func requestList(otherNum, seqNum int) sdk.DeliverTxBatchRequest {
	otherTasks := make([]*sdk.DeliverTxEntry, 0)
	for i := 0; i < otherNum; i++ {
		otherTasks = append(otherTasks, &sdk.DeliverTxEntry{
			Tx:      []byte(fmt.Sprintf("%d", i)),
			TxIndex: i,
		})
	}

	seqTasks := make([]*sdk.DeliverTxEntry, 0)
	for i := 0; i < seqNum; i++ {
		seqTasks = append(seqTasks, &sdk.DeliverTxEntry{
			Tx:      []byte(fmt.Sprintf("%d", i+otherNum)),
			TxIndex: i + otherNum,
		})
	}

	return sdk.DeliverTxBatchRequest{
		OtherEntries: otherTasks,
		SeqEntries:   seqTasks,
	}
}

func initTestCtx(injectStores bool) sdk.Context {
	ctx := sdk.Context{}.WithContext(context.Background())
	keys := make(map[string]types.StoreKey)
	stores := make(map[types.StoreKey]types.CacheWrapper)
	db := dbm.NewMemDB()
	if injectStores {
		mem := dbadapter.Store{DB: db}
		stores[testStoreKey] = cachekv.NewStore(mem)
		keys[testStoreKey.Name()] = testStoreKey
	}
	store := cachemulti.NewStore(db, stores, keys, nil, nil)
	ctx = ctx.WithMultiStore(&store)
	ctx = ctx.WithLogger(log.NewNopLogger())
	return ctx
}

func TestProcessAll(t *testing.T) {
	runtime.SetBlockProfileRate(1)

	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()

	tests := []struct {
		name          string
		runs          int
		req           sdk.DeliverTxBatchRequest
		deliverTxFunc mockDeliverTxFunc
		addStores     bool
		expectedErr   error
		assertions    func(t *testing.T, ctx sdk.Context, res []*abci.ExecTxResult)
	}{
		{
			name:      "Test zero txs does not hang",
			runs:      10,
			addStores: true,
			req:       requestList(0, 0),
			deliverTxFunc: func(ctx sdk.Context, tx []byte) *abci.ExecTxResult {
				panic("should not deliver")
			},
			assertions: func(t *testing.T, ctx sdk.Context, res []*abci.ExecTxResult) {
				require.Len(t, res, 0)
			},
			expectedErr: nil,
		},
		{
			name:      "Test no overlap txs",
			runs:      10,
			addStores: true,
			req:       requestList(1000, 0),
			deliverTxFunc: func(ctx sdk.Context, tx []byte) *abci.ExecTxResult {

				// all txs read and write to the same key to maximize conflicts
				kv := ctx.MultiStore().GetKVStore(testStoreKey)

				// write to the store with this tx's index
				kv.Set(tx, tx)
				val := kv.Get(tx)

				// return what was read from the store (final attempt should be index-1)
				return &abci.ExecTxResult{
					Info: string(val),
				}
			},
			assertions: func(t *testing.T, ctx sdk.Context, res []*abci.ExecTxResult) {
				for idx, response := range res {
					require.Equal(t, fmt.Sprintf("%d", idx), response.Info)
				}
				store := ctx.MultiStore().GetKVStore(testStoreKey)
				for i := 0; i < len(res); i++ {
					val := store.Get([]byte(fmt.Sprintf("%d", i)))
					require.Equal(t, []byte(fmt.Sprintf("%d", i)), val)
				}
			},
			expectedErr: nil,
		},
		{
			name:      "Test every tx accesses same key",
			runs:      1,
			addStores: true,
			req:       requestList(100, 0),
			deliverTxFunc: func(ctx sdk.Context, tx []byte) *abci.ExecTxResult {
				// all txs read and write to the same key to maximize conflicts
				kv := ctx.MultiStore().GetKVStore(testStoreKey)
				val := kv.Get(itemKey)

				// write to the store with this tx's index
				kv.Set(itemKey, tx)

				// return what was read from the store (final attempt should be index-1)
				return &abci.ExecTxResult{
					Info: string(val),
				}
			},
			assertions: func(t *testing.T, ctx sdk.Context, res []*abci.ExecTxResult) {
				for idx, response := range res {
					if idx == 0 {
						require.Equal(t, "", response.Info)
					} else {
						// the info is what was read from the kv store by the tx
						// each tx writes its own index, so the info should be the index of the previous tx
						require.Equal(t, fmt.Sprintf("%d", idx-1), response.Info)
					}
				}
				// confirm last write made it to the parent store
				latest := ctx.MultiStore().GetKVStore(testStoreKey).Get(itemKey)
				require.Equal(t, fmt.Sprintf("%d", len(res)-1), string(latest))
			},
			expectedErr: nil,
		},
		{
			name:      "Test some tx accesses same key",
			runs:      1,
			addStores: true,
			req:       requestList(2000, 0),
			deliverTxFunc: func(ctx sdk.Context, tx []byte) *abci.ExecTxResult {
				if ctx.TxIndex()%10 != 0 {
					return &abci.ExecTxResult{
						Info: "none",
					}
				}
				// all txs read and write to the same key to maximize conflicts
				kv := ctx.MultiStore().GetKVStore(testStoreKey)
				val := string(kv.Get(itemKey))

				// write to the store with this tx's index
				kv.Set(itemKey, tx)

				// return what was read from the store (final attempt should be index-1)
				return &abci.ExecTxResult{
					Info: val,
				}
			},
			assertions:  func(t *testing.T, ctx sdk.Context, res []*abci.ExecTxResult) {},
			expectedErr: nil,
		},
		{
			name:      "Test no stores on context should not panic",
			runs:      10,
			addStores: false,
			req:       requestList(10, 0),
			deliverTxFunc: func(ctx sdk.Context, tx []byte) *abci.ExecTxResult {
				return &abci.ExecTxResult{
					Info: fmt.Sprintf("%d", ctx.TxIndex()),
				}
			},
			assertions: func(t *testing.T, ctx sdk.Context, res []*abci.ExecTxResult) {
				for idx, response := range res {
					require.Equal(t, fmt.Sprintf("%d", idx), response.Info)
				}
			},
			expectedErr: nil,
		},
		{
			name:      "Test every tx accesses same key with delays",
			runs:      1,
			addStores: true,
			req:       requestList(1000, 0),
			deliverTxFunc: func(ctx sdk.Context, tx []byte) *abci.ExecTxResult {
				wait := rand.Intn(10)
				time.Sleep(time.Duration(wait) * time.Millisecond)
				// all txs read and write to the same key to maximize conflicts
				kv := ctx.MultiStore().GetKVStore(testStoreKey)
				val := string(kv.Get(itemKey))
				time.Sleep(time.Duration(wait) * time.Millisecond)
				// write to the store with this tx's index
				newVal := val + fmt.Sprintf("%d", ctx.TxIndex())
				kv.Set(itemKey, []byte(newVal))

				// return what was read from the store (final attempt should be index-1)
				return &abci.ExecTxResult{
					Info: newVal,
				}
			},
			assertions: func(t *testing.T, ctx sdk.Context, res []*abci.ExecTxResult) {
				expected := ""
				for idx, response := range res {
					expected = expected + fmt.Sprintf("%d", idx)
					require.Equal(t, expected, response.Info)
				}
				// confirm last write made it to the parent store
				latest := ctx.MultiStore().GetKVStore(testStoreKey).Get(itemKey)
				require.Equal(t, expected, string(latest))
			},
			expectedErr: nil,
		},
		{
			name:      "Test transaction conflicts with task 0",
			runs:      1,
			addStores: true,
			req:       requestList(100, 100),
			deliverTxFunc: func(ctx sdk.Context, tx []byte) *abci.ExecTxResult {
				kv := ctx.MultiStore().GetKVStore(testStoreKey)
				fmt.Println("tx is 0", string(tx))

				if string(tx) == "0" {
					kv.Set(itemKey, tx)
				} else {
					kv.Get(itemKey)
				}

				return &abci.ExecTxResult{
					Info: string(tx),
				}
			},
			assertions: func(t *testing.T, ctx sdk.Context, res []*abci.ExecTxResult) {
				expected := ""
				for idx, response := range res {
					expected = fmt.Sprintf("%d", idx)
					require.Equal(t, expected, response.Info)
				}
				latest := ctx.MultiStore().GetKVStore(testStoreKey).Get(itemKey)
				require.Equal(t, "0", string(latest))
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < tt.runs; i++ {
				// set a tracer provider
				ctx := initTestCtx(tt.addStores)
				s := NewScheduler(tt.deliverTxFunc)

				now := time.Now()
				res, err := s.ProcessAll(ctx, tt.req)
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				elapsed := time.Since(now)
				fmt.Printf("代码运行时间: %s\n", elapsed)

				require.Len(t, res, len(tt.req.OtherEntries)+len(tt.req.SeqEntries))

				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("Expected error %v, got %v", tt.expectedErr, err)
				} else {
					tt.assertions(t, ctx, res)
				}
			}
		})
	}
}
