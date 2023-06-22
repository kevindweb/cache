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

	DefaultNetwork string = "tcp"
)

const (
	InvalidAddrErr string = "address host:port are invalid"
	EmptyKeyErr    string = "key cannot be empty on request"
	EmptyValErr    string = "value cannot be empty on request"

	ClientUninitializedErr string = "client was not initialized"
	ClientInitTimeoutErr   string = "timed out dialing %s for %s"
)

type Client struct {
	url  string
	conn net.Conn
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

func (c *Client) Ping() error {
	if err := c.validateClient(); err != nil {
		return err
	}

	return nil
}

func (c *Client) Get(k string) (string, error) {
	var (
		err error
	)
	if err = c.validateGet(k); err != nil {
		return "", err
	}

	return "", nil
}

func (c *Client) validateGet(k string) error {
	if err := c.validateClient(); err != nil {
		return err
	}

	if k == "" {
		return errors.New(EmptyKeyErr)
	}

	return nil
}

func (c *Client) validateClient() error {
	if c.url == "" || c.conn == nil {
		return errors.New(ClientUninitializedErr)
	}

	return nil
}

func (c *Client) TearDown() error {
	err := c.conn.Close()
	return err
}

/*
thread 1 - main client, sends serial requests to server
- constantly waiting for requests
thread 2 - scheduler (deduplicates requests)
*/
