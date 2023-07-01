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

	Array           byte   = '*'
	Error           byte   = '-'
	BulkString      byte   = '$'
	CarraigeReturn  byte   = '\r'
	CharacterLength int    = 1
	NewLine         string = "\r\n"
	NewLineLen      int    = 2

	ArgLength  int = 10
	BufferSize int = 256
)

const (
	DialTimeout   time.Duration = time.Second * 2
	ConnRetryWait time.Duration = time.Millisecond * 10

	MaxRequestBatch int = 10

	DefaultNetwork string = "tcp"
	DefaultHost    string = "localhost"
	DefaultPort    int    = 6379
)

const (
	InvalidAddrErr string = "address host:port are invalid"
	EmptyParamErr  string = "parameters cannot be empty on request"
	EmptyResErr    string = "empty response back from %s request"
	EmptyResArgErr string = "empty argument from %s request"

	ClientUninitializedErr string = "client was not initialized"
	ClientInitTimeoutErr   string = "timed out dialing %s for %s"
)
