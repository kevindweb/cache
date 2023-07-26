package server

import (
	"encoding/binary"
	"testing"

	"github.com/kevindweb/cache/internal/constants"

	"github.com/stretchr/testify/assert"
)

// import (
// 	"github.com/kevindweb/cache/pkg/protocol"
// 	"fmt"
// 	"io"
// 	"math/rand"
// 	"net"
// 	"sync"
// 	"testing"
// 	"time"

// 	"github.com/go-redis/redis"
// 	"github.com/google/go-cmp/cmp"
// 	"github.com/stretchr/testify/require"
// )

// var (
// 	portCounter int
// 	mu          sync.Mutex
// )

// func connect(port int) (net.Conn, error) {
// 	attempts := 0

// 	for {
// 		conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
// 		if err == nil {
// 			return conn, nil
// 		}

// 		if isTimeout(err) || attempts > 10 {
// 			return nil, err
// 		}

// 		attempts++
// 		time.Sleep(10 * time.Millisecond)
// 	}
// }

// func TestEcho(t *testing.T) {
// 	t.Parallel()
// 	port, close := setup(t)
// 	defer close()
// 	client := NewRedisClient(port)

// 	err := testEcho(client)
// 	require.NoError(t, err)
// }

// func testEcho(client *redis.Client) error {
// 	strings := [10]string{
// 		"hello",
// 		"world",
// 		"mangos",
// 		"apples",
// 		"oranges",
// 		"watermelons",
// 		"grapes",
// 		"pears",
// 		"horses",
// 		"elephants",
// 	}

// 	randomString := strings[rand.Intn(10)]
// 	resp, err := client.Echo(randomString).Result()
// 	if err != nil {
// 		return err
// 	}

// 	if resp != randomString {
// 		return fmt.Errorf("Expected %#v, got %#v", randomString, resp)
// 	}

// 	return client.Close()
// }

// func TestPingOnce(t *testing.T) {
// 	t.Parallel()
// 	port, close := setup(t)
// 	defer close()
// 	err := testPingPongOnce(port)
// 	require.NoError(t, err)
// }

// func testPingPongOnce(port int) error {
// 	conn, err := connect(port)
// 	if err != nil {
// 		return err
// 	}

// 	_, err = conn.Write([]byte("*1\r\n$4\r\nping\r\n"))
// 	if err != nil {
// 		return err
// 	}

// 	time.Sleep(100 * time.Millisecond) // Ensure we aren't reading partial responses

// 	var readBytes = make([]byte, 16)

// 	err = conn.SetReadDeadline(time.Now().Add(time.Second))
// 	if err != nil {
// 		return err
// 	}

// 	numberOfBytesRead, err := conn.Read(readBytes)
// 	if err != nil {
// 		return err
// 	}

// 	actual := string(readBytes[:numberOfBytesRead])
// 	expected1 := "+PONG\r\n"
// 	expected2 := "$4\r\nPONG\r\n"

// 	if actual != expected1 && actual != expected2 {
// 		return fmt.Errorf("expected response to be either %#v or %#v, got %#v", expected1, expected2, actual)
// 	}

// 	return nil
// }

// func TestPingPongMultiple(t *testing.T) {
// 	t.Parallel()
// 	port, close := setup(t)
// 	defer close()
// 	err := testPingPongMultiple(port)
// 	require.NoError(t, err)
// }

// func testPingPongMultiple(port int) error {
// 	client := NewRedisClient(port)
// 	for i := 1; i <= 3; i++ {
// 		if err := runPing(client, 1); err != nil {
// 			return err
// 		}
// 	}
// 	return client.Close()
// }

// func TestPingPongConcurrent(t *testing.T) {
// 	t.Parallel()
// 	port, close := setup(t)
// 	defer close()
// 	err := testPingPongConcurrent(port)
// 	require.NoError(t, err)
// }

// func testPingPongConcurrent(port int) error {
// 	client1 := NewRedisClient(port)
// 	if err := runPing(client1, 1); err != nil {
// 		return err
// 	}

// 	client2 := NewRedisClient(port)
// 	if err := runPing(client2, 2); err != nil {
// 		return err
// 	}

// 	if err := runPing(client1, 1); err != nil {
// 		return err
// 	}
// 	if err := runPing(client1, 1); err != nil {
// 		return err
// 	}
// 	if err := runPing(client2, 2); err != nil {
// 		return err
// 	}

// 	client1.Close()

// 	client3 := NewRedisClient(port)
// 	if err := runPing(client3, 3); err != nil {
// 		return err
// 	}

// 	client2.Close()
// 	return client3.Close()
// }

// func runPing(client *redis.Client, clientNum int) error {
// 	pong, err := client.Ping().Result()
// 	if err != nil {
// 		return err
// 	}

