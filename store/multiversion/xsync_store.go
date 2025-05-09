package multiversion

import (
	"bytes"
	"cosmossdk.io/store/multiversion/occ"
	"github.com/puzpuzpuz/xsync/v4"

	"sort"

	"cosmossdk.io/store/types"
	db "github.com/cometbft/cometbft-db"
)

var _ MultiVersionStore = (*XSyncStore)(nil)

type XSyncStore struct {
	// map that stores the key string -> MultiVersionValue mapping for accessing from a given key
	multiVersionMap *xsync.Map[string, *multiVersionItem]
	// TODO: do we need to support iterators as well similar to how cachekv does it - yes

	txWritesetKeys *xsync.Map[int, []string]   // map of tx index -> writeset keys []string
	txReadSets     *xsync.Map[int, ReadSet]    // map of tx index -> readset ReadSet
	txIterateSets  *xsync.Map[int, Iterateset] // map of tx index -> iterateset Iterateset

	parentStore types.KVStore
}

func NewMultiXSyncVersionStore(parentStore types.KVStore, txNum int) *XSyncStore {
	multiVersionMap := xsync.NewMap[string, *multiVersionItem](xsync.WithPresize(txNum * 5))
	txWritesetKeys := xsync.NewMap[int, []string](xsync.WithPresize(txNum + 100))
	txReadSets := xsync.NewMap[int, ReadSet](xsync.WithPresize(txNum + 100))
	txIterateSets := xsync.NewMap[int, Iterateset]()
	return &XSyncStore{
		multiVersionMap: multiVersionMap,
		txWritesetKeys:  txWritesetKeys,
		txReadSets:      txReadSets,
		txIterateSets:   txIterateSets,
		parentStore:     parentStore,
	}
}

// VersionedIndexedStore creates a new versioned index store for a given incarnation and transaction index
func (s *XSyncStore) VersionedIndexedStore(index int, incarnation int, abortChannel chan occ.Abort) *VersionIndexedStore {
	return NewVersionIndexedStore(s.parentStore, s, index, incarnation, abortChannel)
}

// GetLatest implements MultiVersionStore.
func (s *XSyncStore) GetLatest(key []byte) (value MultiVersionValueItem) {
	keyString := string(key)
	mvVal, found := s.multiVersionMap.Load(keyString)
	// if the key doesn't exist in the overall map, return nil
	if !found {
		return nil
	}
	latestVal, found := mvVal.GetLatest()
	if !found {
		return nil // this is possible IF there is are writeset that are then removed for that key
	}
	return latestVal
}

// GetLatestBeforeIndex implements MultiVersionStore.
func (s *XSyncStore) GetLatestBeforeIndex(index int, key []byte) (value MultiVersionValueItem) {
	keyString := string(key)
	mvVal, found := s.multiVersionMap.Load(keyString)
	// if the key doesn't exist in the overall map, return nil
	if !found {
		return nil
	}
	val, found := mvVal.GetLatestBeforeIndex(index)
	// otherwise, we may have found a value for that key, but its not written before the index passed in
	if !found {
		return nil
	}
	// found a value prior to the passed in index, return that value (could be estimate OR deleted, but it is a definitive value)
	return val
}

// Has implements MultiVersionStore. It checks if the key exists in the multiversion store at or before the specified index.
func (s *XSyncStore) Has(index int, key []byte) bool {

	keyString := string(key)
	mvVal, found := s.multiVersionMap.Load(keyString)
	// if the key doesn't exist in the overall map, return nil
	if !found {
		return false // this is okay because the caller of this will THEN need to access the parent store to verify that the key doesnt exist there
	}
	_, foundVal := mvVal.GetLatestBeforeIndex(index)
	return foundVal
}

