package client

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"cache/internal/constants"
	"cache/internal/protocol"
	"cache/internal/util"
)

type Client struct {
	workers  []Worker
	requests chan clientReq
}

type Options struct {
	Host    string
	Port    int
	Network string
}

func fillDefaultOptions(opts *Options) Options {
	if opts == nil {
		opts = &Options{}
	}

	if opts.Host == "" {
		opts.Host = constants.DefaultHost
	}

	if opts.Port == 0 {
		opts.Port = constants.DefaultPort
	}

	if opts.Network == "" {
		opts.Network = constants.DefaultNetwork
	}

	return *opts
}

func StartDefault() (*Client, error) {
	c, err := New(Options{})
	if err != nil {
		return nil, err
	}

	c.start()
	return c, c.Ping()
}

func New(opts Options) (*Client, error) {
	opts = fillDefaultOptions(&opts)
	maxClientRequests := constants.MaxRequestBatch * constants.MaxConnectionPool
	requests := make(chan clientReq, maxClientRequests)
	pool, err := createWorkers(requests, opts)
	if err != nil {
		return nil, err
	}

	return &Client{
		workers:  pool,
		requests: requests,
	}, nil
}

func createWorkers(
	requests chan clientReq, opts Options,
) ([]Worker, error) {
	addr := fmt.Sprintf("%s:%d", opts.Host, opts.Port)
	pool := make([]Worker, 0, constants.MaxConnectionPool)
	for i := 0; i < constants.MaxConnectionPool; i++ {
		conn, err := connectWithTimeout(addr, constants.DialTimeout, opts)
		if err != nil {
			return nil, err
		}

		worker := Worker{
			conn:     conn,
			shutdown: make(chan bool, 1),
			requests: requests,
		}
		pool = append(pool, worker)
	}
	return pool, nil
}

func connectWithTimeout(
	addr string, timeout time.Duration, opts Options,
) (net.Conn, error) {
	timedOut := time.Now().Add(timeout)
	for {
		conn, err := net.DialTimeout(opts.Network, addr, timeout)
		if err == nil {
			return conn, nil
		}

		if isTimeout(err) {
			return nil, err
		}

		if time.Now().After(timedOut) {
			return nil, fmt.Errorf(constants.ClientInitTimeoutErr, addr, timeout)
		}

		time.Sleep(constants.ConnRetryWait)
	}
}

func (c *Client) Ping() error {
	if err := c.validateClient(); err != nil {
		return err
	}

	return c.pingRequest()
}

type clientReq struct {
	req protocol.Operation
	res chan []string
}

func (c *Client) pingRequest() error {
	pingOp := protocol.Operation{
		Type: protocol.PING,
	}
	response := c.sendRequest(pingOp)
	return expectResponse(constants.PING, constants.PONG, response)
}

func (c *Client) sendRequest(op protocol.Operation) []string {
	resChan := make(chan []string, 1)
	c.requests <- clientReq{
		req: op,
		res: resChan,
	}
	return <-resChan
}

func errorResponse(command string, res []string) error {
	if len(res) == 0 {
		return fmt.Errorf(constants.EmptyResErr, command)
	}

	first := res[0]
	if len(first) == 0 {
		return fmt.Errorf(constants.EmptyResArgErr, command)
	}

	if first[0] != constants.ERR {
		return nil
	}

	return fmt.Errorf(first[1:])
}

func (c *Client) Get(key string) (string, error) {
	if err := c.validateParams(key); err != nil {
		return "", err
	}

	getOp := protocol.Operation{
		Type: protocol.GET,
		Key:  key,
	}
	response := c.sendRequest(getOp)
	return getResponse(key, response)
}

func getResponse(key string, res []string) (string, error) {
	if err := errorResponse(constants.GET, res); err != nil {
		return "", err
	}

	if len(res) != 1 || res[0] == "" {
		return "", fmt.Errorf(
			"expected value for key %s, received %d results: %v",
			key, len(res), res,
		)
	}

	return res[0], nil
}

func (c *Client) Set(key, val string) error {
	if err := c.validateParams(key, val); err != nil {
		return err
	}

	setOp := protocol.Operation{
		Type:  protocol.SET,
		Key:   key,
		Value: val,
	}
	response := c.sendRequest(setOp)
	return expectResponse(constants.SET, constants.OK, response)
}

