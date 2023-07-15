package redis

import (
	"app/pkg/protocol"
	pb "app/pkg/protocol/github.com/kevindweb/proto"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
)

type Client struct {
	url      string
	conn     net.Conn
	buf      *bytes.Buffer
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
		url:      addr,
		conn:     conn,
		shutdown: make(chan bool, 1),
		requests: make(chan clientReq),
		buf:      bytes.NewBuffer(make([]byte, BufferSize)),
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
	req *pb.Operation
	res chan []string
}

func (c *Client) pingRequest() error {
	pingOp := &pb.Operation{
		Type: pb.Operation_PING,
	}
	response := c.sendRequest(pingOp)
	return expectResponse(PING, PONG, response)
}

func (c *Client) sendRequest(op *pb.Operation) []string {
	resChan := make(chan []string, 1)
	c.requests <- clientReq{
		req: op,
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

	if first[0] != protocol.Error {
		return nil
	}

	return fmt.Errorf(first[1:])
}

func (c *Client) Get(key string) (string, error) {
	if err := c.validateParams(key); err != nil {
		return "", err
	}

	getOp := &pb.Operation{
		Type: pb.Operation_GET,
		Key:  key,
	}
	response := c.sendRequest(getOp)
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

	setOp := &pb.Operation{
		Type:  pb.Operation_SET,
		Key:   key,
		Value: val,
	}
	response := c.sendRequest(setOp)
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

	delOp := &pb.Operation{
		Type: pb.Operation_DELETE,
		Key:  key,
	}
	response := c.sendRequest(delOp)
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
	var (
		timer *time.Timer
		mu    sync.Mutex

		batch = &pb.BatchedRequest{
			Operations: []*pb.Operation{},
		}
		requests = []clientReq{}
	)

	for {
		select {
		case <-c.shutdown:
			return
		case req := <-c.requests:
			mu.Lock()
			if len(batch.Operations) == 0 {
				timer = time.AfterFunc(BaseWaitTime, func() {
					mu.Lock()
					defer mu.Unlock()
					c.processBatch(batch, requests)
					batch.Operations = []*pb.Operation{}
					requests = []clientReq{}
					timer.Reset(BaseWaitTime)
				})
			}
			c.processNewRequest(req, batch, &requests, timer)
			mu.Unlock()
		}
	}
}

func (c *Client) processNewRequest(
	req clientReq, batch *pb.BatchedRequest, requests *[]clientReq, timer *time.Timer,
) {
	batch.Operations = append(batch.Operations, req.req)
	*requests = append(*requests, req)
	if len(batch.Operations) < MaxRequestBatch {
		return
	}

	timer.Stop()
	c.processBatch(batch, *requests)
	batch.Operations = batch.Operations[:0]
	(*requests) = []clientReq{}
	timer.Reset(BaseWaitTime)
}

func (c *Client) processBatch(batch *pb.BatchedRequest, requests []clientReq) {
	if len(requests) == 0 {
		return
	}

	encoded, err := proto.Marshal(batch)
	if err != nil {
		batchError(err, requests)
		return
	}

	_, err = c.conn.Write(encoded)
	if err != nil {
		batchError(err, requests)
		return
	}
	now := time.Now()

	responseBytes, err := readFromConnection(c.conn, ReadTimeout)
	if err != nil {
		batchError(err, requests)
		return
	}

	batchResponse := &pb.BatchedResponse{}
	err = proto.Unmarshal(responseBytes, batchResponse)
	if err != nil {
		batchError(err, requests)
		return
	}
	fmt.Println("waited", time.Since(now), len(requests), "requests")
	fmt.Println("size of request", float64(len(encoded))/1000.0)

	responses := batchResponse.Results

	if len(responses) != len(requests) {
		if len(responses) == 1 {
			err = fmt.Errorf(
				"received 1 response: (%s) %s, requests: %v",
				responses[0].Status, responses[0].Message, batch.Operations[0],
			)
		} else {
			err = fmt.Errorf(
				"expected %d responses, received %d", len(requests), len(responses),
			)
		}
		batchError(err, requests)
		return
	}

	propagateBatch(responses, requests)
}

func batchError(err error, requests []clientReq) {
	bulkErr := errResponse(err.Error())
	for _, req := range requests {
		req.res <- bulkErr
	}
}

func propagateBatch(responses []*pb.Result, requests []clientReq) {
	for i, res := range responses {
		req := requests[i]
		if res.Status == pb.Result_FAILURE {
			req.res <- errResponse(res.Message)
		} else {
			req.res <- []string{res.Message}
		}
	}
}

func readFromConnection(conn net.Conn, timeout time.Duration) ([]byte, error) {
	err := conn.SetReadDeadline(time.Now().Add(timeout))
	if err != nil {
		return []byte{}, err
	}

	responseLengthBytes := make([]byte, Uint32Size)
	_, err = conn.Read(responseLengthBytes)
	if err != nil {
		return []byte{}, err
	}

	responseLength := int(binary.LittleEndian.Uint32(responseLengthBytes))
	responseBytes := make([]byte, responseLength)
	_, err = conn.Read(responseBytes)
	if err != nil {
		return []byte{}, err
	}

	return responseBytes, nil
}

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	netErr, ok := err.(net.Error)
	return ok && netErr.Timeout()
}