func (s *XSyncStore) removeOldWriteset(index int, newWriteSet WriteSet) {
	writeset := make(map[string][]byte)
	if newWriteSet != nil {
		// if non-nil writeset passed in, we can use that to optimize removals
		writeset = newWriteSet
	}
	// if there is already a writeset existing, we should remove that fully
	keys, loaded := s.txWritesetKeys.LoadAndDelete(index)
	if loaded {
		// we need to delete all of the keys in the writeset from the multiversion store
		for _, key := range keys {
			// small optimization to check if the new writeset is going to write this key, if so, we can leave it behind
			if _, ok := writeset[key]; ok {
				// we don't need to remove this key because it will be overwritten anyways - saves the operation of removing + rebalancing underlying btree
				continue
			}
			// remove from the appropriate item if present in multiVersionMap
			mvVal, found := s.multiVersionMap.Load(key)
			// if the key doesn't exist in the overall map, return nil
			if !found {
				continue
			}
			mvVal.Remove(index)
		}
	}
}

// SetWriteset sets a writeset for a transaction index, and also writes all of the multiversion items in the writeset to the multiversion store.
// TODO: returns a list of NEW keys added
func (s *XSyncStore) SetWriteset(index int, incarnation int, writeset WriteSet) {
	// TODO: add telemetry spans
	// remove old writeset if it exists
	s.removeOldWriteset(index, writeset)

	writeSetKeys := make([]string, 0, len(writeset))
	for key, value := range writeset {
		writeSetKeys = append(writeSetKeys, key)
		loadVal, _ := s.multiVersionMap.LoadOrStore(key, NewMultiVersionItem()) // init if necessary
		//mvVal := loadVal.(MultiVersionValue)
		if value == nil {
			// delete if nil value
			// TODO: sync map
			loadVal.Delete(index, incarnation)
		} else {
			loadVal.Set(index, incarnation, value)
		}
	}
	sort.Strings(writeSetKeys) // TODO: if we're sorting here anyways, maybe we just put it into a btree instead of a slice
	s.txWritesetKeys.Store(index, writeSetKeys)
}

// InvalidateWriteset iterates over the keys for the given index and incarnation writeset and replaces with ESTIMATEs
func (s *XSyncStore) InvalidateWriteset(index int, incarnation int) {
	keys, found := s.txWritesetKeys.Load(index)
	if !found {
		return
	}
	//keys := keysAny.([]string)
	for _, key := range keys {
		// invalidate all of the writeset items - is this suboptimal? - we could potentially do concurrently if slow because locking is on an item specific level
		val, _ := s.multiVersionMap.LoadOrStore(key, NewMultiVersionItem())
		val.SetEstimate(index, incarnation)
	}
	// we leave the writeset in place because we'll need it for key removal later if/when we replace with a new writeset
}

// SetEstimatedWriteset is used to directly write estimates instead of writing a writeset and later invalidating
func (s *XSyncStore) SetEstimatedWriteset(index int, incarnation int, writeset WriteSet) {
	// remove old writeset if it exists
	s.removeOldWriteset(index, writeset)

	writeSetKeys := make([]string, 0, len(writeset))
	// still need to save the writeset so we can remove the elements later:
	for key := range writeset {
		writeSetKeys = append(writeSetKeys, key)

		mvVal, _ := s.multiVersionMap.LoadOrStore(key, NewMultiVersionItem()) // init if necessary
		mvVal.SetEstimate(index, incarnation)
	}
	sort.Strings(writeSetKeys)
	s.txWritesetKeys.Store(index, writeSetKeys)
}

// GetAllWritesetKeys implements MultiVersionStore.
func (s *XSyncStore) GetAllWritesetKeys() map[int][]string {
	writesetKeys := make(map[int][]string)
	// TODO: is this safe?
	s.txWritesetKeys.Range(func(index int, value []string) bool {
		writesetKeys[index] = value
		return true
	})

	return writesetKeys
}

func (s *XSyncStore) SetReadset(index int, readset ReadSet) {
	s.txReadSets.Store(index, readset)
}

