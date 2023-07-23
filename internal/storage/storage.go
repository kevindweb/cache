package storage

type KeyValue interface {
	New() KeyValue
	Clear() error
	Set([]byte, []byte) error
	Get([]byte) ([]byte, error)
	Del([]byte) error
}

var (
	Caches = []KeyValue{
		&CacheMap{},
	}
)
