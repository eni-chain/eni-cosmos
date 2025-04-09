package cachekv_test

import (
	"cosmossdk.io/store/types"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/store/cachekv"
	"cosmossdk.io/store/dbadapter"
)

func newSyncCacheKVStore() types.CacheKVStore {
	mem := dbadapter.Store{DB: dbm.NewMemDB()}
	return cachekv.NewSyncStore(mem)
}

func TestSyncCacheKVStore(t *testing.T) {
	mem := dbadapter.Store{DB: dbm.NewMemDB()}
	st := cachekv.NewSyncStore(mem)

	require.Empty(t, st.Get(keyFmt(1)), "Expected `key1` to be empty")

	// put something in mem and in cache
	mem.Set(keyFmt(1), valFmt(1))
	st.Set(keyFmt(1), valFmt(1))
	require.Equal(t, valFmt(1), st.Get(keyFmt(1)))

	// update it in cache, shoudn't change mem
	st.Set(keyFmt(1), valFmt(2))
	require.Equal(t, valFmt(2), st.Get(keyFmt(1)))
	require.Equal(t, valFmt(1), mem.Get(keyFmt(1)))

	// write it. should change mem
	st.Write()
	require.Equal(t, valFmt(2), mem.Get(keyFmt(1)))
	require.Equal(t, valFmt(2), st.Get(keyFmt(1)))

	// more writes and checks
	st.Write()
	st.Write()
	require.Equal(t, valFmt(2), mem.Get(keyFmt(1)))
	require.Equal(t, valFmt(2), st.Get(keyFmt(1)))

	// make a new one, check it
	st = cachekv.NewSyncStore(mem)
	require.Equal(t, valFmt(2), st.Get(keyFmt(1)))

	// make a new one and delete - should not be removed from mem
	st = cachekv.NewSyncStore(mem)
	st.Delete(keyFmt(1))
	require.Empty(t, st.Get(keyFmt(1)))
	require.Equal(t, mem.Get(keyFmt(1)), valFmt(2))

	// Write. should now be removed from both
	st.Write()
	require.Empty(t, st.Get(keyFmt(1)), "Expected `key1` to be empty")
	require.Empty(t, mem.Get(keyFmt(1)), "Expected `key1` to be empty")
}

func TestSyncCacheKVStoreNoNilSet(t *testing.T) {
	mem := dbadapter.Store{DB: dbm.NewMemDB()}
	st := cachekv.NewSyncStore(mem)
	require.Panics(t, func() { st.Set([]byte("key"), nil) }, "setting a nil value should panic")
	require.Panics(t, func() { st.Set(nil, []byte("value")) }, "setting a nil key should panic")
	require.Panics(t, func() { st.Set([]byte(""), []byte("value")) }, "setting an empty key should panic")
}

func TestSyncCacheKVStoreNested(t *testing.T) {
	mem := dbadapter.Store{DB: dbm.NewMemDB()}
	st := cachekv.NewSyncStore(mem)

	// set. check its there on st and not on mem.
	st.Set(keyFmt(1), valFmt(1))
	require.Empty(t, mem.Get(keyFmt(1)))
	require.Equal(t, valFmt(1), st.Get(keyFmt(1)))

	// make a new from st and check
	st2 := cachekv.NewSyncStore(st)
	require.Equal(t, valFmt(1), st2.Get(keyFmt(1)))

	// update the value on st2, check it only effects st2
	st2.Set(keyFmt(1), valFmt(3))
	require.Equal(t, []byte(nil), mem.Get(keyFmt(1)))
	require.Equal(t, valFmt(1), st.Get(keyFmt(1)))
	require.Equal(t, valFmt(3), st2.Get(keyFmt(1)))

	// st2 writes to its parent, st. doesnt effect mem
	st2.Write()
	require.Equal(t, []byte(nil), mem.Get(keyFmt(1)))
	require.Equal(t, valFmt(3), st.Get(keyFmt(1)))

	// updates mem
	st.Write()
	require.Equal(t, valFmt(3), mem.Get(keyFmt(1)))
}

