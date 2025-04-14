package mocks

import (
	store "cosmossdk.io/store/types"
	"io"
)

type MockBenchMultiStore struct {
	Stores map[store.StoreKey]store.KVStore
}

func (m *MockBenchMultiStore) Write() {
	//TODO implement me
	panic("implement me")
}

func (m *MockBenchMultiStore) GetStoreType() store.StoreType {

	//TODO implement me
	panic("implement me")
}

func (m *MockBenchMultiStore) CacheWrap() store.CacheWrap {
	//TODO implement me
	panic("implement me")
}

func (m *MockBenchMultiStore) CacheWrapWithTrace(w io.Writer, tc store.TraceContext) store.CacheWrap {
	//TODO implement me
	panic("implement me")
}

func (m *MockBenchMultiStore) CacheMultiStoreWithVersion(version int64) (store.CacheMultiStore, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockBenchMultiStore) GetStore(key store.StoreKey) store.Store {
	//TODO implement me
	panic("implement me")
}

func (m *MockBenchMultiStore) TracingEnabled() bool {
	//TODO implement me
	panic("implement me")
}

func (m *MockBenchMultiStore) SetTracer(w io.Writer) store.MultiStore {
	//TODO implement me
	panic("implement me")
}

func (m *MockBenchMultiStore) SetTracingContext(context store.TraceContext) store.MultiStore {
	//TODO implement me
	panic("implement me")
}

func (m *MockBenchMultiStore) LatestVersion() int64 {
	//TODO implement me
	panic("implement me")
}

func (m *MockBenchMultiStore) GetKVStore(key store.StoreKey) store.KVStore {
	return m.Stores[key]
}

func (m *MockBenchMultiStore) CacheMultiStore() store.CacheMultiStore {
	return m
}

func (m *MockBenchMultiStore) StoreKeys() []store.StoreKey {
	keys := make([]store.StoreKey, 0, len(m.Stores))
	for k := range m.Stores {
		keys = append(keys, k)
	}
	return keys
}

func (m *MockBenchMultiStore) SetKVStores(_ func(store.StoreKey, store.KVStore) store.CacheWrap) store.MultiStore {
	return m
}

type MockBenchKVStore struct {
	data map[string][]byte
}

func (m *MockBenchKVStore) GetStoreType() store.StoreType {
	//TODO implement me
	panic("implement me")
}

func (m *MockBenchKVStore) CacheWrap() store.CacheWrap {
	//TODO implement me
	panic("implement me")
}

func (m *MockBenchKVStore) CacheWrapWithTrace(w io.Writer, tc store.TraceContext) store.CacheWrap {
	//TODO implement me
	panic("implement me")
}

func (m *MockBenchKVStore) Iterator(start, end []byte) store.Iterator {
	//TODO implement me
	panic("implement me")
}

func (m *MockBenchKVStore) ReverseIterator(start, end []byte) store.Iterator {
	//TODO implement me
	panic("implement me")
}

func (m *MockBenchKVStore) Get(key []byte) []byte {
	if m.data == nil {
		return nil
	}
	return m.data[string(key)]
}

func (m *MockBenchKVStore) Set(key, value []byte) {
	if m.data == nil {
		m.data = make(map[string][]byte)
	}
	m.data[string(key)] = value
}

func (m *MockBenchKVStore) Has(key []byte) bool {
	if m.data == nil {
		return false
	}
	_, ok := m.data[string(key)]
	return ok
}

func (m *MockBenchKVStore) Delete(key []byte) {
	if m.data != nil {
		delete(m.data, string(key))
	}
}
