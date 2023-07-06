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

	for _, args := range requests {
		if true {
			switch args[0] {
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
				return fmt.Errorf("action undefined: %s", args[0])
			}
		}
	}

	return nil
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
	// r.buf.WriteByte(Array)
	// r.buf.WriteString("1")
	// r.buf.WriteString(NewLine)
	// r.buf.WriteByte(Array)
	// r.buf.WriteString("1")
	// r.buf.WriteString(NewLine)
	r.buf.WriteByte(protocol.BulkString)
	r.buf.WriteString(strconv.Itoa(len(msg)))
	r.buf.WriteString(protocol.NewLine)
	r.buf.WriteString(msg)
	r.buf.WriteString(protocol.NewLine)
	r.out = r.buf.Bytes()
}
