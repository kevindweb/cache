package redis

import (
	"app/pkg/protocol"
	pb "app/pkg/protocol/github.com/kevindweb/proto"
	"errors"
	"fmt"
	"log"

	"github.com/tidwall/evio"
	"google.golang.org/protobuf/proto"
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

	batch := &pb.BatchedRequest{}
	if err := proto.Unmarshal(in, batch); err != nil {
		out = s.processErr(err)
		return
	}

	request := &request{
		req: batch.Operations,
		kv:  s.kv,
	}
	if err := request.process(); err != nil {
		out = s.processErr(err)
	} else {
		out = request.out
	}

	s.kv = request.kv

	return
}

func (s *Server) processErr(err error) []byte {
	errResponse := &pb.BatchedResponse{
		Results: []*pb.Result{
			{
				Status:  pb.Result_FAILURE,
				Message: err.Error(),
			},
		},
	}

	encoded, encodeErr := proto.Marshal(errResponse)
	if err != nil {
		msg := fmt.Sprintf("processing error: %v, encoding error: %v", err, encodeErr)
		s.logger.Printf(msg)
		return []byte(msg)
	}

	return encoded
}

type request struct {
	req []*pb.Operation
	out []byte
	kv  map[string]string
}

func (r *request) process() error {
	results := make([]*pb.Result, len(r.req))
	for i, op := range r.req {
		res := &pb.Result{}
		switch op.Type {
		case pb.Operation_PING:
			res.Message = PONG
		case pb.Operation_SET:
			res.Message = r.setResponse(op.Key, op.Value)
		case pb.Operation_GET:
			msg, err := r.getResponse(op.Key)
			if err != "" {
				res.Status = pb.Result_FAILURE
				res.Message = err
			} else {
				res.Message = msg
			}
		case pb.Operation_DELETE:
			res.Message = r.delResponse(op.Key)
		default:
			res.Status = pb.Result_FAILURE
			res.Message = fmt.Sprintf("action undefined: %s", op.Type)
		}
		results[i] = res
	}

	encoded, err := proto.Marshal(&pb.BatchedResponse{
		Results: results,
	})
	if err != nil {
		return err
	}

	r.out = encoded
	return nil
}

func errResponse(err string) []string {
	return []string{fmt.Sprintf("%c%s", protocol.Error, err)}
}

func (r *request) setResponse(key, value string) string {
	r.kv[key] = value
	return OK
}

func (r *request) getResponse(key string) (string, string) {
	var val string
	var ok bool
	if val, ok = r.kv[key]; !ok {
		return "", fmt.Sprintf("key %s not set", key)
	}
	return val, ""
}

func (r *request) delResponse(key string) string {
	delete(r.kv, key)
	return OK
}
