package redis

import (
	"net"
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
	err = c.TearDown()
	require.NoError(t, err)
}

func setupClient() *Client {
	return &Client{
		requests: make(chan clientReq),
		url:      "localhost:1234",
		conn:     &net.TCPConn{},
	}
}

func TestGet(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		key               string
		response          string
		failBeforeRequest bool
		check             func(*testing.T, string, error)
	}{
		{
			name:     "valid get",
			key:      "key",
			response: "data",
			check: func(t *testing.T, v string, err error) {
				require.NoError(t, err)
				require.Equal(t, "data", v)
			},
		},
		{
			name:     "expect error on invalid get response",
			key:      "key",
			response: "",
			check: func(t *testing.T, _ string, err error) {
				require.Error(t, err)
			},
		},
		{
			name:              "empty key",
			failBeforeRequest: true,
			check: func(t *testing.T, _ string, err error) {
				require.Error(t, err)
				require.Equal(t, EmptyParamErr, err.Error())
			},
		},
		{
			name:     "error response",
			key:      "key",
			response: "-Invalid get",
			check: func(t *testing.T, _ string, err error) {
				require.Equal(t, "Invalid get", err.Error())
			},
		},
	}
	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := setupClient()
			if !tt.failBeforeRequest {
				go func() {
					req := <-c.requests
					require.Equal(t, []string{GET, tt.key}, req.req)
					req.res <- []string{tt.response}
				}()
			}
			val, err := c.Get(tt.key)
			tt.check(t, val, err)
		})
	}
}

func TestSet(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		key               string
		val               string
		response          string
		failBeforeRequest bool
		check             func(*testing.T, error)
	}{
		{
			name:     "valid set",
			key:      "key",
			val:      "val",
			response: OK,
			check: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:     "expect error on invalid set response",
			key:      "key",
			val:      "val",
			response: "",
			check: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
		{
			name:              "empty val",
			key:               "key",
			failBeforeRequest: true,
			check: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(t, EmptyParamErr, err.Error())
			},
		},
		{
			name:     "error response",
			key:      "key",
			val:      "val",
			response: "-Invalid set",
			check: func(t *testing.T, err error) {
				require.Equal(t, "Invalid set", err.Error())
			},
		},
	}
	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := setupClient()
			if !tt.failBeforeRequest {
				go func() {
					req := <-c.requests
					require.Equal(t, []string{SET, tt.key, tt.val}, req.req)
					req.res <- []string{tt.response}
				}()
			}
			tt.check(t, c.Set(tt.key, tt.val))
		})
	}
}
func TestDelete(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		key               string
		response          string
		failBeforeRequest bool
		check             func(*testing.T, error)
	}{
		{
			name:     "valid delete",
			key:      "delete",
			response: OK,
			check: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:     "expect error on invalid del response",
			key:      "delete",
			response: PING,
			check: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
		{
			name:              "empty key",
			failBeforeRequest: true,
			check: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(t, EmptyParamErr, err.Error())
			},
		},
		{
			name:     "error response",
			key:      "delete",
			response: "-Invalid delete",
			check: func(t *testing.T, err error) {
				require.Equal(t, "Invalid delete", err.Error())
			},
		},
	}
	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := setupClient()
			if !tt.failBeforeRequest {
				go func() {
					req := <-c.requests
					require.Equal(t, []string{DEL, tt.key}, req.req)
					req.res <- []string{tt.response}
				}()
			}
			tt.check(t, c.Del(tt.key))
		})
	}
}

func TestPing(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		response          string
		failBeforeRequest bool
		check             func(*testing.T, error)
	}{
		{
			name:     "valid ping pong",
			response: PONG,
			check: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:     "server responds PING expect error",
			response: PING,
			check: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
		{
			name:     "error response",
			response: "-Invalid request",
			check: func(t *testing.T, err error) {
				require.Equal(t, "Invalid request", err.Error())
			},
		},
		{
			name:              "invalid client initialization",
			failBeforeRequest: true,
			check: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
	}
	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var c *Client
			if !tt.failBeforeRequest {
				c = setupClient()
				go func() {
					req := <-c.requests
					require.Equal(t, []string{PING}, req.req)
					req.res <- []string{tt.response}
				}()
			} else {
				c = &Client{}
			}
			tt.check(t, c.Ping())
		})
	}
}
