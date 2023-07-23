package server

import (
	"testing"

	"cache/internal/constants"
	"cache/internal/protocol"
	"cache/internal/util"
)

func BenchmarkSingleSet(b *testing.B) {
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

	nextPort := util.GetUniquePort()
	opts := Options{
		Host:    constants.DefaultHost,
		Port:    nextPort,
		Network: constants.DefaultNetwork,
	}

	server, err := StartOptions(opts)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.eventHandler(nil, encoded)
	}
	b.StopTimer()

	err = server.Stop()
	if err != nil {
		b.Fatal(err)
	}
}
