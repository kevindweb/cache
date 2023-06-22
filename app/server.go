package main

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/tidwall/evio"
)

type request struct {
	req string
	out []byte
}

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

	Host string = "localhost"
	Port string = "6379"

	ArgLength  int = 10
	BufferSize int = 256
)

var (
	m map[string]string

	defaultRequest = &request{}
	arguments      = make([]string, ArgLength)

	errBuffer = bytes.NewBuffer(make([]byte, BufferSize))
	msgBuffer = bytes.NewBuffer(make([]byte, BufferSize))
)

func main() {
	var events evio.Events
	events.Data = eventHandler
	address := fmt.Sprintf("tcp://%s:%s", Host, Port)
	if err := evio.Serve(events, address); err != nil {
		panic(err.Error())
	}
}

func eventHandler(_ evio.Conn, in []byte) (out []byte, action evio.Action) {
	defaultRequest.req = string(in)
	if err := defaultRequest.process(); err != nil {
		out = processErr(err)
	} else {
		out = defaultRequest.out
	}

	return
}

func (r *request) process() error {
	dataType := r.req[0]
	switch dataType {
	case Array:
		return r.processArray()
	default:
		return fmt.Errorf("datatype %b not implemented", dataType)
	}
}

func (r *request) processArray() error {
	var (
		err     error
		numArgs int
	)

	c := r.req[1]
	numArgs, err = strconv.Atoi(string(c))
	if err != nil {
		return fmt.Errorf("failed to parse %b: %v", c, err)
	}

	args, err := r.parseArguments(numArgs)
	if err != nil {
		return err
	}

	action := args[0]
	switch action {
	case ECHO:
		r.processSimpleString(args[1])
	case PING:
		r.processSimpleString(PONG)
	case SET:
		r.setResponse(args[1], args[2])
	case GET:
		r.getResponse(args[1])
	case DEL:
		r.delResponse(args[1])
	default:
		return fmt.Errorf("action undefined: %s", action)
	}
	return nil
}

func (r *request) parseArguments(numArgs int) ([]string, error) {
	var (
		err      error
		argument string
		n        = 4
		empty    = []string{}
	)
	if numArgs > len(arguments) {
		arguments = make([]string, numArgs)
	}

	for i := 0; i < numArgs; i++ {
		argument, n, err = r.processArg(n)
		if err != nil {
			return empty, err
		}

		if n == 0 {
			return empty, fmt.Errorf(
				"expected %d args, broke after %d with request: %s",
				numArgs-1, i, r.req,
			)
		}

		arguments[i] = argument
	}
	return arguments, nil
}

func (r *request) setResponse(key, value string) {
	if m == nil {
		m = map[string]string{}
	}

	m[key] = value
	r.processSimpleString(OK)
}

func (r *request) getResponse(key string) {
	val := m[key]
	r.processSimpleString(val)
}

func (r *request) delResponse(key string) {
	delete(m, key)
	r.processSimpleString(OK)
}

func (r *request) processArg(start int) (string, int, error) {
	var (
		dataType      = r.req[start]
		width, length int
		err           error
	)
	switch dataType {
	case BulkString:
		if width, length, err = r.findNumber(start + 1); err != nil {
			return "", 0, err
		}

		offset := start + CharacterLength + NewLineLen + width
		s := r.req[offset : offset+length]
		return s, offset + length + NewLineLen, nil
	default:
		return "", 0, fmt.Errorf(
			"process (%s) not implemented at inx %d", string(dataType), start,
		)
	}
}

func (r *request) findNumber(start int) (int, int, error) {
	var i int
	for i = start + 1; r.req[i] != CarraigeReturn; i++ {
	}
	num := r.req[start:i]
	numWidth := len(num)
	length, err := strconv.Atoi(num)
	return numWidth, length, err
}

func (r *request) processSimpleString(msg string) {
	var (
		msgLength = strconv.Itoa(len(msg))
	)
	msgBuffer.Reset()
	msgBuffer.WriteByte(BulkString)
	msgBuffer.WriteString(msgLength)
	msgBuffer.WriteString(NewLine)
	msgBuffer.WriteString(msg)
	msgBuffer.WriteString(NewLine)
	r.out = msgBuffer.Bytes()
}

func processErr(err error) []byte {
	var (
		msg = err.Error()
	)
	errBuffer.Reset()
	errBuffer.WriteByte(Error)
	errBuffer.WriteString(msg)
	errBuffer.WriteString(NewLine)
	return errBuffer.Bytes()
}
