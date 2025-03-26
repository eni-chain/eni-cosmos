package rwset

import (
	"cosmossdk.io/store/types"
	"github.com/google/btree"
	"sync"
)

const (
	writeSetBTreeDegree = 2
)

type WriteSetValue interface {
	GetLatest() (value WriteSetValueItem, found bool)
	GetLatestValue() (value WriteSetValueItem, found bool)
	GetLatestBeforeIndex(index int) (value WriteSetValueItem, found bool)
	Set(index int, value []byte)
	Delete(index int)
	Remove(index int)
}

type WriteSetValueItem interface {
	IsDeleted() bool
	Value() []byte
	Index() int
}

type writeSetItem struct {
	valueTree *btree.BTree // contains versions values written to this key
	mtx       sync.RWMutex // manages read + write accesses
}

var _ WriteSetValue = (*writeSetItem)(nil)

func NewWriteSetItem() WriteSetValue {
	return &writeSetItem{
		valueTree: btree.New(writeSetBTreeDegree),
	}
}

// GetLatest returns the latest written value to the btree, and returns a boolean indicating whether it was found.
func (item *writeSetItem) GetLatest() (WriteSetValueItem, bool) {
	item.mtx.RLock()
	defer item.mtx.RUnlock()

	bTreeItem := item.valueTree.Max()
	if bTreeItem == nil {
		return nil, false
	}
	valueItem := bTreeItem.(*valueItem)
	return valueItem, true
}

// GetLatestValue returns the latest written value that isn't an ESTIMATE and returns a boolean indicating whether it was found.
// This can be used when we want to write finalized values, since ESTIMATEs can be considered to be irrelevant at that point
func (item *writeSetItem) GetLatestValue() (WriteSetValueItem, bool) {
	item.mtx.RLock()
	defer item.mtx.RUnlock()

	var vItem *valueItem
	var found bool
	item.valueTree.Descend(func(bTreeItem btree.Item) bool {
		// only return if non-estimate
		item := bTreeItem.(*valueItem)
		// else we want to return
		vItem = item
		found = true
		return false
	})
	return vItem, found
}

// GetLatestBeforeIndex returns the latest written value to the btree prior to the index passed in, and returns a boolean indicating whether it was found.
//
// A `nil` value along with `found=true` indicates a deletion that has occurred and the underlying parent store doesn't need to be hit.
func (item *writeSetItem) GetLatestBeforeIndex(index int) (WriteSetValueItem, bool) {
	item.mtx.RLock()
	defer item.mtx.RUnlock()

	// we want to find the value at the index that is LESS than the current index
	pivot := &valueItem{index: index - 1}

	var vItem *valueItem
	var found bool
	// start from pivot which contains our current index, and return on first item we hit.
	// This will ensure we get the latest indexed value relative to our current index
	item.valueTree.DescendLessOrEqual(pivot, func(bTreeItem btree.Item) bool {
		vItem = bTreeItem.(*valueItem)
		found = true
		return false
	})
	return vItem, found
}

func (item *writeSetItem) Set(index int, value []byte) {
	types.AssertValidValue(value)
	item.mtx.Lock()
	defer item.mtx.Unlock()

	valueItem := NewValueItem(index, value)
	item.valueTree.ReplaceOrInsert(valueItem)
}

func (item *writeSetItem) Delete(index int) {
	item.mtx.Lock()
	defer item.mtx.Unlock()

	deletedItem := NewDeletedItem(index)
	item.valueTree.ReplaceOrInsert(deletedItem)
}

func (item *writeSetItem) Remove(index int) {
	item.mtx.Lock()
	defer item.mtx.Unlock()

	item.valueTree.Delete(&valueItem{index: index})
}

type valueItem struct {
	index int
	value []byte
}

var _ WriteSetValueItem = (*valueItem)(nil)

// Index implements WriteSetValueItem.
func (v *valueItem) Index() int {
	return v.index
}

// IsDeleted implements WriteSetValueItem.
func (v *valueItem) IsDeleted() bool {
	return v.value == nil
}

// Value implements WriteSetValueItem.
func (v *valueItem) Value() []byte {
	return v.value
}

// Less implement Less for btree.Item for valueItem
func (i *valueItem) Less(other btree.Item) bool {
	return i.index < other.(*valueItem).index
}

func NewValueItem(index int, value []byte) *valueItem {
	return &valueItem{
		index: index,
		value: value,
	}
}

func NewDeletedItem(index int) *valueItem {
	return &valueItem{
		index: index,
		value: nil,
	}
}