func (c *Client) validateParams(params ...string) error {
	if err := c.validateClient(); err != nil {
		return err
	}

	for _, param := range params {
		if param == "" {
			return errors.New(constants.EmptyParamErr)
		}
	}

	return nil
}

func (c *Client) Del(key string) error {
	if err := c.validateParams(key); err != nil {
		return err
	}

	delOp := protocol.Operation{
		Type: protocol.DELETE,
		Key:  key,
	}
	response := c.sendRequest(delOp)
	return expectResponse(constants.DEL, constants.OK, response)
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
	if len(c.workers) == 0 {
		return errors.New(constants.ClientUninitializedErr)
	}

	return nil
}

func (c *Client) Stop() error {
	for _, worker := range c.workers {
		worker.shutdown <- true
		if err := worker.conn.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) start() {
	for _, worker := range c.workers {
		worker := worker
		go worker.scheduler()
	}
}

type Worker struct {
	conn     net.Conn
	shutdown chan bool
	requests chan clientReq
}

func (w *Worker) scheduler() {
	var (
		timer *time.Timer
		mu    sync.Mutex

		batch = &protocol.BatchedRequest{
			Operations: []protocol.Operation{},
		}
		requests = []clientReq{}
	)

	for {
		select {
		case <-w.shutdown:
			return
		case req := <-w.requests:
			mu.Lock()
			if len(batch.Operations) == 0 {
				timer = time.AfterFunc(constants.BaseWaitTime, func() {
					mu.Lock()
					defer mu.Unlock()
					w.processBatch(batch, requests)
					batch.Operations = []protocol.Operation{}
					requests = []clientReq{}
					timer.Reset(constants.BaseWaitTime)
				})
			}
			w.processNewRequest(req, batch, &requests, timer)
			mu.Unlock()
		}
	}
}

func (w *Worker) processNewRequest(
	req clientReq,
	batch *protocol.BatchedRequest,
	requests *[]clientReq,
	timer *time.Timer,
) {
	batch.Operations = append(batch.Operations, req.req)
	*requests = append(*requests, req)
	if len(batch.Operations) < constants.MaxRequestBatch {
		return
	}

	timer.Stop()
	w.processBatch(batch, *requests)
	batch.Operations = batch.Operations[:0]
	(*requests) = []clientReq{}
	timer.Reset(constants.BaseWaitTime)
}

func (w *Worker) processBatch(
	batch *protocol.BatchedRequest, requests []clientReq,
) {
	if len(requests) == 0 {
		return
	}

	encoded, err := batch.MarshalMsg(nil)
	if err != nil {
		batchError(err, requests)
		return
	}

	_, err = w.conn.Write(encoded)
	if err != nil {
		batchError(err, requests)
		return
	}

	responseBytes, err := readFromConnection(w.conn, constants.ReadTimeout)
	if err != nil {
		batchError(err, requests)
		return
	}

	batchResponse := &protocol.BatchedResponse{}
	if _, err := batchResponse.UnmarshalMsg(responseBytes); err != nil {
		batchError(err, requests)
		return
	}

	responses := batchResponse.Results

	if len(responses) != len(requests) {
		if len(responses) == 1 {
			err = fmt.Errorf(
				"received 1 response: (%s) %s, requests: %v",
				responses[0].Status,
				responses[0].Message,
				batch.Operations[0],
			)
		} else {
			err = fmt.Errorf(
				"expected %d responses, received %d",
				len(requests), len(responses),
			)
		}
		batchError(err, requests)
		return
	}

	propagateBatch(responses, requests)
}

func batchError(err error, requests []clientReq) {
	bulkErr := util.ErrResponse(err.Error())
	for _, req := range requests {
		req.res <- bulkErr
	}
}

func propagateBatch(responses []protocol.Result, requests []clientReq) {
	for i, res := range responses {
		req := requests[i]
		if res.Status == protocol.FAILURE {
			req.res <- util.ErrResponse(res.Message)
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

	responseLengthBytes := make([]byte, constants.HeaderSize)
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