func TestSyncCacheKVIteratorBounds(t *testing.T) {
	st := newSyncCacheKVStore()

	// set some items
	nItems := 5
	for i := 0; i < nItems; i++ {
		st.Set(keyFmt(i), valFmt(i))
	}

	// iterate over all of them
	itr := st.Iterator(nil, nil)
	i := 0
	for ; itr.Valid(); itr.Next() {
		k, v := itr.Key(), itr.Value()
		require.Equal(t, keyFmt(i), k)
		require.Equal(t, valFmt(i), v)
		i++
	}
	require.Equal(t, nItems, i)
	require.NoError(t, itr.Close())

	// iterate over none
	itr = st.Iterator(bz("money"), nil)
	i = 0
	for ; itr.Valid(); itr.Next() {
		i++
	}
	require.Equal(t, 0, i)
	require.NoError(t, itr.Close())

	// iterate over lower
	itr = st.Iterator(keyFmt(0), keyFmt(3))
	i = 0
	for ; itr.Valid(); itr.Next() {
		k, v := itr.Key(), itr.Value()
		require.Equal(t, keyFmt(i), k)
		require.Equal(t, valFmt(i), v)
		i++
	}
	require.Equal(t, 3, i)
	require.NoError(t, itr.Close())

	// iterate over upper
	itr = st.Iterator(keyFmt(2), keyFmt(4))
	i = 2
	for ; itr.Valid(); itr.Next() {
		k, v := itr.Key(), itr.Value()
		require.Equal(t, keyFmt(i), k)
		require.Equal(t, valFmt(i), v)
		i++
	}
	require.Equal(t, 4, i)
	require.NoError(t, itr.Close())
}

func TestSyncCacheKVReverseIteratorBounds(t *testing.T) {
	st := newSyncCacheKVStore()

	// set some items
	nItems := 5
	for i := 0; i < nItems; i++ {
		st.Set(keyFmt(i), valFmt(i))
	}

	// iterate over all of them
	itr := st.ReverseIterator(nil, nil)
	i := 0
	for ; itr.Valid(); itr.Next() {
		k, v := itr.Key(), itr.Value()
		require.Equal(t, keyFmt(nItems-1-i), k)
		require.Equal(t, valFmt(nItems-1-i), v)
		i++
	}
	require.Equal(t, nItems, i)
	require.NoError(t, itr.Close())

	// iterate over none
	itr = st.ReverseIterator(bz("money"), nil)
	i = 0
	for ; itr.Valid(); itr.Next() {
		i++
	}
	require.Equal(t, 0, i)
	require.NoError(t, itr.Close())

	// iterate over lower
	end := 3
	itr = st.ReverseIterator(keyFmt(0), keyFmt(end))
	i = 0
	for ; itr.Valid(); itr.Next() {
		i++
		k, v := itr.Key(), itr.Value()
		require.Equal(t, keyFmt(end-i), k)
		require.Equal(t, valFmt(end-i), v)
	}
	require.Equal(t, 3, i)
	require.NoError(t, itr.Close())

	// iterate over upper
	end = 4
	itr = st.ReverseIterator(keyFmt(2), keyFmt(end))
	i = 0
	for ; itr.Valid(); itr.Next() {
		i++
		k, v := itr.Key(), itr.Value()
		require.Equal(t, keyFmt(end-i), k)
		require.Equal(t, valFmt(end-i), v)
	}
	require.Equal(t, 2, i)
	require.NoError(t, itr.Close())
}

func TestSyncCacheKVMergeIteratorBasics(t *testing.T) {
	st := newSyncCacheKVStore()

	// set and delete an item in the cache, iterator should be empty
	k, v := keyFmt(0), valFmt(0)
	st.Set(k, v)
	st.Delete(k)
	assertIterateDomain(t, st, 0)

	// now set it and assert its there
	st.Set(k, v)
	assertIterateDomain(t, st, 1)

	// write it and assert its there
	st.Write()
	assertIterateDomain(t, st, 1)

	// remove it in cache and assert its not
	st.Delete(k)
	assertIterateDomain(t, st, 0)

	// write the delete and assert its not there
	st.Write()
	assertIterateDomain(t, st, 0)

	// add two keys and assert theyre there
	k1, v1 := keyFmt(1), valFmt(1)
	st.Set(k, v)
	st.Set(k1, v1)
	assertIterateDomain(t, st, 2)

	// write it and assert theyre there
	st.Write()
	assertIterateDomain(t, st, 2)

	// remove one in cache and assert its not
	st.Delete(k1)
	assertIterateDomain(t, st, 1)

	// write the delete and assert its not there
	st.Write()
	assertIterateDomain(t, st, 1)

	// delete the other key in cache and asserts its empty
	st.Delete(k)
	assertIterateDomain(t, st, 0)
}

func TestSyncCacheKVMergeIteratorDeleteLast(t *testing.T) {
	st := newSyncCacheKVStore()

	// set some items and write them
	nItems := 5
	for i := 0; i < nItems; i++ {
		st.Set(keyFmt(i), valFmt(i))
	}
	st.Write()

	// set some more items and leave dirty
	for i := nItems; i < nItems*2; i++ {
		st.Set(keyFmt(i), valFmt(i))
	}

	// iterate over all of them
	assertIterateDomain(t, st, nItems*2)

	// delete them all
	for i := 0; i < nItems*2; i++ {
		last := nItems*2 - 1 - i
		st.Delete(keyFmt(last))
		assertIterateDomain(t, st, last)
	}
}

