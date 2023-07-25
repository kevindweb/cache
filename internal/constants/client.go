package constants

import "time"

const (
	BaseWaitTime    = time.Microsecond * 500
	DialTimeout     = time.Second * 2
	ConnRetryWait   = time.Millisecond * 10
	ReadTimeout     = time.Second * 1
	ShutdownTimeout = time.Millisecond * 500

	DefaultNetwork = "tcp"
	DefaultHost    = "localhost"
	DefaultPort    = 6379

	MaxConnectionPool = 20
	MaxRequestBatch   = 200

	InvalidAddrErr = "address host:port are invalid"
	InvalidPortErr = "invalid configured port %d"
	EmptyParamErr  = "parameters cannot be empty on request"
	EmptyResErr    = "empty response back from %s request"
	EmptyResArgErr = "empty argument from %s request"

	ClientUninitializedErr = "client was not initialized"
	ClientInitTimeoutErr   = "timed out dialing %s for %s"

	UndefinedOpErr = "undefined operation: %s"
)
