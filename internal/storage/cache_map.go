package storage

import (
	"fmt"
)

type CacheMap struct {
	kv map[string][]byte
}

func NewCacheMap() KeyValue {
	cm := CacheMap{}
	return cm.New()
}

func (cm CacheMap) New() KeyValue {
	return &CacheMap{
		kv: map[string][]byte{},
	}
}

func (cm *CacheMap) Free() error {
	cm.kv = map[string][]byte{}
	return nil
}

func (cm *CacheMap) Set(key []byte, value []byte) error {
	cm.kv[string(key)] = cp(value)
	return nil
}

func cp(src []byte) []byte {
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}

func (cm *CacheMap) Get(key []byte) ([]byte, error) {
	val, ok := cm.kv[string(key)]
	if !ok {
		return []byte{}, fmt.Errorf("key %s not set", key)
	}
	return val, nil
}

func (cm *CacheMap) Del(key []byte) error {
	delete(cm.kv, string(key))
	return nil
}