func TestSyncCacheKVMergeIteratorDeletes(t *testing.T) {
	st := newSyncCacheKVStore()
	truth := dbm.NewMemDB()

	// set some items and write them
	nItems := 10
	for i := 0; i < nItems; i++ {
		doOp(t, st, truth, opSet, i)
	}
	st.Write()

	// delete every other item, starting from 0
	for i := 0; i < nItems; i += 2 {
		doOp(t, st, truth, opDel, i)
		assertIterateDomainCompare(t, st, truth)
	}

	// reset
	st = newSyncCacheKVStore()
	truth = dbm.NewMemDB()

	// set some items and write them
	for i := 0; i < nItems; i++ {
		doOp(t, st, truth, opSet, i)
	}
	st.Write()

	// delete every other item, starting from 1
	for i := 1; i < nItems; i += 2 {
		doOp(t, st, truth, opDel, i)
		assertIterateDomainCompare(t, st, truth)
	}
}

func TestSyncCacheKVMergeIteratorChunks(t *testing.T) {
	st := newSyncCacheKVStore()

	// Use the truth to check values on the merge iterator
	truth := dbm.NewMemDB()

	// sets to the parent
	setRange(t, st, truth, 0, 20)
	setRange(t, st, truth, 40, 60)
	st.Write()

	// sets to the cache
	setRange(t, st, truth, 20, 40)
	setRange(t, st, truth, 60, 80)
	assertIterateDomainCheck(t, st, truth, []keyRange{{0, 80}})

	// remove some parents and some cache
	deleteRange(t, st, truth, 15, 25)
	assertIterateDomainCheck(t, st, truth, []keyRange{{0, 15}, {25, 80}})

	// remove some parents and some cache
	deleteRange(t, st, truth, 35, 45)
	assertIterateDomainCheck(t, st, truth, []keyRange{{0, 15}, {25, 35}, {45, 80}})

	// write, add more to the cache, and delete some cache
	st.Write()
	setRange(t, st, truth, 38, 42)
	deleteRange(t, st, truth, 40, 43)
	assertIterateDomainCheck(t, st, truth, []keyRange{{0, 15}, {25, 35}, {38, 40}, {45, 80}})
}

func TestSyncCacheKVMergeIteratorDomain(t *testing.T) {
	st := newSyncCacheKVStore()

	itr := st.Iterator(nil, nil)
	start, end := itr.Domain()
	require.Equal(t, start, end)
	require.NoError(t, itr.Close())

	itr = st.Iterator(keyFmt(40), keyFmt(60))
	start, end = itr.Domain()
	require.Equal(t, keyFmt(40), start)
	require.Equal(t, keyFmt(60), end)
	require.NoError(t, itr.Close())

	start, end = st.ReverseIterator(keyFmt(0), keyFmt(80)).Domain()
	require.Equal(t, keyFmt(0), start)
	require.Equal(t, keyFmt(80), end)
}

func TestSyncCacheKVMergeIteratorRandom(t *testing.T) {
	st := newSyncCacheKVStore()
	truth := dbm.NewMemDB()

	start, end := 25, 975
	max := 1000
	setRange(t, st, truth, start, end)

	// do an op, test the iterator
	for i := 0; i < 2000; i++ {
		doRandomOp(t, st, truth, max)
		assertIterateDomainCompare(t, st, truth)
	}
}

func TestSyncNilEndIterator(t *testing.T) {
	const SIZE = 3000

	tests := []struct {
		name       string
		write      bool
		startIndex int
		end        []byte
	}{
		{name: "write=false, end=nil", write: false, end: nil, startIndex: 1000},
		{name: "write=false, end=nil; full key scan", write: false, end: nil, startIndex: 2000},
		{name: "write=true, end=nil", write: true, end: nil, startIndex: 1000},
		{name: "write=false, end=non-nil", write: false, end: keyFmt(3000), startIndex: 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := newSyncCacheKVStore()

			for i := 0; i < SIZE; i++ {
				kstr := keyFmt(i)
				st.Set(kstr, valFmt(i))
			}

			if tt.write {
				st.Write()
			}

			itr := st.Iterator(keyFmt(tt.startIndex), tt.end)
			i := tt.startIndex
			j := 0
			for itr.Valid() {
				require.Equal(t, keyFmt(i), itr.Key())
				require.Equal(t, valFmt(i), itr.Value())
				itr.Next()
				i++
				j++
			}

			require.Equal(t, SIZE-tt.startIndex, j)
			require.NoError(t, itr.Close())
		})
	}
}

