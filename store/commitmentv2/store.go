package commitmentv2

import (
	"bytes"
	"fmt"
	"io"

	"cosmossdk.io/errors"
	"cosmossdk.io/log"
	"cosmossdk.io/store/cachekv"
	"cosmossdk.io/store/internal/kv"
	"cosmossdk.io/store/listenkv"
	pruningtypes "cosmossdk.io/store/pruning/types"
	"cosmossdk.io/store/tracekv"
	"cosmossdk.io/store/types"
	"github.com/cometbft/cometbft/proto/tendermint/crypto"
	"github.com/cosmos/iavl"
	sctypes "github.com/eni-chain/eni-db/sc/types"
)

var (
	_ types.CommitKVStore = (*Store)(nil)
	_ types.Queryable     = (*Store)(nil)
)

const DefaultCacheSizeLimit = 4000000 // TODO: revert back to 1000000 after paritioning r/w caches
//var ErrInvalidHeight = errors.Register(RootCodespace, 26, "invalid height")
//var ErrUnknownRequest = errors.Register(RootCodespace, 6, "unknown request")

//const RootCodespace = "sdk"

// Store Implements types.KVStore and CommitKVStore.
type Store struct {
	tree      sctypes.Tree
	logger    log.Logger
	changeSet iavl.ChangeSet
}

func NewStore(tree sctypes.Tree, logger log.Logger) *Store {
	return &Store{
		tree:   tree,
		logger: logger,
	}
}

func (st *Store) Commit() types.CommitID {
	panic("memiavl store is not supposed to be committed alone")
}

func (st *Store) LastCommitID() types.CommitID {
	hash := st.tree.RootHash()
	return types.CommitID{
		Version: st.tree.Version(),
		Hash:    hash,
	}
}

// SetPruning panics as pruning options should be provided at initialization
// since IAVl accepts pruning options directly.
func (st *Store) SetPruning(_ pruningtypes.PruningOptions) {
	panic("cannot set pruning options on an initialized IAVL store")
}

// SetPruning panics as pruning options should be provided at initialization
// since IAVl accepts pruning options directly.
func (st *Store) GetPruning() pruningtypes.PruningOptions {
	panic("cannot get pruning options on an initialized IAVL store")
}

func (st *Store) WorkingHash() []byte {
	panic("not implemented")
}

// Implements Store.
func (st *Store) GetStoreType() types.StoreType {
	return types.StoreTypeIAVL
}

func (st *Store) CacheWrap() types.CacheWrap {
	return cachekv.NewStore(st)
}

// CacheWrapWithTrace implements the Store interface.
func (st *Store) CacheWrapWithTrace(w io.Writer, tc types.TraceContext) types.CacheWrap {
	return cachekv.NewStore(tracekv.NewStore(st, w, tc))
}

func (st *Store) CacheWrapWithListeners(storeKey types.StoreKey, listeners *types.MemoryListener) types.CacheWrap {
	return cachekv.NewStore(listenkv.NewStore(st, storeKey, listeners))
}

// Implements types.KVStore.
//
// we assume Set is only called in `Commit`, so the written state is only visible after commit.
func (st *Store) Set(key, value []byte) {
	st.changeSet.Pairs = append(st.changeSet.Pairs, &iavl.KVPair{
		Key: key, Value: value,
	})
}

// Implements types.KVStore.
func (st *Store) Get(key []byte) []byte {
	//query changeSet first
	for _, p := range st.changeSet.Pairs {
		if bytes.Equal(p.Key, key) {
			if p.Delete {
				return nil
			}
			return p.Value
		}
	}
	return st.tree.Get(key)
}

// Implements types.KVStore.
func (st *Store) Has(key []byte) bool {
	//query changeSet first
	for _, p := range st.changeSet.Pairs {
		if bytes.Equal(p.Key, key) {
			return !p.Delete
		}
	}
	return st.tree.Has(key)
}

// Implements types.KVStore.
//
// we assume Delete is only called in `Commit`, so the written state is only visible after commit.
func (st *Store) Delete(key []byte) {
	st.changeSet.Pairs = append(st.changeSet.Pairs, &iavl.KVPair{
		Key: key, Delete: true,
	})
}

func (st *Store) Iterator(start, end []byte) types.Iterator {
	return st.tree.Iterator(start, end, true)
}

func (st *Store) ReverseIterator(start, end []byte) types.Iterator {
	return st.tree.Iterator(start, end, false)
}

// SetInitialVersion sets the initial version of the IAVL tree. It is used when
// starting a new chain at an arbitrary height.
// implements interface StoreWithInitialVersion
func (st *Store) SetInitialVersion(_ int64) {
	panic("memiavl store's SetInitialVersion is not supposed to be called directly")
}

// PopChangeSet returns the change set and clear it
func (st *Store) PopChangeSet() iavl.ChangeSet {
	cs := st.changeSet
	st.changeSet = iavl.ChangeSet{}
	return cs
}

func (st *Store) Query(req *types.RequestQuery) (res *types.ResponseQuery, err error) {
	if req.Height > 0 && req.Height != st.tree.Version() {
		return QueryResult(fmt.Errorf("invalid height"), false), nil
	}
	res.Height = st.tree.Version()

	switch req.Path {
	case "/key": // get by key
		res.Key = req.Data // data holds the key bytes
		res.Value = st.tree.Get(res.Key)
		if !req.Prove {
			break
		}

		// get proof from tree and convert to merkle.Proof before adding to result
		commitmentProof := st.tree.GetProof(res.Key)
		op := types.NewIavlCommitmentOp(res.Key, commitmentProof)
		res.ProofOps = &crypto.ProofOps{Ops: []crypto.ProofOp{op.ProofOp()}}
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
		return QueryResult(fmt.Errorf("unexpected query path: %v", req.Path), false), nil
	}

	return res, nil
}

func (st *Store) VersionExists(version int64) bool {
	// one version per SC tree
	return version == st.tree.Version()
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

func (st *Store) GetChangedPairs(prefix []byte) (res []*iavl.KVPair) {
	// not sure if we can assume pairs are sorted or not, so be conservative
	// here and iterate through everything
	for _, p := range st.changeSet.Pairs {
		if bytes.HasPrefix(p.Key, prefix) {
			res = append(res, p)
		}
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
