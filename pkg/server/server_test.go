package server

import (
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/kevindweb/cache/internal/constants"
	"github.com/kevindweb/cache/internal/protocol"
	"github.com/kevindweb/cache/internal/storage"

	"github.com/stretchr/testify/assert"
)

func TestProcessRequestEmptyCache(t *testing.T) {
	t.Parallel()
	key := []byte("key")
	value := []byte("key")
	tests := []struct {
		name string
		op   protocol.Operation
		want protocol.Result
	}{
		{
			name: "valid set",
			op: protocol.Operation{
				Type:  protocol.SET,
				Key:   key,
				Value: value,
			},
			want: protocol.Result{
				Status:  protocol.SUCCESS,
				Message: constants.Ok(),
			},
		},
		{
			name: "no key to get",
			op: protocol.Operation{
				Type: protocol.GET,
				Key:  key,
			},
			want: protocol.Result{
				Status:  protocol.FAILURE,
				Message: []byte(fmt.Sprintf(storage.UnsetKeyErr, key)),
			},
		},
		{
			name: "valid empty delete",
			op: protocol.Operation{
				Type: protocol.DELETE,
				Key:  key,
			},
			want: protocol.Result{
				Status:  protocol.SUCCESS,
				Message: constants.Ok(),
			},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := Server{
				kv: storage.NewCacheMap(),
				ok: constants.Ok(),
			}
			res := server.processRequest(tc.op)
			assert.Equal(t, tc.want, res)
		})
	}
}

func TestHeader(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		input  []byte
		buffer []byte
	}{
		{
			name:   "large buffer should not overflow",
			input:  []byte("h"),
			buffer: []byte("this is a large buffer"),
		},
		{
			name:   "empty data",
			input:  []byte{},
			buffer: make([]byte, 100),
		},
		{
			name:   "empty buffer",
			input:  []byte("h"),
			buffer: make([]byte, 0),
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := &Server{
				outBuffer: tc.buffer,
			}
			got := server.writeHeader(tc.input)
			length := int(binary.LittleEndian.Uint32(got[:constants.HeaderSize]))
			assert.Equal(t, len(tc.input), length)
			assert.Equal(t, tc.input, got[constants.HeaderSize:])
			assert.True(t, len(server.outBuffer) >= len(tc.input)+constants.HeaderSize)
		})
	}
}