// TestIteratorDeadlock demonstrate the deadlock issue in cache store.
func TestSyncIteratorDeadlock(t *testing.T) {
	mem := dbadapter.Store{DB: dbm.NewMemDB()}
	store := cachekv.NewSyncStore(mem)
	// the channel buffer is 64 and received once, so put at least 66 elements.
	for i := 0; i < 66; i++ {
		store.Set([]byte(fmt.Sprintf("key%d", i)), []byte{1})
	}
	it := store.Iterator(nil, nil)
	defer it.Close()
	store.Set([]byte("key20"), []byte{1})
	// it'll be blocked here with previous version, or enable lock on btree.
	it2 := store.Iterator(nil, nil)
	defer it2.Close()
}

type keySyncRange struct {
	start int
	end   int
}

func (kr keySyncRange) len() int {
	return kr.end - kr.start
}

// we can iterate over this and make sure our real iterators have all the right keys
type keySyncRangeCounter struct {
	rangeIdx  int
	idx       int
	keyRanges []keyRange
}

func (krc *keySyncRangeCounter) valid() bool {
	maxRangeIdx := len(krc.keyRanges) - 1
	maxRange := krc.keyRanges[maxRangeIdx]

	// if we're not in the max range, we're valid
	if krc.rangeIdx <= maxRangeIdx &&
		krc.idx < maxRange.len() {
		return true
	}

	return false
}

func (krc *keySyncRangeCounter) next() {
	thisKeyRange := krc.keyRanges[krc.rangeIdx]
	if krc.idx == thisKeyRange.len()-1 {
		krc.rangeIdx++
		krc.idx = 0
	} else {
		krc.idx++
	}
}

func (krc *keySyncRangeCounter) key() int {
	thisKeyRange := krc.keyRanges[krc.rangeIdx]
	return thisKeyRange.start + krc.idx
}

//--------------------------------------------------------

func BenchmarkSyncCacheKVStoreGetNoKeyFound(b *testing.B) {
	b.ReportAllocs()
	st := newSyncCacheKVStore()
	b.ResetTimer()
	// assumes b.N < 2**24
	for i := 0; i < b.N; i++ {
		st.Get([]byte{byte((i & 0xFF0000) >> 16), byte((i & 0xFF00) >> 8), byte(i & 0xFF)})
	}
}

func BenchmarkSyncCacheKVStoreGetKeyFound(b *testing.B) {
	b.ReportAllocs()
	st := newSyncCacheKVStore()
	for i := 0; i < b.N; i++ {
		arr := []byte{byte((i & 0xFF0000) >> 16), byte((i & 0xFF00) >> 8), byte(i & 0xFF)}
		st.Set(arr, arr)
	}
	b.ResetTimer()
	// assumes b.N < 2**24
	for i := 0; i < b.N; i++ {
		st.Get([]byte{byte((i & 0xFF0000) >> 16), byte((i & 0xFF00) >> 8), byte(i & 0xFF)})
	}
}

// 10 0.7389 0.67 0.6163  20 12978 ns  13459 ns 13417 ns  30 18226 ns 19482 ns 19993 ns 40  23082 ns 26994 ns 24203 ns/op
func BenchmarkSyncConcurrentRW10(b *testing.B) {
	st := newSyncCacheKVStore()
	b.N = 100000 //

	var wg sync.WaitGroup
	wg.Add(100) // 10 goroutine

	for i := 0; i < 50000; i++ {
		arr := []byte{byte((i & 0xFF0000) >> 16), byte((i & 0xFF00) >> 8), byte(i & 0xFF)}
		st.Set(arr, arr)
	}

	b.ResetTimer()
	for g := 0; g < 100; g++ { //40 10580 ns 10430 ns 10655 ns 30 10882 ns 11553 ns  10419 ns 50 10473 ns 10336 ns 10742 ns 100 13991 ns 13113 ns 16134 ns 14931 ns
		c := g
		go func() {
			defer wg.Done()
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			i := 0
			for ; i < b.N; i++ {
				i := r.Intn(1000000)
				val := st.Get([]byte{byte((i & 0xFF0000) >> 16), byte((i & 0xFF00) >> 8), byte(i & 0xFF)})
				if val == nil {
					arr := []byte{byte((i & 0xFF0000) >> 16), byte((i & 0xFF00) >> 8), byte(i & 0xFF)}
					st.Set(arr, arr)
				}
			}
			fmt.Printf("g %d i %d\n", c, i)
		}()
	}
	wg.Wait()
	//fmt.Printf("elapsed time %d\n", time.Since(startTime).Microseconds())
}
