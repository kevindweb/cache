package server

import (
	"cache/internal/constants"
	"cache/internal/protocol"
	"sync/atomic"
	"testing"
)

var (
	port = int32(constants.DefaultPort)
)

func startWithOptions(opts Options) (*Server, error) {
	s, err := New(opts)
	if err != nil {
		return nil, err
	}

	go func() {
		err := s.Start()
		if err != nil {
			panic(err)
		}
	}()

	return s, nil
}

func BenchmarkSet(b *testing.B) {
	batchSize := 1
	ops := []protocol.Operation{}
	for i := 0; i < batchSize; i++ {
		ops = append(ops, protocol.Operation{
			Type:  protocol.SET,
			Key:   []byte("world"),
			Value: []byte("hello"),
		})
	}

	batch := protocol.BatchedRequest{
		Operations: ops,
	}

	var encoded []byte
	var err error
	if encoded, err = batch.MarshalMsg(nil); err != nil {
		b.Fatal(err)
	}

	nextPort := int(atomic.AddInt32(&port, 1))
	opts := Options{
		Host:    constants.DefaultHost,
		Port:    nextPort,
		Network: constants.DefaultNetwork,
	}

	server, err := startWithOptions(opts)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.eventHandler(nil, encoded)
	}
	b.StopTimer()

	server.Stop()
}
