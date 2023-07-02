package redis

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"
)

type Client struct {
	url      string
	conn     net.Conn
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

	c.initChannels()
	c.start()

	if err := c.Ping(); err != nil {
		return nil, err
	}

	return c, nil
}

func connectWithTimeout(addr string, timeout time.Duration) (net.Conn, error) {
	timedOut := time.Now().Add(timeout)
	for {
		conn, err := net.DialTimeout(DefaultNetwork, addr, timeout)
		if err == nil {
			return conn, nil
		}

		if isTimeout(err) {
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
	c.requests = make(chan clientReq, MaxRequestBatch)
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

func createMessage(args []string) []byte {
	var builder bytes.Buffer

	if len(args) == 0 {
		return []byte{}
	}

	builder.WriteByte(Array)
	builder.WriteString("1")
	builder.WriteString(NewLine)
	builder.WriteByte(Array)
	builder.WriteString(strconv.Itoa(len(args)))
	builder.WriteString(NewLine)
	for _, arg := range args {
		builder.WriteByte(BulkString)
		msgLength := strconv.Itoa(len(arg))
		builder.WriteString(msgLength)
		builder.WriteString(NewLine)
		builder.WriteString(arg)
		builder.WriteString(NewLine)
	}
	return builder.Bytes()
}

func (c *Client) Stop() error {
	err := c.conn.Close()
	if err != nil {
		return err
	}

	c.shutdown <- true
	return nil
}

func (c *Client) start() {
	go c.scheduler()
}

func (c *Client) scheduler() {
	for {
		select {
		case <-c.shutdown:
			return
		case req := <-c.requests:
			data := createMessage(req.req)
			_, err := c.conn.Write(data)
			if err != nil {
				req.res <- []string{string(Error) + err.Error()}
				continue
			}

			timeout := time.Second * 1
			batchResponse, err := readFromConnection(c.conn, timeout)
			if err != nil && !isTimeout(err) {
				req.res <- []string{string(Error) + err.Error()}
				return
			}

			responses, err := splitBatchResponse(string(batchResponse))
			if err != nil {
				req.res <- []string{string(Error) + err.Error()}
				return
			}

			fmt.Println("Response:", responses)

			req.res <- responses[0]
		}
	}
}

func splitBatchResponse(batchResponse string) ([][]string, error) {
	if len(batchResponse) == 0 {
		return nil, fmt.Errorf(EmptyBatchResponseErr)
	}

	dataType := batchResponse[0]
	switch dataType {
	case Array:
	default:
		return nil, fmt.Errorf(InvalidBatchResponseErr, string(dataType), 0, batchResponse)
	}

	digitWidth, batchLength, err := parseNumber(batchResponse, 1)
	if err != nil {
		return nil, err
	}

	if batchLength == 0 {
		return nil, fmt.Errorf(EmptyBatchResponseErr)
	}

	start := DataTypeLength + digitWidth + NewLineLen
	var response []string

	batch := make([][]string, batchLength)
	for i := 0; i < batchLength; i++ {
		response, start, err = parseResponse(batchResponse, batchLength, start)
		if err != nil {
			return nil, err
		}
		batch[i] = response
	}

	return batch, nil
}

func parseResponse(batchResponse string, numArgs, start int) ([]string, int, error) {
	dataType := batchResponse[start]
	switch dataType {
	case Array:
		return processArray(batchResponse, start+1)
	case BulkString:
		return parseArguments(batchResponse, numArgs, start)
	default:
		return nil, 0, fmt.Errorf(InvalidBatchResponseErr, string(dataType), start, batchResponse)
	}
}

func processArray(response string, offset int) ([]string, int, error) {
	digitWidth, numArgs, err := parseNumber(response, offset)
	if err != nil {
		return nil, 0, err
	}

	args, offset, err := parseArguments(response, numArgs, offset+NewLineLen+digitWidth)
	if err != nil {
		return nil, 0, err
	}

	return args, offset, nil
}

func parseArguments(response string, numArgs, offset int) ([]string, int, error) {
	args := make([]string, numArgs)
	var err error

	for i := 0; i < numArgs; i++ {
		args[i], offset, err = processArg(response, offset)
		if err != nil {
			return nil, 0, err
		}

		if offset == 0 {
			return nil, 0, fmt.Errorf(
				"expected %d args, broke after %d with request: %s", numArgs-1, i, response,
			)
		}
	}
	return args, offset, nil
}

func processArg(response string, offset int) (string, int, error) {
	dataType := response[offset]
	switch dataType {
	case BulkString:
		width, length, err := parseNumber(response, offset+1)
		if err != nil {
			return "", 0, err
		}

		offset += DataTypeLength + NewLineLen + width
		s := response[offset : offset+length]
		return s, offset + length + NewLineLen, nil
	default:
		return "", 0, fmt.Errorf(InvalidBatchResponseErr, string(dataType), offset, response)
	}
}

func parseNumber(str string, start int) (int, int, error) {
	var i int
	for i = start + 1; str[i] != CarraigeReturn; i++ {
	}
	numStr := str[start:i]
	numWidth := len(numStr)
	num, err := strconv.Atoi(numStr)
	return numWidth, num, err
}

// func parseRedisResponse(response string) ([][]string, error) {
// 	var results [][]string

// 	lines := strings.Split(response, "\r\n")
// 	for _, line := range lines {
// 		if line[0] == Array {
// 			// This is a multi-bulk response
// 			count := strings.TrimSpace(line[1:])
// 			numArgs, err := strconv.Atoi(count)
// 			if err != nil {
// 				return nil, err
// 			}

// 			arguments := []string{}
// 			for i := 0; i < numArgs; i++ {
// 				arguments = append(arguments, strings.TrimSpace(lines[i+2]))
// 			}
// 			results = append(results, arguments)
// 		} else {
// 			// This is a single-bulk response
// 			results = append(results, []string{strings.TrimSpace(line)})
// 		}
// 	}

// 	return results, nil
// }

func readFromConnection(conn net.Conn, timeout time.Duration) ([]byte, error) {
	response := make([]byte, 0)

	err := conn.SetReadDeadline(time.Now().Add(timeout))
	if err != nil {
		return response, err
	}

	for {
		buffer := make([]byte, 1024) // TODO: Adjust the buffer size based on workload
		n, err := conn.Read(buffer)
		if err != nil {
			return response, err
		}

		if n == 0 {
			break
		}

		response = append(response, buffer[:n]...)
	}

	return response, nil
}

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	netErr, ok := err.(net.Error)
	return ok && netErr.Timeout()
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
