package storage

type KeyValue interface {
	New() KeyValue
	Free() error
	Set([]byte, []byte) error
	Get([]byte) ([]byte, error)
	Del([]byte) error
}

func caches() []KeyValue {
	return []KeyValue{
		NewCacheMap(),
	}
}
