package redis

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/tidwall/evio"
)

type Server struct {
	Address  string
	shutdown bool
	logger   *log.Logger
	kv       map[string]string
}

func NewServer(host string, port int) (*Server, error) {
	if host == "" || port == 0 {
		return nil, errors.New(InvalidAddrErr)
	}

	return &Server{
		Address: fmt.Sprintf("%s://%s:%d", DefaultNetwork, host, port),
		logger:  log.New(nil, "", 0),
		kv:      map[string]string{},
	}, nil
}

func (s *Server) Start() error {
	events := evio.Events{
		Data: s.eventHandler,
	}
	return evio.Serve(events, s.Address)
}

func (s *Server) Stop() {
	s.shutdown = true
}

func (s *Server) eventHandler(c evio.Conn, in []byte) (out []byte, action evio.Action) {
	if s.shutdown {
		action = evio.Shutdown
		return
	}

	request := &request{
		req: string(in),
		kv:  s.kv,
	}
	if err := request.process(); err != nil {
		out = processErr(err)
	} else {
		out = request.out
	}

	s.kv = request.kv

	return
}

type request struct {
	req string
	out []byte
	kv  map[string]string
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
	numArgs := int(r.req[1] - '0')
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
	arguments := make([]string, numArgs)
	var err error

	n := 4
	for i := 0; i < numArgs; i++ {
		arguments[i], n, err = r.processArg(n)
		if err != nil {
			return nil, err
		}

		if n == 0 {
			return nil, fmt.Errorf("expected %d args, broke after %d with request: %s", numArgs-1, i, r.req)
		}
	}
	return arguments, nil
}

func (r *request) processArg(start int) (string, int, error) {
	dataType := r.req[start]
	switch dataType {
	case BulkString:
		width, length, err := r.findNumber(start + 1)
		if err != nil {
			return "", 0, err
		}

		offset := start + CharacterLength + NewLineLen + width
		s := r.req[offset : offset+length]
		return s, offset + length + NewLineLen, nil
	default:
		return "", 0, fmt.Errorf("process (%s) not implemented at inx %d", string(dataType), start)
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

func (r *request) setResponse(key, value string) {
	r.kv[key] = value
	r.processSimpleString(OK)
}

func (r *request) getResponse(key string) {
	val := r.kv[key]
	r.processSimpleString(val)
}

func (r *request) delResponse(key string) {
	delete(r.kv, key)
	r.processSimpleString(OK)
}

func (r *request) processSimpleString(msg string) {
	msgBuffer := bytes.NewBuffer([]byte{})
	msgBuffer.WriteByte(BulkString)
	msgBuffer.WriteString(strconv.Itoa(len(msg)))
	msgBuffer.WriteString(NewLine)
	msgBuffer.WriteString(msg)
	msgBuffer.WriteString(NewLine)
	r.out = msgBuffer.Bytes()
}

func processErr(err error) []byte {
	errBuffer := bytes.NewBuffer([]byte{})
	errBuffer.WriteByte(Error)
	errBuffer.WriteString(err.Error())
	errBuffer.WriteString(NewLine)
	return errBuffer.Bytes()
}
