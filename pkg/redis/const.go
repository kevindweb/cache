package redis

import "time"

const (
	PONG string = "PONG"
	OK   string = "OK"

	PING string = "ping"
	ECHO string = "echo"
	GET  string = "get"
	DEL  string = "del"
	SET  string = "set"

	Uint32Size int = 4
)

const (
	DialTimeout   time.Duration = time.Second * 2
	ConnRetryWait time.Duration = time.Millisecond * 10

	DefaultNetwork string = "tcp"
	DefaultHost    string = "localhost"
	DefaultPort    int    = 6379
)

var (
	ReadTimeout time.Duration = time.Second * 1
)

const (
	MaxConnectionPool int           = 20
	MaxRequestBatch   int           = 200
	BaseWaitTime      time.Duration = 500 * time.Microsecond
)

const (
	InvalidAddrErr string = "address host:port are invalid"
	EmptyParamErr  string = "parameters cannot be empty on request"
	EmptyResErr    string = "empty response back from %s request"
	EmptyResArgErr string = "empty argument from %s request"

	ClientUninitializedErr string = "client was not initialized"
	ClientInitTimeoutErr   string = "timed out dialing %s for %s"
)
