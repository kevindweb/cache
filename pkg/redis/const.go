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
	ERR  string = "err"

	ArgLength  int = 10
	BufferSize int = 256
	Uint32Size int = 4
)

const (
	DialTimeout   time.Duration = time.Second * 2
	ConnRetryWait time.Duration = time.Millisecond * 10

	// max protobuf size is 2^16 bytes (7279 * 9 bytes/message)
	MaxRequestBatch int = 7200

	DefaultNetwork string = "tcp"
	DefaultHost    string = "localhost"
	DefaultPort    int    = 6379
)

var (
	ReadTimeout time.Duration = time.Second * 1
)

const (
	BaseWaitTime = 500 * time.Millisecond
)

const (
	InvalidAddrErr string = "address host:port are invalid"
	EmptyParamErr  string = "parameters cannot be empty on request"
	EmptyResErr    string = "empty response back from %s request"
	EmptyResArgErr string = "empty argument from %s request"

	ClientUninitializedErr string = "client was not initialized"
	ClientInitTimeoutErr   string = "timed out dialing %s for %s"
)
