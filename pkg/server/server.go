package server

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"

	"cache/internal/constants"
	"cache/internal/protocol"

	"github.com/tidwall/evio"
)

type Server struct {
	Address   string
	shutdown  bool
	logger    *log.Logger
	kv        map[string]string
	request   protocol.BatchedRequest
	requests  []protocol.Operation
	response  protocol.BatchedResponse
	resBuffer []byte
	outBuffer []byte
}

type Options struct {
	Host    string
	Port    int
	Network string
}

func fillDefaultOptions(opts *Options) Options {
	if opts == nil {
		opts = &Options{}
	}

	if opts.Host == "" {
		opts.Host = constants.DefaultHost
	}

	if opts.Port == 0 {
		opts.Port = constants.DefaultPort
	}

	if opts.Network == "" {
		opts.Host = constants.DefaultNetwork
	}

	return *opts
}

func NewServer(opts Options) (*Server, error) {
	opts = fillDefaultOptions(&opts)
	bufferSize := constants.MaxRequestBatch * constants.RequestSizeBytes
	return &Server{
		Address: fmt.Sprintf("%s://%s:%d", opts.Network, opts.Host, opts.Port),
		logger:  log.New(os.Stdout, "", 0),
		kv:      map[string]string{},
		request: protocol.BatchedRequest{
			Operations: make([]protocol.Operation, constants.MaxRequestBatch),
		},
		response: protocol.BatchedResponse{
			Results: make([]protocol.Result, 0, constants.MaxRequestBatch),
		},
		resBuffer: make([]byte, bufferSize),
		outBuffer: make([]byte, bufferSize),
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

	if _, err := (&s.request).UnmarshalMsg(in); err != nil {
		out = s.processErr(err)
		return
	}

	s.requests = s.request.Operations
	if err := s.process(); err != nil {
		out = s.processErr(err)
		return
	}

	out = s.writeLength(s.resBuffer)
	return
}

func (s *Server) writeLength(data []byte) []byte {
	dataLength := len(data)
	totalLength := constants.HeaderSize + dataLength
	if cap(s.outBuffer) < totalLength {
		s.outBuffer = make([]byte, totalLength)
	}
	binary.LittleEndian.PutUint32(s.outBuffer[:constants.HeaderSize], uint32(dataLength))
	copy(s.outBuffer[constants.HeaderSize:totalLength], data)
	return s.outBuffer[:totalLength]
}

func (s *Server) processErr(err error) []byte {
	s.response.Results = s.response.Results[:0]
	s.response.Results[0] = protocol.Result{
		Status:  protocol.FAILURE,
		Message: err.Error(),
	}

	var encodeErr error
	if s.resBuffer, encodeErr = s.response.MarshalMsg(s.resBuffer[:0]); err != nil {
		msg := fmt.Sprintf("processing error: %v, encoding error: %v", err, encodeErr)
		s.logger.Printf(msg)
		return []byte(msg)
	}

	return s.writeLength(s.resBuffer)
}

func (s *Server) process() error {
	results := s.response.Results[:0]
	if len(s.requests) > cap(results) {
		results = make([]protocol.Result, len(s.requests))
	}

	for _, op := range s.requests {
		res := protocol.Result{}
		switch op.Type {
		case protocol.PING:
			res.Message = constants.PONG
		case protocol.SET:
			res.Message = s.setResponse(op.Key, op.Value)
		case protocol.GET:
			msg, err := s.getResponse(op.Key)
			if err != "" {
				res.Status = protocol.FAILURE
				res.Message = err
			} else {
				res.Message = string(msg)
			}
		case protocol.DELETE:
			res.Message = s.delResponse(op.Key)
		default:
			res.Status = protocol.FAILURE
			res.Message = fmt.Sprintf("action undefined: %s", op.Type)
		}
		results = append(results, res)
	}

	var err error
	s.response.Results = results
	if s.resBuffer, err = s.response.MarshalMsg(s.resBuffer[:0]); err != nil {
		return err
	}

	return nil
}

func (s *Server) setResponse(key string, value string) string {
	s.kv[key] = value
	return constants.OK
}

func (s *Server) getResponse(key string) (string, string) {
	if val, ok := s.kv[key]; !ok {
		return "", fmt.Sprintf("key %s not set", key)
	} else {
		return val, ""
	}
}

func (s *Server) delResponse(key string) string {
	delete(s.kv, key)
	return constants.OK
}
