package protocol

import "fmt"

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
		return "SET"
	case PING:
		return "PING"
	default:
		return fmt.Sprintf("%d", int(op))
	}
}

type Operation struct {
	Type  OperationType `msg:"type"`
	Key   string        `msg:"key"`
	Value string        `msg:"value"`
}

type BatchedResponse struct {
	Results []Result `msg:"results"`
}

type ResultStatus int

const (
	SUCCESS ResultStatus = iota
	FAILURE
)

type Result struct {
	Status  ResultStatus `msg:"status"`
	Message string       `msg:"message"`
}
