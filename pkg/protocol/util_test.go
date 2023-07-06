package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProtocolParse(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		request  string
		expected [][]string
		wantErr  bool
	}{
		{
			name:    "completely invalid",
			request: "hello",
			wantErr: true,
		},
		{
			name:    "invalid error response",
			request: "-noNewLine",
			wantErr: true,
		},
		{
			name:    "invalid argument width",
			request: "*2\r\n*1\r\n$13\r\nnotenough\r\n*1\r\n$2\r\nOK\r\n",
			wantErr: true,
		},
		{
			name:    "invalid batch number",
			request: "*0\r\n*1\r\n$13\r\ntotallyvalids\r\n*1\r\n$2\r\nOK\r\n",
			wantErr: true,
		},
		{
			name:    "multi bulk string requires array",
			request: "$3\r\ndel\r\n$3\r\nkey\r\n",
			wantErr: true,
		},
		{
			name:     "single valid set request no bulk",
			request:  "*3\r\n$3\r\nSet\r\n$3\r\nkey\r\n$3\r\nval\r\n",
			expected: [][]string{{"Set", "key", "val"}},
		},
		{
			name:     "valid simple string",
			request:  "+simplestring\r\n",
			expected: [][]string{{"simplestring"}},
		},
		{
			name:     "valid simple error response",
			request:  "-this is an error\r\n",
			expected: [][]string{{"err", "this is an error"}},
		},
		{
			name:     "valid bulk error no bulk string",
			request:  "*1\r\n*1\r\n-bulkerr\r\n",
			expected: [][]string{{"err", "bulkerr"}},
		},
		{
			name:     "valid ok",
			request:  "*1\r\n*1\r\n$2\r\nOK\r\n",
			expected: [][]string{{"OK"}},
		},
		{
			name:     "many batched requests",
			request:  "*2\r\n*1\r\n$13\r\ntotallyvalids\r\n*1\r\n$2\r\nOK\r\n",
			expected: [][]string{{"totallyvalids"}, {"OK"}},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			requests, err := splitBatchResponse(tc.request)
			require.Equal(t, tc.wantErr, err != nil, "error expected: %t, got err: %v", tc.wantErr, err)
			if tc.wantErr {
				return
			}
			require.Equal(t, tc.expected, requests)
		})
	}
}

func TestProcessArg(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name           string
		response       string
		offset         int
		expectedResult []string
		expectedOffset int
		wantErr        bool
	}{
		{
			name:           "valid simple string",
			response:       "+simplestring\r\n",
			offset:         0,
			expectedResult: []string{"simplestring"},
			expectedOffset: 15,
			wantErr:        false,
		},
		{
			name:           "valid bulk string",
			response:       "$3\r\nkey\r\n",
			offset:         0,
			expectedResult: []string{"key"},
			expectedOffset: 9,
			wantErr:        false,
		},
		{
			name:           "valid error response",
			response:       "-this is an error\r\n",
			offset:         0,
			expectedResult: []string{"err", "this is an error"},
			expectedOffset: 19,
			wantErr:        false,
		},
		{
			name:           "invalid response",
			response:       "invalid\r\n",
			offset:         0,
			expectedResult: nil,
			expectedOffset: 0,
			wantErr:        true,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, offset, err := processArg(tc.response, tc.offset)
			require.Equal(t, tc.wantErr, err != nil, "error expected: %t, got err: %v", tc.wantErr, err)
			if tc.wantErr {
				return
			}
			require.Equal(t, tc.expectedResult, result)
			require.Equal(t, tc.expectedOffset, offset)
		})
	}
}

func TestParseLine(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name           string
		response       string
		start          int
		expectedResult string
		expectedOffset int
		wantErr        bool
	}{
		{
			name:           "valid line",
			response:       "line\r\n",
			start:          0,
			expectedResult: "line",
			expectedOffset: 6,
			wantErr:        false,
		},
		{
			name:           "incomplete line",
			response:       "incomplete",
			start:          0,
			expectedResult: "",
			expectedOffset: 0,
			wantErr:        true,
		},
		{
			name:           "empty response",
			response:       "",
			start:          0,
			expectedResult: "",
			expectedOffset: 0,
			wantErr:        true,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, offset, err := parseLine(tc.response, tc.start)
			require.Equal(t, tc.wantErr, err != nil, "error expected: %t, got err: %v", tc.wantErr, err)
			if tc.wantErr {
				return
			}
			require.Equal(t, tc.expectedResult, result)
			require.Equal(t, tc.expectedOffset, offset)
		})
	}
}

func TestParseNumber(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name           string
		response       string
		start          int
		expectedResult int
		expectedWidth  int
		wantErr        bool
	}{
		{
			name:           "valid number",
			response:       "123\r\n",
			start:          0,
			expectedResult: 123,
			expectedWidth:  3,
			wantErr:        false,
		},
		{
			name:           "invalid number",
			response:       "invalid\r\n",
			start:          0,
			expectedResult: 0,
			expectedWidth:  0,
			wantErr:        true,
		},
		{
			name:           "incomplete line",
			response:       "incomplete",
			start:          0,
			expectedResult: 0,
			expectedWidth:  0,
			wantErr:        true,
		},
		{
			name:           "empty response",
			response:       "",
			start:          0,
			expectedResult: 0,
			expectedWidth:  0,
			wantErr:        true,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			width, result, err := parseNumber(tc.response, tc.start)
			require.Equal(t, tc.wantErr, err != nil, "error expected: %t, got err: %v", tc.wantErr, err)
			if tc.wantErr {
				return
			}
			require.Equal(t, tc.expectedResult, result)
			require.Equal(t, tc.expectedWidth, width)
		})
	}
}