func (s *XSyncStore) GetReadset(index int) ReadSet {
	readsetAny, found := s.txReadSets.Load(index)
	if !found {
		return nil
	}
	return readsetAny
}

func (s *XSyncStore) SetIterateset(index int, iterateset Iterateset) {
	s.txIterateSets.Store(index, iterateset)
}

func (s *XSyncStore) GetIterateset(index int) Iterateset {
	iteratesetAny, found := s.txIterateSets.Load(index)
	if !found {
		return nil
	}
	return iteratesetAny
}

func (s *XSyncStore) ClearReadset(index int) {
	s.txReadSets.Delete(index)
}

func (s *XSyncStore) ClearIterateset(index int) {
	s.txIterateSets.Delete(index)
}

// CollectIteratorItems implements MultiVersionStore. It will return a memDB containing all of the keys present in the multiversion store within the iteration range prior to (exclusive of) the index.
func (s *XSyncStore) CollectIteratorItems(index int) *db.MemDB {
	sortedItems := db.NewMemDB()

	// get all writeset keys prior to index
	for i := 0; i < index; i++ {
		writesetAny, found := s.txWritesetKeys.Load(i)
		if !found {
			continue
		}
		//indexedWriteset := writesetAny.([]string)
		// TODO: do we want to exclude keys out of the range or just let the iterator handle it?
		for _, key := range writesetAny {
			// TODO: inefficient because (logn) for each key + rebalancing? maybe theres a better way to add to a tree to reduce rebalancing overhead
			sortedItems.Set([]byte(key), []byte{})
		}
	}
	return sortedItems
}

func (s *XSyncStore) validateIterator(index int, tracker iterationTracker) bool {
	// collect items from multiversion store
	sortedItems := s.CollectIteratorItems(index)
	// add the iterationtracker writeset keys to the sorted items
	for key := range tracker.writeset {
		sortedItems.Set([]byte(key), []byte{})
	}
	validChannel := make(chan bool, 1)
	abortChannel := make(chan occ.Abort, 1)

	// listen for abort while iterating
	go func(iterationTracker iterationTracker, items *db.MemDB, returnChan chan bool, abortChan chan occ.Abort) {
		var parentIter types.Iterator
		expectedKeys := iterationTracker.iteratedKeys
		foundKeys := 0
		iter := s.newMVSValidationIterator(index, iterationTracker.startKey, iterationTracker.endKey, items, iterationTracker.ascending, iterationTracker.writeset, abortChan)
		if iterationTracker.ascending {
			parentIter = s.parentStore.Iterator(iterationTracker.startKey, iterationTracker.endKey)
		} else {
			parentIter = s.parentStore.ReverseIterator(iterationTracker.startKey, iterationTracker.endKey)
		}
		// create a new MVSMergeiterator
		mergeIterator := NewMVSMergeIterator(parentIter, iter, iterationTracker.ascending, NoOpHandler{})
		defer mergeIterator.Close()
		for ; mergeIterator.Valid(); mergeIterator.Next() {
			if (len(expectedKeys) - foundKeys) == 0 {
				// if we have no more expected keys, then the iterator is invalid
				returnChan <- false
				return
			}
			key := mergeIterator.Key()
			// TODO: is this ok to not delete the key since we shouldnt have duplicate keys?
			if _, ok := expectedKeys[string(key)]; !ok {
				// if key isn't found
				returnChan <- false
				return
			}
			// remove from expected keys
			foundKeys += 1
			// delete(expectedKeys, string(key))

			// if our iterator key was the early stop, then we can break
			if bytes.Equal(key, iterationTracker.earlyStopKey) {
				break
			}
		}
		// return whether we found the exact number of expected keys
		returnChan <- !((len(expectedKeys) - foundKeys) > 0)
	}(tracker, sortedItems, validChannel, abortChannel)
	select {
	case <-abortChannel:
		// if we get an abort, then we know that the iterator is invalid
		return false
	case valid := <-validChannel:
		return valid
	}
}