// 	if pong != "PONG" {
// 		return fmt.Errorf("client-%d: Expected \"PONG\", got %#v", clientNum, pong)
// 	}

// 	return nil
// }

// func TestGetSet(t *testing.T) {
// 	t.Parallel()
// 	port, close := setup(t)
// 	defer close()
// 	err := testGetSet(port)
// 	require.NoError(t, err)
// }

// func testGetSet(port int) error {
// 	client := NewRedisClient(port)
// 	strings := [10]string{
// 		"abcd",
// 		"defg",
// 		"heya",
// 		"heya",
// 		"heya",
// 		"heya",
// 		"heya",
// 		"heya",
// 		"heya",
// 		"heya",
// 	}

// 	randomKey := strings[rand.Intn(10)]
// 	randomValue := strings[rand.Intn(10)]

// 	resp, err := client.Set(randomKey, randomValue, 0).Result()
// 	if err != nil {
// 		return err
// 	}

// 	if resp != "OK" {
// 		return fmt.Errorf("Expected \"OK\", got %#v", resp)
// 	}

// 	resp, err = client.Get(randomKey).Result()
// 	if err != nil {
// 		return err
// 	}

// 	if resp != randomValue {
// 		return fmt.Errorf("Expected %#v, got %#v", randomValue, resp)
// 	}

// 	return client.Close()
// }

// func TestRedisSetKV(t *testing.T) {
// 	t.Parallel()
// 	port, close := setup(t)
// 	defer close()
// 	client := NewRedisClient(port)
// 	err := setGet(client)
// 	require.NoError(t, err)
// 	client.Close()
// }

// func setGet(client *redis.Client) error {
// 	k := "hello"
// 	v := "world"
// 	resp, err := client.Set(k, v, 0).Result()
// 	if err != nil {
// 		return err
// 	}

// 	if resp != "OK" {
// 		return fmt.Errorf("set Expected \"OK\", got %#v", resp)
// 	}

// 	resp, err = client.Get(k).Result()
// 	if err != nil {
// 		return err
// 	}

// 	if resp != v {
// 		return fmt.Errorf("get Expected %s, got %#v", v, resp)
// 	}

// 	return err
// }

// // func TestSetArray(t *testing.T) {
// // 	t.Parallel()
// // 	var buf bytes.Buffer
// // 	req := request{
// // 		req: "*3\r\n$3\r\nset\r\n$5\r\nworld\r\n$5\r\nhello\r\n",
// // 		kv:  map[string]string{},
// // 		buf: &buf,
// // 	}
// // 	err := req.process()
// // 	require.NoError(t, err)
// // 	require.Equal(t, "$2\r\nOK\r\n", string(req.out))
// // }

// func setup(t *testing.T) (int, func()) {
// 	port := uniquePort()
// 	s, err := NewServer(DefaultHost, port)
// 	require.NoError(t, err)
// 	go func() {
// 		err := s.Start()
// 		require.NoError(t, err)
// 	}()
// 	return port, s.Stop
// }

// func TestErrorHandling(t *testing.T) {
// 	t.Parallel()
// 	port, close := setup(t)
// 	defer close()
// 	conn, err := connect(port)
// 	require.NoError(t, err)

// 	_, err = conn.Write([]byte("*1\r\n$13\r\ninvalidaction\r\n"))
// 	require.NoError(t, err)

// 	time.Sleep(100 * time.Millisecond) // Ensure we aren't reading partial responses

// 	err = conn.SetReadDeadline(time.Now().Add(time.Second))
// 	require.NoError(t, err)
// 	var readBytes = make([]byte, 1024)
// 	numberOfBytesRead, err := conn.Read(readBytes)
// 	require.NoError(t, err)

// 	actual := string(readBytes[:numberOfBytesRead])
// 	expected := "-action undefined: invalidaction\r\n"
// 	if diff := cmp.Diff(expected, actual); diff != "" {
// 		t.Fatal(diff)
// 	}
// }

// func TestClientServerTeardown(t *testing.T) {
// 	t.Parallel()
// 	var (
// 		err error
// 	)
// 	s, err := NewServer(DefaultHost, DefaultPort)
// 	require.NoError(t, err)
// 	go func() {
// 		err := s.Start()
// 		require.NoError(t, err)
// 	}()

// 	c, err := NewClient(DefaultHost, DefaultPort)
// 	require.NoError(t, err)

// 	s.Stop()

// 	err = c.Ping()
// 	require.Equal(t, io.EOF, err)

// 	err = c.Stop()
// 	require.NoError(t, err)
// }

func TestDummy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		data   []byte
		buffer []byte
	}{
		{
			name: "",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
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
