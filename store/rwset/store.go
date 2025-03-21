package rwset

import (
	"bytes"
	"sort"
	"sync"

	"cosmossdk.io/store/types"
	db "github.com/cometbft/cometbft-db"
)

// RwSetStore 交易级别可见的store结构，用于读写集冲突检测及commit前的key、value存储
type RwSetStore interface {
	GetLatest(key []byte) (value WriteSetValueItem)
	GetLatestBeforeIndex(index int, key []byte) (value WriteSetValueItem)
	Has(index int, key []byte) bool
	WriteLatestToStore()
	SetWriteSet(index int, writeSet WriteSet)
	InvalidateWriteSet(index int)
	GetAllWriteSetKeys() map[int][]string
	CollectIteratorItems(index int) *db.MemDB
	SetReadSet(index int, readSet ReadSet)
	GetReadSet(index int) ReadSet
	ClearReadSet(index int)
	TxExecutionStore(index int) *TxExecutionStore
	ValidateTransactionState(index int, isSync bool) (bool, []int)
}

type WriteSet map[string][]byte
type ReadSet map[string][][]byte

var _ RwSetStore = (*Store)(nil)

type Store struct {
	// key (write key) -> value (WriteSet)
	writeSetMap *sync.Map

	txWriteSetKeys *sync.Map // map of tx index -> writeSet keys []string
	txReadSets     *sync.Map // map of tx index -> readSet ReadSet

	parentStore types.KVStore
}

func NewRwSetStore(parentStore types.KVStore) *Store {
	return &Store{
		writeSetMap:    &sync.Map{},
		txWriteSetKeys: &sync.Map{},
		txReadSets:     &sync.Map{},
		parentStore:    parentStore,
	}
}

// TxExecutionStore creates a new versioned index store for a given incarnation and transaction index
func (s *Store) TxExecutionStore(index int) *TxExecutionStore {
	return NewTxExecutionStore(s.parentStore, s, index)
}

// GetLatest implements RwSetStore.
func (s *Store) GetLatest(key []byte) (value WriteSetValueItem) {
	keyString := string(key)
	mvVal, found := s.writeSetMap.Load(keyString)
	// if the key doesn't exist in the overall map, return nil
	if !found {
		return nil
	}
	latestVal, found := mvVal.(WriteSetValue).GetLatest()
	if !found {
		return nil // this is possible IF there is are writeSet that are then removed for that key
	}
	return latestVal
}

// GetLatestBeforeIndex implements RwSetStore.
func (s *Store) GetLatestBeforeIndex(index int, key []byte) (value WriteSetValueItem) {
	keyString := string(key)
	mvVal, found := s.writeSetMap.Load(keyString)
	// if the key doesn't exist in the overall map, return nil
	if !found {
		return nil
	}
	val, found := mvVal.(WriteSetValue).GetLatestBeforeIndex(index)
	// otherwise, we may have found a value for that key, but its not written before the index passed in
	if !found {
		return nil
	}
	// found a value prior to the passed in index, return that value (could be estimate OR deleted, but it is a definitive value)
	return val
}

// Has implements RwSetStore. It checks if the key exists in the multiversion store at or before the specified index.
func (s *Store) Has(index int, key []byte) bool {

	keyString := string(key)
	mvVal, found := s.writeSetMap.Load(keyString)
	// if the key doesn't exist in the overall map, return nil
	if !found {
		return false // this is okay because the caller of this will THEN need to access the parent store to verify that the key doesnt exist there
	}
	_, foundVal := mvVal.(WriteSetValue).GetLatestBeforeIndex(index)
	return foundVal
}

func (s *Store) removeOldWriteSet(index int, newWriteSet WriteSet) {
	writeSet := make(map[string][]byte)
	if newWriteSet != nil {
		// if non-nil writeSet passed in, we can use that to optimize removals
		writeSet = newWriteSet
	}
	// if there is already a writeSet existing, we should remove that fully
	oldKeys, loaded := s.txWriteSetKeys.LoadAndDelete(index)
	if loaded {
		keys := oldKeys.([]string)
		// we need to delete all of the keys in the writeSet from the multiversion store
		for _, key := range keys {
			// small optimization to check if the new writeSet is going to write this key, if so, we can leave it behind
			if _, ok := writeSet[key]; ok {
				// we don't need to remove this key because it will be overwritten anyways - saves the operation of removing + rebalancing underlying btree
				continue
			}
			// remove from the appropriate item if present in writeSetMap
			mvVal, found := s.writeSetMap.Load(key)
			// if the key doesn't exist in the overall map, return nil
			if !found {
				continue
			}
			mvVal.(WriteSetValue).Remove(index)
		}
	}
}

// SetWriteSet sets a writeSet for a transaction index.
func (s *Store) SetWriteSet(index int, writeSet WriteSet) {
	s.removeOldWriteSet(index, writeSet)

	writeSetKeys := make([]string, 0, len(writeSet))
	for key, value := range writeSet {
		writeSetKeys = append(writeSetKeys, key)
		loadVal, _ := s.writeSetMap.LoadOrStore(key, NewWriteSetItem()) // init if necessary
		mvVal := loadVal.(WriteSetValue)
		if value == nil {
			mvVal.Delete(index)
		} else {
			mvVal.Set(index, value)
		}
	}
	sort.Strings(writeSetKeys)
	s.txWriteSetKeys.Store(index, writeSetKeys)
}

