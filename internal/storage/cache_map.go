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

func (cm *CacheMap) Clear() error {
	cm.kv = map[string][]byte{}
	return nil
}

func (cm *CacheMap) Set(key []byte, value []byte) error {
	cm.kv[string(key)] = value
	return nil
}

func (cm *CacheMap) Get(key []byte) ([]byte, error) {
	if val, ok := cm.kv[string(key)]; !ok {
		return []byte{}, fmt.Errorf("key %s not set", key)
	} else {
		return val, nil
	}
}

func (cm *CacheMap) Del(key []byte) error {
	delete(cm.kv, string(key))
	return nil
}
