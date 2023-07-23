package constants

const (
	RequestSizeBytes = 30
	HeaderSize       = 4
)

const (
	PONG = "PONG"
	OK   = "OK"

	ERR = '-'

	PING = "ping"
	ECHO = "echo"
	GET  = "get"
	DEL  = "del"
	SET  = "set"
)

var (
	PONG_B = []byte(PONG)
	OK_B   = []byte(OK)
)
