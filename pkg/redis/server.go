package redis

import (
	"app/pkg/protocol"
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
	s.errBuf.WriteByte(protocol.Error)
	s.errBuf.WriteString(err.Error())
	s.errBuf.WriteString(protocol.NewLine)
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
	requests, err := protocol.SplitBatchResponse(r.req)
	if err != nil {
		return err
	}

	responses := make([][]string, len(requests))
	for i, args := range requests {
		var res []string
		switch args[0] {
		case ECHO:
			res = []string{args[1]}
		case PING:
			res = []string{PONG}
		case SET:
			res = r.setResponse(args[1], args[2])
		case GET:
			res = r.getResponse(args[1])
		case DEL:
			res = r.delResponse(args[1])
		default:
			res = errResponse(fmt.Sprintf("action undefined: %s", args[0]))
		}
		responses[i] = res
	}

	r.aggregateResponses(responses)

	return nil
}

func errResponse(err string) []string {
	return []string{fmt.Sprintf("%c%s", protocol.Error, err)}
}

func (r *request) setResponse(key, value string) []string {
	r.kv[key] = value
	return []string{OK}
}

func (r *request) getResponse(key string) []string {
	var val string
	var ok bool
	if val, ok = r.kv[key]; !ok {
		return errResponse(fmt.Sprintf("key %s not set", key))
	}
	return []string{val}
}

func (r *request) delResponse(key string) []string {
	delete(r.kv, key)
	return []string{OK}
}

func (r *request) aggregateResponses(responses [][]string) {
	r.buf.Reset()
	defer func() {
		r.out = r.buf.Bytes()
	}()

	if len(responses) == 1 {
		r.handleSingleResponse(responses[0])
		return
	}

	r.writeBulkPrefix(len(responses))
	for _, response := range responses {
		r.writeArguments(response)
	}
}

func (r *request) handleSingleResponse(response []string) {
	if len(response) == 1 {
		r.writeSimpleString(response[0])
		return
	}

	r.writeArguments(response)
}

func (r *request) writeBulkPrefix(count int) {
	r.buf.WriteByte(protocol.Array)
	r.buf.WriteString(strconv.Itoa(count))
	r.buf.WriteString(protocol.NewLine)
}

func (r *request) writeArguments(args []string) {
	r.writeBulkPrefix(len(args))
	for _, arg := range args {
		r.writeSimpleString(arg)
	}
}

func (r *request) writeSimpleString(msg string) {
	if msg[0] != protocol.Error {
		r.buf.WriteByte(protocol.BulkString)
		r.buf.WriteString(strconv.Itoa(len(msg)))
		r.buf.WriteString(protocol.NewLine)
	}
	r.buf.WriteString(msg)
	r.buf.WriteString(protocol.NewLine)
}
