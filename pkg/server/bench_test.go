package server

import (
	"testing"

	"github.com/kevindweb/cache/internal/constants"
	"github.com/kevindweb/cache/internal/protocol"
	"github.com/kevindweb/cache/internal/util"
)

func encode(b *testing.B, data protocol.BatchedRequest) []byte {
	var encoded []byte
	var err error
	if encoded, err = data.MarshalMsg(nil); err != nil {
		b.Fatal(err)
	}
	return encoded
}

func startUniqueServer(b *testing.B) (*Server, func()) {
	server, err := StartOptions(Options{
		Host:    constants.DefaultHost,
		Port:    util.GetUniquePort(),
		Network: constants.DefaultNetwork,
	})
	if err != nil {
		b.Fatal(err)
	}

	return server, func() {
		stopErr := server.Stop()
		if stopErr != nil {
			b.Fatal(stopErr)
		}
	}
}

func BenchmarkSingleSet(b *testing.B) {
	batch := protocol.BatchedRequest{
		Operations: []protocol.Operation{
			{
				Type:  protocol.SET,
				Key:   []byte("world"),
				Value: []byte("hello"),
			},
		},
	}

	encoded := encode(b, batch)
	server, stop := startUniqueServer(b)
	defer stop()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.eventHandler(nil, encoded)
	}
	b.StopTimer()
}

func BenchmarkBulkSet(b *testing.B) {
	numSetOperations := 100
	operations := make([]protocol.Operation, 0, numSetOperations)
	for i := 0; i < numSetOperations; i++ {
		operations = append(operations, protocol.Operation{
			Type:  protocol.SET,
			Key:   []byte("world"),
			Value: []byte("hello"),
		})
	}
	batch := protocol.BatchedRequest{
		Operations: operations,
	}

	encoded := encode(b, batch)
	server, stop := startUniqueServer(b)
	defer stop()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.eventHandler(nil, encoded)
	}
	b.StopTimer()
}

func BenchmarkSingleGet(b *testing.B) {
	batch := protocol.BatchedRequest{
		Operations: []protocol.Operation{
			{
				Type:  protocol.SET,
				Key:   []byte("world"),
				Value: []byte("hello"),
			},
		},
	}

	encodedSet := encode(b, batch)
	server, stop := startUniqueServer(b)
	defer stop()
	server.eventHandler(nil, encodedSet)

	batchGet := protocol.BatchedRequest{
		Operations: []protocol.Operation{
			{
				Type: protocol.GET,
				Key:  []byte("world"),
			},
		},
	}

	encodedGet := encode(b, batchGet)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.eventHandler(nil, encodedGet)
	}
	b.StopTimer()
}

func BenchmarkBulkGet(b *testing.B) {
	setOperation := protocol.BatchedRequest{
		Operations: []protocol.Operation{
			{
				Type:  protocol.SET,
				Key:   []byte("world"),
				Value: []byte("hello"),
			},
		},
	}

	encodedSet := encode(b, setOperation)
	server, stop := startUniqueServer(b)
	defer stop()
	server.eventHandler(nil, encodedSet)

	numSetOperations := constants.MaxRequestBatch
	operations := make([]protocol.Operation, 0, numSetOperations)
	for i := 0; i < numSetOperations; i++ {
		operations = append(operations, protocol.Operation{
			Type: protocol.GET,
			Key:  []byte("world"),
		})
	}
	batchGet := protocol.BatchedRequest{
		Operations: operations,
	}

	encodedGet := encode(b, batchGet)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.eventHandler(nil, encodedGet)
	}
	b.StopTimer()
}
