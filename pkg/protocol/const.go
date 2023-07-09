package protocol

const (
	PONG string = "PONG"
	OK   string = "OK"

	ERR string = "err"

	Array          byte   = '*'
	Error          byte   = '-'
	BulkString     byte   = '$'
	SimpleString   byte   = '+'
	CarraigeReturn byte   = '\r'
	DataTypeLength int    = 1
	NewLine        string = "\r\n"
	NewLineLen     int    = 2

	ArgLength  int = 10
	BufferSize int = 256
)

const (
	EmptyBatchResponseErr   string = "empty batch response from request"
	InvalidBatchResponseErr string = "response datatype %s is not implemented at inx %d: %s"
)
