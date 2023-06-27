package redis

import (
	"errors"
	"fmt"
	"net"
	"time"
)

const (
	DialTimeout   time.Duration = time.Second * 2
	ConnRetryWait time.Duration = time.Millisecond * 10

	MaxBatch int = 10

	DefaultNetwork string = "tcp"
)

const (
	ErrorType string = "err"
)

const (
	InvalidAddrErr string = "address host:port are invalid"
	EmptyParamErr  string = "parameters cannot be empty on request"
	EmptyResErr    string = "empty response back from %s request"
	EmptyResArgErr string = "empty argument from %s request"

	ClientUninitializedErr string = "client was not initialized"
	ClientInitTimeoutErr   string = "timed out dialing %s for %s"
)

type Client struct {
	url  string
	conn net.Conn

	shutdown chan bool
	requests chan clientReq
}

func NewClient(host string, port int) (*Client, error) {
	if host == "" || port == 0 {
		return nil, errors.New(InvalidAddrErr)
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := connectWithTimeout(addr, DialTimeout)
	if err != nil {
		return nil, err
	}

	c := &Client{
		url:  addr,
		conn: conn,
	}

	if err := c.Ping(); err != nil {
		return nil, err
	}

	c.initChannels()
	// go c.scheduler()
	return c, nil
}

func connectWithTimeout(addr string, timeout time.Duration) (net.Conn, error) {
	var (
		err  error
		conn net.Conn

		timedOut = time.Now().Add(timeout)
	)
	for {
		if conn, err = net.Dial(DefaultNetwork, addr); err == nil {
			return conn, nil
		}

		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil, err
		}

		if time.Now().After(timedOut) {
			return nil, fmt.Errorf(ClientInitTimeoutErr, addr, timeout)
		}

		time.Sleep(ConnRetryWait)
	}
}

func (c *Client) initChannels() {
	c.shutdown = make(chan bool, 1)
}

func (c *Client) Ping() error {
	if err := c.validateClient(); err != nil {
		return err
	}

	return c.pingRequest()
}

type clientReq struct {
	req []string
	res chan []string
}

func (c *Client) pingRequest() error {
	response := c.sendRequest(PING)
	return expectResponse(PING, PONG, response)
}

func (c *Client) sendRequest(args ...string) []string {
	resChan := make(chan []string)
	c.requests <- clientReq{
		req: args,
		res: resChan,
	}
	return <-resChan
}

func errorResponse(command string, res []string) error {
	if len(res) == 0 {
		return fmt.Errorf(EmptyResErr, command)
	}

	first := res[0]
	if len(first) == 0 {
		return fmt.Errorf(EmptyResArgErr, command)
	}

	if first[0] != Error {
		return nil
	}

	return fmt.Errorf(first[1:])
}

func (c *Client) Get(key string) (string, error) {
	if err := c.validateParams(key); err != nil {
		return "", err
	}

	response := c.sendRequest(GET, key)
	return getResponse(key, response)
}

func getResponse(key string, res []string) (string, error) {
	if err := errorResponse(GET, res); err != nil {
		return "", err
	}

	if len(res) != 1 || res[0] == "" {
		return "", fmt.Errorf(
			"expected value for key %s, received %d results: %v", key, len(res), res,
		)
	}

	return res[0], nil
}

func (c *Client) Set(key, val string) error {
	if err := c.validateParams(key, val); err != nil {
		return err
	}

	response := c.sendRequest(SET, key, val)
	return expectResponse(SET, OK, response)
}

func (c *Client) validateParams(params ...string) error {
	if err := c.validateClient(); err != nil {
		return err
	}

	for _, param := range params {
		if param == "" {
			return errors.New(EmptyParamErr)
		}
	}

	return nil
}

func (c *Client) Del(key string) error {
	if err := c.validateParams(key); err != nil {
		return err
	}

	response := c.sendRequest(DEL, key)
	return expectResponse(DEL, OK, response)
}

func expectResponse(command, expected string, res []string) error {
	if err := errorResponse(command, res); err != nil {
		return err
	}

	if len(res) != 1 || res[0] != expected {
		return fmt.Errorf(
			"expected %s for %s, received %v", expected, command, res,
		)
	}

	return nil
}

func (c *Client) validateClient() error {
	if c.url == "" || c.conn == nil {
		return errors.New(ClientUninitializedErr)
	}

	return nil
}

// func createSimpleString(msg string) string {
// 	var (
// 		msgLength = strconv.Itoa(len(msg))
// 		builder   strings.Builder
// 	)
// 	builder.WriteByte(BulkString)
// 	builder.WriteString(msgLength)
// 	builder.WriteString(NewLine)
// 	builder.WriteString(msg)
// 	builder.WriteString(NewLine)
// 	return builder.String()
// }

func (c *Client) TearDown() error {
	err := c.conn.Close()
	if err != nil {
		return err
	}

	c.shutdown <- true
	return nil
}

// func (c *Client) scheduler() {
// 	bulkRequest, resChannels := c.aggregateRequests(true)
// 	for {
// 		select {
// 		case <-c.shutdown:
// 			fmt.Println("Shutting down scheduler")
// 			return
// 		default:
// 			c.conn.Write(bulkRequest)
// 			bulkRequest, resChannels = c.aggregateRequests(false)
// 			bulkResponse := c.read()
// 			c.partitionResponses(bulkResponse, resChannels)
// 		}
// 	}
// }

// func (c *Client) read() []byte {
// 	// c.conn.Read()
// 	return []byte{}
// }

// func (c *Client) aggregateRequests(blocking bool) ([]byte, [][]chan string) {
// 	seen := map[string]int{}
// 	resChannels := [][]chan string{}
// 	s := make([]byte, 0)

// 	var (
// 		req        clientReq
// 		requestInx int
// 	)

// 	if blocking {
// 		req = <-c.requests
// 	}

// 	for req = range c.requests {
// 		requestData := req.data
// 		duplicateInx, dup := seen[requestData]
// 		if !dup {
// 			seen[requestData] = requestInx

// 		} else {
// 			resChannels[duplicateInx] = append(resChannels[duplicateInx], req.res)
// 			resChannels = append(resChannels)
// 		}
// 		s = append(s, i)
// 		requestInx++
// 	}

// 	s = addBulkHeader(s, requestInx)
// 	return s, resChannels
// }

// func addBulkHeader(s []byte, numRequests int) []byte {
// 	var (
// 		buffer bytes.Buffer
// 	)

// 	buffer.WriteByte(Array)
// 	buffer.WriteString(strconv.Itoa(numRequests))
// 	buffer.WriteString(NewLine)
// 	buffer.Write(s)
// 	return buffer.Bytes()
// }

// func (c *Client) partitionResponses(responses []byte, channels [][]chan string) {

// }

// /*
// thread 1 - main client, sends serial requests to server
// - constantly waiting for requests
// thread 2 - scheduler (deduplicates requests)
// */
