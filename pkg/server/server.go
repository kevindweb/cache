package server

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"time"

	"cache/internal/constants"
	"cache/internal/protocol"
	"cache/internal/storage"

	"github.com/tidwall/evio"
)

type Server struct {
	Address   string
	shutdown  bool
	stopped   chan (bool)
	logger    *log.Logger
	kv        storage.KeyValue
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

func New(opts Options) (*Server, error) {
	opts = fillDefaultOptions(&opts)
	if err := validateOptions(opts); err != nil {
		return nil, err
	}

	bufferSize := constants.MaxRequestBatch * constants.RequestSizeBytes
	return &Server{
		Address: fmt.Sprintf("%s://%s:%d", opts.Network, opts.Host, opts.Port),
		stopped: make(chan bool),
		logger:  log.New(os.Stdout, "", 0),
		kv:      storage.NewCacheMap(),
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
		opts.Network = constants.DefaultNetwork
	}

	return *opts
}

func validateOptions(opts Options) error {
	if opts.Port <= 0 {
		return fmt.Errorf(constants.InvalidPortErr, opts.Port)
	}

	return nil
}

func StartDefault() (*Server, error) {
	return StartOptions(Options{})
}

func StartOptions(opts Options) (*Server, error) {
	s, err := New(opts)
	if err != nil {
		return nil, err
	}

	go func() {
		if startErr := s.Start(); startErr != nil {
			panic(startErr)
		}
	}()

	return s, nil
}

func (s *Server) Start() error {
	events := evio.Events{
		Data: s.eventHandler,
	}
	return evio.Serve(events, s.Address)
}

func (s *Server) Stop() error {
	s.shutdown = true
	select {
	case <-s.stopped:
	case <-time.After(constants.ShutdownTimeout):
	}
	return s.free()
}

func (s *Server) free() error {
	s.request = protocol.BatchedRequest{}
	s.response = protocol.BatchedResponse{}
	s.resBuffer = []byte{}
	s.outBuffer = []byte{}
	return s.kv.Free()
}

func (s *Server) eventHandler(_ evio.Conn, in []byte) ([]byte, evio.Action) {
	if s.shutdown {
		s.stopped <- true
		return []byte{}, evio.Shutdown
	}

	if _, err := (&s.request).UnmarshalMsg(in); err != nil {
		return s.processErr(err), evio.None
	}

	s.requests = s.request.Operations
	if err := s.process(); err != nil {
		return s.processErr(err), evio.None
	}

	return s.writeHeader(s.resBuffer), evio.None
}

func (s *Server) writeHeader(data []byte) []byte {
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
		Message: []byte(err.Error()),
	}

	var encodeErr error
	if s.resBuffer, encodeErr = s.response.MarshalMsg(s.resBuffer[:0]); err != nil {
		msg := fmt.Sprintf("processing error: %v, encoding error: %v", err, encodeErr)
		s.logger.Printf(msg)
		return []byte(msg)
	}

	return s.writeHeader(s.resBuffer)
}

func (s *Server) process() error {
	results := s.response.Results[:0]
	if len(s.requests) > cap(results) {
		results = make([]protocol.Result, 0, len(s.requests))
	}

	for _, op := range s.requests {
		res := s.processRequest(op)
		results = append(results, res)
	}

	var err error
	s.response.Results = results
	if s.resBuffer, err = s.response.MarshalMsg(s.resBuffer[:0]); err != nil {
		return err
	}

	return nil
}

func (s *Server) processRequest(op protocol.Operation) protocol.Result {
	res := protocol.Result{}
	switch op.Type {
	case protocol.PING:
		res.Message = []byte(constants.PONG)
	case protocol.SET:
		err := s.kv.Set(op.Key, op.Value)
		handleOperationResult(&res, []byte(constants.OK), err)
	case protocol.GET:
		val, err := s.kv.Get(op.Key)
		handleOperationResult(&res, val, err)
	case protocol.DELETE:
		err := s.kv.Del(op.Key)
		handleOperationResult(&res, []byte(constants.OK), err)
	default:
		res.Status = protocol.FAILURE
		res.Message = []byte(fmt.Sprintf(constants.UndefinedOpErr, op.Type))
	}
	return res
}

func handleOperationResult(res *protocol.Result, msg []byte, err error) {
	if err != nil {
		res.Status = protocol.FAILURE
		res.Message = []byte(err.Error())
	} else {
		res.Message = msg
	}
}