// InvalidateWriteSet iterates over the keys for the given index and incarnation writeSet and replaces with ESTIMATEs
func (s *Store) InvalidateWriteSet(index int) {
	keysAny, found := s.txWriteSetKeys.Load(index)
	if !found {
		return
	}
	keys := keysAny.([]string)
	for _, key := range keys {
		// invalidate all of the writeSet items - is this suboptimal? - we could potentially do concurrently if slow because locking is on an item specific level
		_, _ = s.writeSetMap.LoadOrStore(key, NewWriteSetItem())
	}
	// we leave the writeSet in place because we'll need it for key removal later if/when we replace with a new writeSet
}

// GetAllWriteSetKeys implements RwSetStore.
func (s *Store) GetAllWriteSetKeys() map[int][]string {
	writeSetKeys := make(map[int][]string)
	s.txWriteSetKeys.Range(func(key, value interface{}) bool {
		index := key.(int)
		keys := value.([]string)
		writeSetKeys[index] = keys
		return true
	})

	return writeSetKeys
}

func (s *Store) SetReadSet(index int, readSet ReadSet) {
	s.txReadSets.Store(index, readSet)
}

func (s *Store) GetReadSet(index int) ReadSet {
	readSetAny, found := s.txReadSets.Load(index)
	if !found {
		return nil
	}
	return readSetAny.(ReadSet)
}

func (s *Store) ClearReadSet(index int) {
	s.txReadSets.Delete(index)
}

// CollectIteratorItems implements RwSetStore. It will return a memDB containing all of the keys present in the multiversion store within the iteration range prior to (exclusive of) the index.
func (s *Store) CollectIteratorItems(index int) *db.MemDB {
	sortedItems := db.NewMemDB()

	// get all writeSet keys prior to index
	for i := 0; i < index; i++ {
		writeSetAny, found := s.txWriteSetKeys.Load(i)
		if !found {
			continue
		}
		indexedWriteSet := writeSetAny.([]string)
		for _, key := range indexedWriteSet {
			sortedItems.Set([]byte(key), []byte{})
		}
	}
	return sortedItems
}

func (s *Store) checkReadSetAtIndex(index int, isSync bool) (bool, []int) {
	conflictSet := make(map[int]struct{})
	valid := true

	readSetAny, found := s.txReadSets.Load(index)
	if !found {
		return true, []int{}
	}
	readSet := readSetAny.(ReadSet)
	for key, valueArr := range readSet {
		if len(valueArr) != 1 {
			valid = false
			continue
		}
		value := valueArr[0]
		latestValue := s.GetLatestBeforeIndex(index, []byte(key))
		if latestValue == nil {
			parentVal := s.parentStore.Get([]byte(key))
			if !bytes.Equal(parentVal, value) {
				valid = false
			}
		} else {
			if isSync {
				if !bytes.Equal(value, latestValue.Value()) {
					conflictSet[latestValue.Index()] = struct{}{}
					valid = false
				} else if latestValue.IsDeleted() {
					if value != nil {
						conflictSet[latestValue.Index()] = struct{}{}
						valid = false
					}
				}
			} else {
				valid = false
				conflictSet[latestValue.Index()] = struct{}{}
			}
		}
	}

	conflictIndices := make([]int, 0, len(conflictSet))
	for index := range conflictSet {
		conflictIndices = append(conflictIndices, index)
	}

	sort.Ints(conflictIndices)

	return valid, conflictIndices
}

func (s *Store) checkWriteSetAtIndex(index int) (bool, []int) {
	conflictSet := make(map[int]struct{})
	valid := true

	writeSetAny, found := s.txWriteSetKeys.Load(index)
	if !found {
		return true, []int{}
	}
	writeSet := writeSetAny.([]string)

	for _, key := range writeSet {
		latestValue := s.GetLatestBeforeIndex(index, []byte(key))
		if latestValue != nil {
			valid = false
			conflictSet[latestValue.Index()] = struct{}{}
		}
	}

	conflictIndices := make([]int, 0, len(conflictSet))
	for id := range conflictSet {
		conflictIndices = append(conflictIndices, id)
	}

	sort.Ints(conflictIndices)

	return valid, conflictIndices
}

func (s *Store) ValidateTransactionState(index int, isSync bool) (bool, []int) {
	readSetValid, readSetConflicts := s.checkReadSetAtIndex(index, isSync)
	if !isSync {
		writeSetValid, writeSetConflicts := s.checkWriteSetAtIndex(index)
		return writeSetValid && readSetValid, append(writeSetConflicts, readSetConflicts...)
	}

	return readSetValid, readSetConflicts
}

func (s *Store) WriteLatestToStore() {
	// sort the keys
	keys := []string{}
	s.writeSetMap.Range(func(key, value interface{}) bool {
		keys = append(keys, key.(string))
		return true
	})
	sort.Strings(keys)

	for _, key := range keys {
		val, ok := s.writeSetMap.Load(key)
		if !ok {
			continue
		}
		mvValue, found := val.(WriteSetValue).GetLatestValue()
		if !found {
			// this means that at some point, there was an estimate, but we have since removed it so there isn't anything writeable at the key, so we can skip
			continue
		}
		// if the value is deleted, then delete it from the parent store
		if mvValue.IsDeleted() {
			// We use []byte(key) instead of conv.UnsafeStrToBytes because we cannot
			// be sure if the underlying store might do a save with the byteslice or
			// not. Once we get confirmation that .Delete is guaranteed not to
			// save the byteslice, then we can assume only a read-only copy is sufficient.
			s.parentStore.Delete([]byte(key))
			continue
		}
		if mvValue.Value() != nil {
			s.parentStore.Set([]byte(key), mvValue.Value())
		}
	}
}
