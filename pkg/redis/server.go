package redis

import (
	"app/pkg/protocol"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"os"

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

func NewServer(host string, port int) (*Server, error) {
	if host == "" || port == 0 {
		return nil, errors.New(InvalidAddrErr)
	}

	return &Server{
		Address: fmt.Sprintf("%s://%s:%d", DefaultNetwork, host, port),
		logger:  log.New(os.Stdout, "", 0),
		kv:      map[string]string{},
		request: protocol.BatchedRequest{
			Operations: make([]protocol.Operation, 10),
		},
		response:  protocol.BatchedResponse{},
		resBuffer: make([]byte, 18000),
		outBuffer: make([]byte, 18000),
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

	s.writeLength(s.resBuffer)
	out = s.resBuffer
	return
}

func (s *Server) writeLength(data []byte) {
	dataLength := len(data)
	totalLength := Uint32Size + dataLength
	if cap(s.outBuffer) < totalLength {
		s.outBuffer = make([]byte, totalLength)
	}
	binary.LittleEndian.PutUint32(s.resBuffer[0:Uint32Size], uint32(dataLength))
	copy(s.resBuffer[Uint32Size:totalLength], data)
	s.resBuffer = s.resBuffer[:totalLength]
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

	s.writeLength(s.resBuffer)
	return s.resBuffer
}

func (s *Server) process() error {
	results := s.response.Results
	if len(s.requests) > cap(results) {
		results = make([]protocol.Result, 0, len(s.requests))
	}

	for _, op := range s.requests {
		res := protocol.Result{}
		switch op.Type {
		case protocol.PING:
			res.Message = PONG
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

	s.response.Results = results

	var err error
	if s.resBuffer, err = s.response.MarshalMsg(s.resBuffer[:0]); err != nil {
		return err
	}

	s.response.Results = results[:0]
	return nil
}

func errResponse(err string) []string {
	return []string{fmt.Sprintf("%c%s", protocol.Error, err)}
}

func (s *Server) setResponse(key string, value string) string {
	s.kv[key] = value
	return OK
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
	return OK
}
