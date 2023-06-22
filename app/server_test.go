package main

import (
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func NewRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:        "localhost:6379",
		DialTimeout: 1 * time.Second,
		Dialer:      connect,
	})
}

func connect() (net.Conn, error) {
	attempts := 0

	for {
		var err error
		var conn net.Conn

		conn, err = net.Dial("tcp", "localhost:6379")
		if err == nil {
			return conn, nil
		}

		// Already a timeout
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil, err
		}

		// 50 * 100ms = 5s
		if attempts > 10 {
			return nil, err
		}

		attempts += 1
		time.Sleep(100 * time.Millisecond)
	}
}

// Tests 'ECHO'
func TestEcho(t *testing.T) {
	t.Parallel()
	err := testEcho()
	require.NoError(t, err)
}

func testEcho() error {
	client := NewRedisClient()

	strings := [10]string{
		"hello",
		"world",
		"mangos",
		"apples",
		"oranges",
		"watermelons",
		"grapes",
		"pears",
		"horses",
		"elephants",
	}

	randomString := strings[rand.Intn(10)]
	fmt.Printf("Sending command: echo %s", randomString)
	resp, err := client.Echo(randomString).Result()
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	if resp != randomString {
		return fmt.Errorf("Expected %#v, got %#v", randomString, resp)
	}

	client.Close()

	return nil
}

func TestPingOnce(t *testing.T) {
	t.Parallel()
	err := testPingPongOnce()
	require.NoError(t, err)
}

func testPingPongOnce() error {
	conn, err := connect()
	if err != nil {
		return err
	}

	fmt.Println("Connection established, sending PING command (*1\\r\\n$4\\r\\nping\\r\\n)")

	_, err = conn.Write([]byte("*1\r\n$4\r\nping\r\n"))
	if err != nil {
		return err
	}

	time.Sleep(100 * time.Millisecond) // Ensure we aren't reading partial responses

	fmt.Println("Reading response...")

	var readBytes = make([]byte, 16)

	err = conn.SetReadDeadline(time.Now().Add(time.Second))
	if err != nil {
		return err
	}

	numberOfBytesRead, err := conn.Read(readBytes)
	if err != nil {
		return err
	}

	actual := string(readBytes[:numberOfBytesRead])
	expected1 := "+PONG\r\n"
	expected2 := "$4\r\nPONG\r\n"

	if actual != expected1 && actual != expected2 {
		return fmt.Errorf("expected response to be either %#v or %#v, got %#v", expected1, expected2, actual)
	}

	return nil
}

func TestPingPongMultiple(t *testing.T) {
	t.Parallel()
	err := testPingPongMultiple()
	require.NoError(t, err)
}

func testPingPongMultiple() error {
	client := NewRedisClient()

	for i := 1; i <= 3; i++ {
		if err := runPing(client, 1); err != nil {
			return err
		}
	}

	fmt.Printf("Success, closing connection...")
	client.Close()

	return nil
}

func TestPingPongConcurrent(t *testing.T) {
	t.Parallel()
	err := testPingPongConcurrent()
	require.NoError(t, err)
}

func testPingPongConcurrent() error {
	client1 := NewRedisClient()

	if err := runPing(client1, 1); err != nil {
		return err
	}

	client2 := NewRedisClient()
	if err := runPing(client2, 2); err != nil {
		return err
	}

	if err := runPing(client1, 1); err != nil {
		return err
	}
	if err := runPing(client1, 1); err != nil {
		return err
	}
	if err := runPing(client2, 2); err != nil {
		return err
	}

	fmt.Printf("client-%d: Success, closing connection...", 1)
	client1.Close()

	client3 := NewRedisClient()
	if err := runPing(client3, 3); err != nil {
		return err
	}

	fmt.Printf("client-%d: Success, closing connection...", 2)
	client2.Close()
	fmt.Printf("client-%d: Success, closing connection...", 3)
	client3.Close()

	return nil
}

func runPing(client *redis.Client, clientNum int) error {
	fmt.Printf("client-%d: Sending ping command...", clientNum)
	pong, err := client.Ping().Result()
	if err != nil {
		return err
	}

	fmt.Printf("client-%d: Received response.", clientNum)
	if pong != "PONG" {
		return fmt.Errorf("client-%d: Expected \"PONG\", got %#v", clientNum, pong)
	}

	return nil
}

func TestGetSet(t *testing.T) {
	t.Parallel()
	err := testGetSet()
	require.NoError(t, err)
}

func testGetSet() error {
	client := NewRedisClient()
	strings := [10]string{
		"abcd",
		"defg",
		"heya",
		"heya",
		"heya",
		"heya",
		"heya",
		"heya",
		"heya",
		"heya",
	}

	randomKey := strings[rand.Intn(10)]
	randomValue := strings[rand.Intn(10)]

	fmt.Printf("Setting key %s to %s\n", randomKey, randomValue)
	resp, err := client.Set(randomKey, randomValue, 0).Result()
	if err != nil {
		return err
	}

	if resp != "OK" {
		return fmt.Errorf("Expected \"OK\", got %#v", resp)
	}

	fmt.Printf("Getting key %s\n", randomKey)
	resp, err = client.Get(randomKey).Result()
	if err != nil {
		return err
	}

	if resp != randomValue {
		return fmt.Errorf("Expected %#v, got %#v", randomValue, resp)
	}

	fmt.Println("after close")
	client.Close()
	return nil
}

func TestRedisSetKV(t *testing.T) {
	client := NewRedisClient()
	err := setGet(client)
	require.NoError(t, err)
	client.Close()
}

func setGet(client *redis.Client) error {
	k := "hello"
	v := "world"
	resp, err := client.Set(k, v, 0).Result()
	if err != nil {
		return err
	}

	if resp != "OK" {
		return fmt.Errorf("set Expected \"OK\", got %#v", resp)
	}

	resp, err = client.Get(k).Result()
	if err != nil {
		return err
	}

	if resp != v {
		return fmt.Errorf("get Expected %s, got %#v", v, resp)
	}

	return err
}

func BenchmarkProcessSet(b *testing.B) {
	setCommand := []byte("*3\r\n$3\r\nset\r\n$5\r\nworld\r\n$5\r\nhello\r\n")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eventHandler(nil, setCommand)
	}
	b.StopTimer()
}

func TestSetArray(t *testing.T) {
	t.Parallel()
	req := request{
		req: "*3\r\n$3\r\nset\r\n$5\r\nworld\r\n$5\r\nhello\r\n",
	}
	err := req.process()
	require.NoError(t, err)
	require.Equal(t, "$2\r\nOK\r\n", string(req.out))
}

func TestFindNumber(t *testing.T) {
	t.Parallel()
	req := request{
		req: "$123456\r",
	}
	width, num, err := req.findNumber(1)
	require.NoError(t, err)
	require.Equal(t, 123456, num)
	require.Equal(t, 6, width)
}

func TestErrorHandling(t *testing.T) {
	conn, err := connect()
	require.NoError(t, err)

	_, err = conn.Write([]byte("*1\r\n$13\r\ninvalidaction\r\n"))
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond) // Ensure we aren't reading partial responses

	err = conn.SetReadDeadline(time.Now().Add(time.Second))
	require.NoError(t, err)
	var readBytes = make([]byte, 1024)
	numberOfBytesRead, err := conn.Read(readBytes)
	require.NoError(t, err)

	actual := string(readBytes[:numberOfBytesRead])
	expected := "-action undefined: invalidaction\r\n"
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Fatal(diff)
	}
}
