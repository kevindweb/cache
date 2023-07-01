package redis

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

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

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := setupClient()

			if !tc.failBeforeRequest {
				go func() {
					req := <-c.requests
					require.Equal(t, []string{GET, tc.key}, req.req)
					req.res <- []string{tc.response}
				}()
			}

			val, err := c.Get(tc.key)
			tc.check(t, val, err)
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

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := setupClient()

			if !tc.failBeforeRequest {
				go func() {
					req := <-c.requests
					require.Equal(t, []string{SET, tc.key, tc.val}, req.req)
					req.res <- []string{tc.response}
				}()
			}

			tc.check(t, c.Set(tc.key, tc.val))
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

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := setupClient()

			if !tc.failBeforeRequest {
				go func() {
					req := <-c.requests
					require.Equal(t, []string{DEL, tc.key}, req.req)
					req.res <- []string{tc.response}
				}()
			}

			tc.check(t, c.Del(tc.key))
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

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var c *Client

			if !tc.failBeforeRequest {
				c = setupClient()

				go func() {
					req := <-c.requests
					require.Equal(t, []string{PING}, req.req)
					req.res <- []string{tc.response}
				}()
			} else {
				c = &Client{}
			}

			tc.check(t, c.Ping())
		})
	}
}
