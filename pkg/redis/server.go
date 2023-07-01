package redis

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/tidwall/evio"
)

type request struct {
	req string
	out []byte
	kv  map[string]string
}

type Server struct {
	Address  string
	shutdown bool
	log      *log.Logger
	request  *request
}

func NewServer(host string, port int) (*Server, error) {
	if host == "" || port == 0 {
		return nil, errors.New(InvalidAddrErr)
	}

	return &Server{
		Address: fmt.Sprintf("%s://%s:%d", DefaultNetwork, host, port),
		log:     &log.Logger{},
		request: &request{
			kv: map[string]string{},
		},
	}, nil
}

func (s *Server) Start() error {
	var events evio.Events
	events.Data = s.eventHandler
	return evio.Serve(events, s.Address)
}

func (s *Server) Stop() {
	s.shutdown = true
}

func (s *Server) eventHandler(_ evio.Conn, in []byte) (out []byte, action evio.Action) {
	if s.shutdown {
		action = evio.Shutdown
		return
	}

	s.request.req = string(in)
	if err := s.request.process(); err != nil {
		out = processErr(err)
	} else {
		out = s.request.out
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
		err       error
		argument  string
		n         = 4
		empty     = []string{}
		arguments = make([]string, numArgs)
	)

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
	if r.kv == nil {
		r.kv = map[string]string{}
	}

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
		msgBuffer bytes.Buffer
	)
	msgBuffer.WriteByte(BulkString)
	msgBuffer.WriteString(msgLength)
	msgBuffer.WriteString(NewLine)
	msgBuffer.WriteString(msg)
	msgBuffer.WriteString(NewLine)
	r.out = msgBuffer.Bytes()
}

func processErr(err error) []byte {
	var (
		msg       = err.Error()
		errBuffer bytes.Buffer
	)
	errBuffer.Reset()
	errBuffer.WriteByte(Error)
	errBuffer.WriteString(msg)
	errBuffer.WriteString(NewLine)
	return errBuffer.Bytes()
}