func (s *XSyncStore) checkIteratorAtIndex(index int) bool {
	valid := true
	iterateset, found := s.txIterateSets.Load(index)
	if !found {
		return true
	}
	//iterateset := iterateSetAny.(Iterateset)
	for _, iterationTracker := range iterateset {
		// TODO: if the value of the key is nil maybe we need to exclude it? - actually it should
		iteratorValid := s.validateIterator(index, *iterationTracker)
		valid = valid && iteratorValid
	}
	return valid
}

func (s *XSyncStore) checkReadsetAtIndex(index int) (bool, []int) {
	conflictSet := make(map[int]struct{})
	valid := true

	readset, found := s.txReadSets.Load(index)
	if !found {
		return true, []int{}
	}
	//readset := readSetAny.(ReadSet)
	// iterate over readset and check if the value is the same as the latest value relateive to txIndex in the multiversion store
	for key, valueArr := range readset {
		if len(valueArr) != 1 {
			valid = false
			continue
		}
		value := valueArr[0]
		// get the latest value from the multiversion store
		latestValue := s.GetLatestBeforeIndex(index, []byte(key))
		if latestValue == nil {
			// this is possible if we previously read a value from a transaction write that was later reverted, so this time we read from parent store
			parentVal := s.parentStore.Get([]byte(key))
			if !bytes.Equal(parentVal, value) {
				valid = false
			}
		} else {
			// if estimate, mark as conflict index - but don't invalidate
			if latestValue.IsEstimate() {
				conflictSet[latestValue.Index()] = struct{}{}
			} else if latestValue.IsDeleted() {
				if value != nil {
					// conflict
					// TODO: would we want to return early?
					conflictSet[latestValue.Index()] = struct{}{}
					valid = false
				}
			} else if !bytes.Equal(latestValue.Value(), value) {
				conflictSet[latestValue.Index()] = struct{}{}
				valid = false
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

// TODO: do we want to return bool + []int where bool indicates whether it was valid and then []int indicates only ones for which we need to wait due to estimates? - yes i think so?
func (s *XSyncStore) ValidateTransactionState(index int) (bool, []int) {
	// defer telemetry.MeasureSince(time.Now(), "store", "mvs", "validate")

	// TODO: can we parallelize for all iterators?
	iteratorValid := s.checkIteratorAtIndex(index)

	readsetValid, conflictIndices := s.checkReadsetAtIndex(index)

	return iteratorValid && readsetValid, conflictIndices
}

func (s *XSyncStore) WriteLatestToStore() {
	// sort the keys
	keys := []string{}
	s.multiVersionMap.Range(func(key string, value *multiVersionItem) bool {
		keys = append(keys, key)
		return true
	})
	sort.Strings(keys)

	for _, key := range keys {
		val, ok := s.multiVersionMap.Load(key)
		if !ok {
			continue
		}
		mvValue, found := val.GetLatestNonEstimate()
		if !found {
			// this means that at some point, there was an estimate, but we have since removed it so there isn't anything writeable at the key, so we can skip
			continue
		}
		// we shouldn't have any ESTIMATE values when performing the write, because we read the latest non-estimate values only
		if mvValue.IsEstimate() {
			panic("should not have any estimate values when writing to parent store")
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

func (store *XSyncStore) newMVSValidationIterator(
	index int,
	start, end []byte,
	items *db.MemDB,
	ascending bool,
	writeset WriteSet,
	abortChannel chan occ.Abort,
) *validationIterator {
	var iter types.Iterator
	var err error

	if ascending {
		iter, err = items.Iterator(start, end)
	} else {
		iter, err = items.ReverseIterator(start, end)
	}

	if err != nil {
		if iter != nil {
			iter.Close()
		}
		panic(err)
	}

	return &validationIterator{
		Iterator:     iter,
		mvStore:      store,
		index:        index,
		abortChannel: abortChannel,
		writeset:     writeset,
		readCache:    make(map[string][]byte),
	}
}
