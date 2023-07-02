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
	errBuf   *bytes.Buffer
	reqBuf   *bytes.Buffer
	args     []string
}

func NewServer(host string, port int) (*Server, error) {
	if host == "" || port == 0 {
		return nil, errors.New(InvalidAddrErr)
	}

	return &Server{
		Address: fmt.Sprintf("%s://%s:%d", DefaultNetwork, host, port),
		logger:  log.New(nil, "", 0),
		kv:      map[string]string{},
		errBuf:  bytes.NewBuffer(make([]byte, BufferSize)),
		reqBuf:  bytes.NewBuffer(make([]byte, BufferSize)),
		args:    make([]string, ArgLength),
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
		req:  string(in),
		kv:   s.kv,
		buf:  s.reqBuf,
		args: s.args,
	}
	if err := request.process(); err != nil {
		out = s.processErr(err)
	} else {
		out = request.out
	}

	s.kv = request.kv
	s.reqBuf = request.buf
	s.args = request.args

	return
}

func (s *Server) processErr(err error) []byte {
	s.errBuf.Reset()
	s.errBuf.WriteByte(Error)
	s.errBuf.WriteString(err.Error())
	s.errBuf.WriteString(NewLine)
	return s.errBuf.Bytes()
}

type request struct {
	req  string
	out  []byte
	kv   map[string]string
	buf  *bytes.Buffer
	args []string
}

func (r *request) process() error {
	dataType := r.req[0]
	switch dataType {
	case Array:
	default:
		return fmt.Errorf("datatype %b not implemented", dataType)
	}

	digitWidth, numRequests, err := r.findNumber(1)
	if err != nil {
		return err
	}

	for i := 0; i < numRequests; i++ {
		if true {
			return r.processArray(digitWidth + DataTypeLength + NewLineLen)
		}
	}

	return nil
}

func (r *request) processArray(start int) error {
	numArgs := int(r.req[start+1] - '0')
	err := r.parseArguments(start, numArgs)
	if err != nil {
		return err
	}

	args := r.args
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

func (r *request) parseArguments(offset, numArgs int) error {
	if numArgs > len(r.args) {
		r.args = make([]string, numArgs)
	}
	var err error

	n := 4 + offset
	for i := 0; i < numArgs; i++ {
		r.args[i], n, err = r.processArg(n)
		if err != nil {
			return err
		}

		if n == 0 {
			return fmt.Errorf("expected %d args, broke after %d with request: %s", numArgs-1, i, r.req)
		}
	}
	return nil
}

func (r *request) processArg(start int) (string, int, error) {
	dataType := r.req[start]
	switch dataType {
	case BulkString:
		width, length, err := r.findNumber(start + 1)
		if err != nil {
			return "", 0, err
		}

		offset := start + DataTypeLength + NewLineLen + width
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
	r.buf.Reset()
	r.buf.WriteByte(Array)
	r.buf.WriteString("1")
	r.buf.WriteString(NewLine)
	r.buf.WriteByte(Array)
	r.buf.WriteString("1")
	r.buf.WriteString(NewLine)
	r.buf.WriteByte(BulkString)
	r.buf.WriteString(strconv.Itoa(len(msg)))
	r.buf.WriteString(NewLine)
	r.buf.WriteString(msg)
	r.buf.WriteString(NewLine)
	r.out = r.buf.Bytes()
}
