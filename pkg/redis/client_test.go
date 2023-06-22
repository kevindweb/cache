package redis

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientInit(t *testing.T) {
	host := "localhost"
	port := 5432
	go func() {
		NewServer(host, port)
	}()
	c, err := NewClient(host, port)
	require.NoError(t, err)
	require.NotNil(t, c, "client shouldn't be nil")
}
