package redis

import (
	"fmt"
	"strings"
)

// func parseRequests(batch *string) ([][]string, error) {
// 	return nil, nil
// }

func parseRequests(batch *string) ([][]string, error) {
	var requests [][]string
	if batch == nil {
		return nil, nil
	}

	lines := strings.Split(*batch, "\n")
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		request := strings.Split(line, "\r\n")
		if len(request) < 2 {
			return nil, fmt.Errorf("invalid request: %s", line)
		}

		// Check if the first line is a valid command
		if !validCommand(request[0]) {
			return nil, fmt.Errorf("invalid command: %s", request[0])
		}

		requests = append(requests, request)
	}

	// Check if the batch is valid
	if len(requests) == 0 {
		return nil, fmt.Errorf("empty batch")
	}

	// Check if the number of commands in the batch matches the number of responses
	numCommands := len(requests[0])
	for _, request := range requests[1:] {
		if len(request) != numCommands {
			return nil, fmt.Errorf("invalid batch: number of commands does not match")
		}
	}

	return requests, nil
}

func validCommand(command string) bool {
	for _, validCommand := range []string{PING, ECHO, GET, DEL, SET} {
		if validCommand == command {
			return true
		}
	}

	return false
}
