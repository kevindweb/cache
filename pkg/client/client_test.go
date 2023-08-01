package client

import (
	"testing"

	"github.com/kevindweb/cache/internal/constants"
	"github.com/kevindweb/cache/internal/protocol"
	"github.com/stretchr/testify/require"
)

func setupClient() *Client {
	return &Client{
		requests: make(chan clientReq),
		workers:  []Worker{{}},
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
				require.Equal(t, constants.EmptyParamErr, err.Error())
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
					getOperation := protocol.Operation{
						Type: protocol.GET,
						Key:  []byte(tc.key),
					}
					require.Equal(t, getOperation, req.req)
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
			response: constants.OK,
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
				require.Equal(t, constants.EmptyParamErr, err.Error())
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
					setOperation := protocol.Operation{
						Type:  protocol.SET,
						Key:   []byte(tc.key),
						Value: []byte(tc.val),
					}
					require.Equal(t, setOperation, req.req)
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
			response: constants.OK,
			check: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:     "expect error on invalid del response",
			key:      "delete",
			response: constants.PING,
			check: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
		{
			name:              "empty key",
			failBeforeRequest: true,
			check: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Equal(t, constants.EmptyParamErr, err.Error())
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
					delOperation := protocol.Operation{
						Type: protocol.DELETE,
						Key:  []byte(tc.key),
					}
					require.Equal(t, delOperation, req.req)
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
			response: constants.PONG,
			check: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:     "server responds PING expect error",
			response: constants.PING,
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
					pingOperation := protocol.Operation{
						Type: protocol.PING,
					}
					require.Equal(t, pingOperation, req.req)
					req.res <- []string{tc.response}
				}()
			} else {
				c = &Client{}
			}
			tc.check(t, c.Ping())
		})
	}
}

func TestDeduplication(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name           string
		batch          []protocol.Operation
		wantOperations []protocol.Operation
		wantIndex      map[int][]int
	}{
		{
			name: "duplicate operations",
			batch: []protocol.Operation{
				{
					Type: protocol.GET,
					Key:  []byte("hello"),
				},
				{
					Type: protocol.GET,
					Key:  []byte("hello"),
				},
				{
					Type: protocol.DELETE,
					Key:  []byte("key"),
				},
				{
					Type:  protocol.SET,
					Key:   []byte("bye"),
					Value: []byte("set"),
				},
				{
					Type: protocol.GET,
					Key:  []byte("hello"),
				},
				{
					Type: protocol.GET,
					Key:  []byte("hello"),
				},
				{
					Type:  protocol.SET,
					Key:   []byte("bye"),
					Value: []byte("set"),
				},
			},
			wantOperations: []protocol.Operation{
				{
					Type: protocol.GET,
					Key:  []byte("hello"),
				},
				{
					Type: protocol.DELETE,
					Key:  []byte("key"),
				},
				{
					Type:  protocol.SET,
					Key:   []byte("bye"),
					Value: []byte("set"),
				},
			},
			wantIndex: map[int][]int{
				0: {0, 1, 4, 5},
				1: {2},
				2: {3, 6},
			},
		},
		{
			name: "no duplicates",
			batch: []protocol.Operation{
				{
					Type: protocol.DELETE,
					Key:  []byte("hello"),
				},
				{
					Type:  protocol.SET,
					Key:   []byte("hello"),
					Value: []byte("hi"),
				},
			},
			wantOperations: []protocol.Operation{
				{
					Type: protocol.DELETE,
					Key:  []byte("hello"),
				},
				{
					Type:  protocol.SET,
					Key:   []byte("hello"),
					Value: []byte("hi"),
				},
			},
			wantIndex: map[int][]int{
				0: {0},
				1: {1},
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotOperations, gotIndex := requestDeduplication(tc.batch)
			require.Equal(t, tc.wantOperations, gotOperations)
			require.Equal(t, tc.wantIndex, gotIndex)
		})
	}
}
