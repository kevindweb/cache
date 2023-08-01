package protocol

import (
	"strconv"
)

//go:generate msgp

type BatchedRequest struct {
	Operations []Operation `msg:"operations"`
}

type OperationType int

const (
	SET OperationType = iota
	GET
	DELETE
	PING
)

func (op OperationType) String() string {
	switch op {
	case SET:
		return "SET"
	case GET:
		return "GET"
	case DELETE:
		return "DELETE"
	case PING:
		return "PING"
	default:
		return strconv.Itoa(int(op))
	}
}

type Operation struct {
	Type  OperationType `msg:"type"`
	Key   []byte        `msg:"key"`
	Value []byte        `msg:"value"`
}

func (op Operation) Index() string {
	return op.Type.String() + "-" + string(op.Key) + "-" + string(op.Value)
}

type BatchedResponse struct {
	Results []Result `msg:"results"`
}

type ResultStatus int

const (
	SUCCESS ResultStatus = iota
	FAILURE
)

func (status ResultStatus) String() string {
	switch status {
	case SUCCESS:
		return "SUCCESS"
	case FAILURE:
		return "FAILURE"
	default:
		return strconv.Itoa(int(status))
	}
}

type Result struct {
	Status  ResultStatus `msg:"status"`
	Message []byte       `msg:"message"`
}
