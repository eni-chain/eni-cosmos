package statev2

import (
	"fmt"
	"io"

	"cosmossdk.io/errors"
	"cosmossdk.io/store/internal/kv"

	"cosmossdk.io/store/cachekv"
	"cosmossdk.io/store/listenkv"
	"cosmossdk.io/store/tracekv"
	"cosmossdk.io/store/types"
	sstypes "github.com/eni-chain/eni-db/ss/types"
)

const StoreTypeSSStore = 100

var ErrInvalidHeight = errors.Register(RootCodespace, 26, "invalid height")
var ErrUnknownRequest = errors.Register(RootCodespace, 6, "unknown request")

const RootCodespace = "sdk"

var (
	_ types.KVStore   = (*Store)(nil)
	_ types.Queryable = (*Store)(nil)
)

// Store wraps a SS store and implements a cosmos KVStore
type Store struct {
	store    sstypes.StateStore
	storeKey types.StoreKey
	version  int64
}

func NewStore(store sstypes.StateStore, storeKey types.StoreKey, version int64) *Store {
	return &Store{store, storeKey, version}
}

func (st *Store) GetStoreType() types.StoreType {
	return StoreTypeSSStore
}

func (st *Store) CacheWrap() types.CacheWrap {
	return cachekv.NewStore(st)
}

func (st *Store) CacheWrapWithTrace(w io.Writer, tc types.TraceContext) types.CacheWrap {
	return cachekv.NewStore(tracekv.NewStore(st, w, tc))
}

func (st *Store) CacheWrapWithListeners(storeKey types.StoreKey, listeners *types.MemoryListener) types.CacheWrap {
	return cachekv.NewStore(listenkv.NewStore(st, storeKey, listeners))
}

func (st *Store) Get(key []byte) []byte {
	value, err := st.store.Get(st.storeKey.Name(), st.version, key)
	if err != nil {
		panic(err)
	}
	return value
}

func (st *Store) Has(key []byte) bool {
	has, err := st.store.Has(st.storeKey.Name(), st.version, key)
	if err != nil {
		panic(err)
	}
	return has
}

func (st *Store) Set(_, _ []byte) {
	panic("write operation is not supported")
}

func (st *Store) Delete(_ []byte) {
	panic("write operation is not supported")
}

func (st *Store) Iterator(start, end []byte) types.Iterator {
	itr, err := st.store.Iterator(st.storeKey.Name(), st.version, start, end)
	if err != nil {
		panic(err)
	}
	return itr
}

func (st *Store) ReverseIterator(start, end []byte) types.Iterator {
	itr, err := st.store.ReverseIterator(st.storeKey.Name(), st.version, start, end)
	if err != nil {
		panic(err)
	}
	return itr
}

func (st *Store) GetWorkingHash() ([]byte, error) {
	panic("get working hash operation is not supported")
}

func (st *Store) Query(req *types.RequestQuery) (res *types.ResponseQuery, err error) {
	if req.Height > 0 && req.Height > st.version {
		return QueryResult(ErrInvalidHeight.Wrap("invalid height"), false), nil
	}
	res.Height = st.version
	switch req.Path {
	case "/key": // get by key
		res.Key = req.Data // data holds the key bytes
		res.Value = st.Get(res.Key)
	case "/subspace":
		pairs := kv.Pairs{
			Pairs: make([]kv.Pair, 0),
		}
		subspace := req.Data
		res.Key = subspace
		iterator := types.KVStorePrefixIterator(st, subspace)
		for ; iterator.Valid(); iterator.Next() {
			pairs.Pairs = append(pairs.Pairs, kv.Pair{Key: iterator.Key(), Value: iterator.Value()})
		}
		iterator.Close()

		bz, err := pairs.Marshal()
		if err != nil {
			panic(fmt.Errorf("failed to marshal KV pairs: %w", err))
		}
		res.Value = bz
	default:
		return QueryResult(ErrUnknownRequest.Wrapf("unexpected query path: %v", req.Path), false), nil
	}

	return res, nil
}

func (st *Store) VersionExists(version int64) bool {
	earliest, err := st.store.GetEarliestVersion()
	if err != nil {
		panic(err)
	}
	return version >= earliest
}

func (st *Store) DeleteAll(start, end []byte) error {
	iter := st.Iterator(start, end)
	keys := [][]byte{}
	for ; iter.Valid(); iter.Next() {
		keys = append(keys, iter.Key())
	}
	iter.Close()
	for _, key := range keys {
		st.Delete(key)
	}
	return nil
}

func (st *Store) GetAllKeyStrsInRange(start, end []byte) (res []string) {
	iter := st.Iterator(start, end)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		res = append(res, string(iter.Key()))
	}
	return
}
func QueryResult(err error, debug bool) *types.ResponseQuery {
	space, code, log := errors.ABCIInfo(err, debug)
	return &types.ResponseQuery{
		Codespace: space,
		Code:      code,
		Log:       log,
	}
}
