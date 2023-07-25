package server

import (
	"testing"

	"cache/internal/constants"
	"cache/internal/protocol"
	"cache/internal/util"
)

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
		err := server.Stop()
		if err != nil {
			b.Fatal(err)
		}
	}
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
