package multiversion

import (
	"bytes"
	"errors"

	"cosmossdk.io/store/types"
)

// mvsMergeIterator merges a parent Iterator and a cache Iterator.
// The cache iterator may return nil keys to signal that an item
// had been deleted (but not deleted in the parent).
// If the cache iterator has the same key as the parent, the
// cache shadows (overrides) the parent.
type mvsMergeIterator struct {
	parent    types.Iterator
	cache     types.Iterator
	ascending bool
	ReadsetHandler
}

var _ types.Iterator = (*mvsMergeIterator)(nil)

func NewMVSMergeIterator(
	parent, cache types.Iterator,
	ascending bool,
	readsetHandler ReadsetHandler,
) *mvsMergeIterator {
	iter := &mvsMergeIterator{
		parent:         parent,
		cache:          cache,
		ascending:      ascending,
		ReadsetHandler: readsetHandler,
	}

	return iter
}

// Domain implements Iterator.
// It returns the union of the iter.Parent doman, and the iter.Cache domain.
// If the domains are disjoint, this includes the domain in between them as well.
func (iter *mvsMergeIterator) Domain() (start, end []byte) {
	startP, endP := iter.parent.Domain()
	startC, endC := iter.cache.Domain()

	if iter.compare(startP, startC) < 0 {
		start = startP
	} else {
		start = startC
	}

	if iter.compare(endP, endC) < 0 {
		end = endC
	} else {
		end = endP
	}

	return start, end
}

// Valid implements Iterator.
func (iter *mvsMergeIterator) Valid() bool {
	return iter.skipUntilExistsOrInvalid()
}

// Next implements Iterator
func (iter *mvsMergeIterator) Next() {
	iter.skipUntilExistsOrInvalid()
	iter.assertValid()

	// If parent is invalid, get the next cache item.
	if !iter.parent.Valid() {
		iter.cache.Next()
		return
	}

	// If cache is invalid, get the next parent item.
	if !iter.cache.Valid() {
		iter.parent.Next()
		return
	}

	// Both are valid.  Compare keys.
	keyP, keyC := iter.parent.Key(), iter.cache.Key()
	switch iter.compare(keyP, keyC) {
	case -1: // parent < cache
		iter.parent.Next()
	case 0: // parent == cache
		iter.parent.Next()
		iter.cache.Next()
	case 1: // parent > cache
		iter.cache.Next()
	}
}

// Key implements Iterator
func (iter *mvsMergeIterator) Key() []byte {
	iter.skipUntilExistsOrInvalid()
	iter.assertValid()

	// If parent is invalid, get the cache key.
	if !iter.parent.Valid() {
		return iter.cache.Key()
	}

	// If cache is invalid, get the parent key.
	if !iter.cache.Valid() {
		return iter.parent.Key()
	}

	// Both are valid.  Compare keys.
	keyP, keyC := iter.parent.Key(), iter.cache.Key()

	cmp := iter.compare(keyP, keyC)
	switch cmp {
	case -1: // parent < cache
		return keyP
	case 0: // parent == cache
		return keyP
	case 1: // parent > cache
		return keyC
	default:
		panic("invalid compare result")
	}
}

// Value implements Iterator
func (iter *mvsMergeIterator) Value() []byte {
	iter.skipUntilExistsOrInvalid()
	iter.assertValid()

	// If parent is invalid, get the cache value.
	if !iter.parent.Valid() {
		value := iter.cache.Value()
		return value
	}

	// If cache is invalid, get the parent value.
	if !iter.cache.Valid() {
		value := iter.parent.Value()
		// add values read from parent to readset
		iter.ReadsetHandler.UpdateReadSet(iter.parent.Key(), value)
		return value
	}

	// Both are valid.  Compare keys.
	keyP, keyC := iter.parent.Key(), iter.cache.Key()

	cmp := iter.compare(keyP, keyC)
	switch cmp {
	case -1: // parent < cache
		value := iter.parent.Value()
		// add values read from parent to readset
		iter.ReadsetHandler.UpdateReadSet(iter.parent.Key(), value)
		return value
	case 0, 1: // parent >= cache
		value := iter.cache.Value()
		return value
	default:
		panic("invalid comparison result")
	}
}

// Close implements Iterator
func (iter *mvsMergeIterator) Close() error {
	if err := iter.parent.Close(); err != nil {
		// still want to close cache iterator regardless
		iter.cache.Close()
		return err
	}

	return iter.cache.Close()
}

// Error returns an error if the mvsMergeIterator is invalid defined by the
// Valid method.
func (iter *mvsMergeIterator) Error() error {
	if !iter.Valid() {
		return errors.New("invalid mvsMergeIterator")
	}

	return nil
}

// If not valid, panics.
// NOTE: May have side-effect of iterating over cache.
func (iter *mvsMergeIterator) assertValid() {
	if err := iter.Error(); err != nil {
		panic(err)
	}
}

// Like bytes.Compare but opposite if not ascending.
func (iter *mvsMergeIterator) compare(a, b []byte) int {
	if iter.ascending {
		return bytes.Compare(a, b)
	}

	return bytes.Compare(a, b) * -1
}

// Skip all delete-items from the cache w/ `key < until`.  After this function,
// current cache item is a non-delete-item, or `until <= key`.
// If the current cache item is not a delete item, does nothing.
// If `until` is nil, there is no limit, and cache may end up invalid.
// CONTRACT: cache is valid.
func (iter *mvsMergeIterator) skipCacheDeletes(until []byte) {
	for iter.cache.Valid() &&
		iter.cache.Value() == nil &&
		(until == nil || iter.compare(iter.cache.Key(), until) < 0) {
		iter.cache.Next()
	}
}

// Fast forwards cache (or parent+cache in case of deleted items) until current
// item exists, or until iterator becomes invalid.
// Returns whether the iterator is valid.
func (iter *mvsMergeIterator) skipUntilExistsOrInvalid() bool {
	for {
		// If parent is invalid, fast-forward cache.
		if !iter.parent.Valid() {
			iter.skipCacheDeletes(nil)
			return iter.cache.Valid()
		}
		// Parent is valid.
		if !iter.cache.Valid() {
			return true
		}
		// Parent is valid, cache is valid.

		// Compare parent and cache.
		keyP := iter.parent.Key()
		keyC := iter.cache.Key()

		switch iter.compare(keyP, keyC) {
		case -1: // parent < cache.
			return true

		case 0: // parent == cache.
			// Skip over if cache item is a delete.
			valueC := iter.cache.Value()
			if valueC == nil {
				iter.parent.Next()
				iter.cache.Next()

				continue
			}
			// Cache is not a delete.

			return true // cache exists.
		case 1: // cache < parent
			// Skip over if cache item is a delete.
			valueC := iter.cache.Value()
			if valueC == nil {
				iter.skipCacheDeletes(keyP)
				continue
			}
			// Cache is not a delete.

			return true // cache exists.
		}
	}
}
