package rwset

import (
	"bytes"
	"cosmossdk.io/store/types"
	dbm "github.com/cosmos/cosmos-db"
	io "io"
	"sort"
)

// ReadSetHandler exposes a handler for adding items to readSet, useful for iterators
type ReadSetHandler interface {
	UpdateReadSet(key []byte, value []byte)
}

type NoOpHandler struct{}

func (NoOpHandler) UpdateReadSet(key []byte, value []byte) {}

// TxExecutionStore wraps the multiversion store in a way that implements the KVStore interface, but also stores the index of the transaction, and so store actions are applied to the multiversion store using that index
type TxExecutionStore struct {
	readSet  map[string][][]byte
	writeSet map[string][]byte // contains the key -> value mapping for all keys written to the store

	sortedStore      *dbm.MemDB // always ascending sorted
	rwSetStore       RwSetStore
	parent           types.KVStore
	transactionIndex int
}

func (store *TxExecutionStore) Write() {
	//TODO implement me
	panic("implement me")
}

func (store *TxExecutionStore) CacheWrap() types.CacheWrap {
	//TODO implement me
	panic("implement me")
}

func (store *TxExecutionStore) CacheWrapWithTrace(w io.Writer, tc types.TraceContext) types.CacheWrap {
	//TODO implement me
	panic("implement me")
}

func (store *TxExecutionStore) Iterator(start, end []byte) types.Iterator {
	panic("version indexed store does not support iterators")
}

func (store *TxExecutionStore) ReverseIterator(start, end []byte) types.Iterator {
	panic("version indexed store does not support reverse iterators")
}

func (store *TxExecutionStore) DeleteAll(start, end []byte) error {
	panic("version indexed store does not support delete all")
}

func (store *TxExecutionStore) GetAllKeyStrsInRange(start, end []byte) []string {
	panic("version indexed store does not support get all key strs in range")
}

var _ types.KVStore = (*TxExecutionStore)(nil)
var _ ReadSetHandler = (*TxExecutionStore)(nil)

func NewTxExecutionStore(parent types.KVStore, rwSetStore RwSetStore, transactionIndex int) *TxExecutionStore {
	return &TxExecutionStore{
		readSet:          make(map[string][][]byte),
		writeSet:         make(map[string][]byte),
		sortedStore:      dbm.NewMemDB(),
		parent:           parent,
		rwSetStore:       rwSetStore,
		transactionIndex: transactionIndex,
	}
}

// GetReadSet returns the readSet
func (store *TxExecutionStore) GetReadSet() map[string][][]byte {
	return store.readSet
}

// GetWriteSet returns the writeSet
func (store *TxExecutionStore) GetWriteSet() map[string][]byte {
	return store.writeSet
}

// Get implements types.KVStore.
func (store *TxExecutionStore) Get(key []byte) []byte {
	types.AssertValidKey(key)
	strKey := string(key)
	// first check the MVKV writeSet, and return that value if present
	cacheValue, ok := store.writeSet[strKey]
	if ok {
		// return the value from the cache, no need to update any readSet stuff
		return cacheValue
	}
	// read the readSet to see if the value exists - and return if applicable
	if readSetVal, ok := store.readSet[strKey]; ok {
		// just return the first one, if there is more than one, we will fail the validation anyways
		return readSetVal[0]
	}

	mvsValue := store.rwSetStore.GetLatestBeforeIndex(store.transactionIndex, key)
	if mvsValue != nil {
		// This handles both detecting readSet conflicts and updating readSet if applicable
		return store.parseValueAndUpdateReadSet(strKey, mvsValue)
	}
	parentValue := store.parent.Get(key)
	store.UpdateReadSet(key, parentValue)
	return parentValue
}

// This functions handles reads with deleted items and values and verifies that the data is consistent to what we currently have in the readSet (IF we have a readSet value for that key)
func (store *TxExecutionStore) parseValueAndUpdateReadSet(strKey string, mvsValue WriteSetValueItem) []byte {
	value := mvsValue.Value()
	if mvsValue.IsDeleted() {
		value = nil
	}
	store.UpdateReadSet([]byte(strKey), value)
	return value
}

// ValidateReadSet This function iterates over the readSet
func (store *TxExecutionStore) ValidateReadSet() bool {
	readSetKeys := make([]string, 0, len(store.readSet))
	for key := range store.readSet {
		readSetKeys = append(readSetKeys, key)
	}
	sort.Strings(readSetKeys)

	// iterate over readSet keys and values
	for _, strKey := range readSetKeys {
		key := []byte(strKey)
		valueArr := store.readSet[strKey]
		if len(valueArr) != 1 {
			// if we have more than one value, we will fail the validation since we dedup when adding to readSet
			return false
		}
		value := valueArr[0]
		mvsValue := store.rwSetStore.GetLatestBeforeIndex(store.transactionIndex, key)
		if mvsValue != nil {
			if mvsValue.IsDeleted() {
				// check for `nil`
				if value != nil {
					return false
				}
			} else {
				// check for equality
				if string(value) != string(mvsValue.Value()) {
					return false
				}
			}

			continue // value is valid, continue to next key
		}

		parentValue := store.parent.Get(key)
		if string(parentValue) != string(value) {
			panic("we shouldn't ever have a readSet conflict in parent store")
		}
		// value was correct, we can continue to the next value
	}
	return true
}

// Delete implements types.KVStore.
func (store *TxExecutionStore) Delete(key []byte) {
	types.AssertValidKey(key)
	store.setValue(key, nil)
}

// Has implements types.KVStore.
func (store *TxExecutionStore) Has(key []byte) bool {
	// necessary locking happens within store.Get
	return store.Get(key) != nil
}

// Set implements types.KVStore.
func (store *TxExecutionStore) Set(key []byte, value []byte) {
	types.AssertValidKey(key)
	store.setValue(key, value)
}

// GetStoreType implements types.KVStore.
func (store *TxExecutionStore) GetStoreType() types.StoreType {
	return store.parent.GetStoreType()
}

// Only entrypoint to mutate writeSet
func (store *TxExecutionStore) setValue(key, value []byte) {
	types.AssertValidKey(key)

	keyStr := string(key)
	store.writeSet[keyStr] = value
}

func (store *TxExecutionStore) WriteToRwSetStore() {
	store.rwSetStore.SetWriteSet(store.transactionIndex, store.writeSet)
	store.rwSetStore.SetReadSet(store.transactionIndex, store.readSet)
}

func (store *TxExecutionStore) UpdateReadSet(key []byte, value []byte) {
	keyStr := string(key)
	if _, ok := store.readSet[keyStr]; !ok {
		store.readSet[keyStr] = [][]byte{}
	}
	for _, readSetVal := range store.readSet[keyStr] {
		if bytes.Equal(value, readSetVal) {
			return
		}
	}
	store.readSet[keyStr] = append(store.readSet[keyStr], value)
}
