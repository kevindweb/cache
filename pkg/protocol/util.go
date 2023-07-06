package protocol

import (
	"fmt"
	"strconv"
)

func splitBatchResponse(batchResponse string) ([][]string, error) {
	if len(batchResponse) == 0 {
		return nil, fmt.Errorf(EmptyBatchResponseErr)
	}

	dataType := batchResponse[0]
	fmt.Printf("dataType: %s\n", string(dataType)) // Debugging statement
	switch dataType {
	case Array:
	case SimpleString:
		response, _, err := parseLine(batchResponse, 1)
		if err != nil {
			return nil, err
		}
		return [][]string{{response}}, nil
	case Error:
		errStr, _, err := parseLine(batchResponse, 1)
		if err != nil {
			return nil, err
		}
		return [][]string{{ERR, errStr}}, nil
	default:
		return nil, fmt.Errorf(InvalidBatchResponseErr, string(dataType), 0, batchResponse)
	}

	digitWidth, batchLength, err := parseNumber(batchResponse, 1)
	if err != nil {
		return nil, err
	}
	fmt.Printf("digitWidth: %d, batchLength: %d\n", digitWidth, batchLength) // Debugging statement

	if batchLength == 0 {
		return nil, fmt.Errorf(EmptyBatchResponseErr)
	}

	start := DataTypeLength + digitWidth + NewLineLen
	var response []string

	if batchResponse[start] != Array {
		response, _, err = parseResponse(batchResponse, batchLength, 0)
		if err != nil {
			return nil, err
		}
		return [][]string{response}, nil
	}

	batch := make([][]string, batchLength)
	for i := 0; i < batchLength; i++ {
		fmt.Println("happening", batch)
		fmt.Println("batch response", start, batchResponse[start:start+1])
		response, start, err = parseResponse(batchResponse, batchLength, start)
		if err != nil {
			return nil, err
		}
		batch[i] = response
	}

	return batch, nil
}

func parseResponse(batchResponse string, numArgs, start int) ([]string, int, error) {
	dataType := batchResponse[start]
	fmt.Printf("parseResponse: dataType: %s, start: %d\n", string(dataType), start) // Debugging statement
	switch dataType {
	case Array:
		return processArray(batchResponse, start+1)
	case BulkString:
		return parseArguments(batchResponse, numArgs, start)
	case SimpleString:
		str, until, err := parseLine(batchResponse, start+1)
		if err != nil {
			return nil, 0, err
		}
		return []string{str}, until, nil
	default:
		return nil, 0, fmt.Errorf(InvalidBatchResponseErr, string(dataType), start, batchResponse)
	}
}

func processArray(response string, offset int) ([]string, int, error) {
	digitWidth, numArgs, err := parseNumber(response, offset)
	if err != nil {
		return nil, 0, err
	}
	fmt.Printf("processArray: digitWidth: %d, numArgs: %d\n", digitWidth, numArgs) // Debugging statement

	args, offset, err := parseArguments(response, numArgs, offset+NewLineLen+digitWidth)
	if err != nil {
		return nil, 0, err
	}

	return args, offset, nil
}

func parseArguments(response string, numArgs, offset int) ([]string, int, error) {
	args := make([]string, numArgs)
	var err error
	var processedArgs []string

	for i := 0; i < numArgs; i++ {
		processedArgs, offset, err = processArg(response, offset)
		if err != nil {
			return nil, 0, err
		}

		if processedArgs[0] == ERR {
			args[0] = ERR
			args = append(args, processedArgs[1])
			break
		}

		if len(processedArgs) > 1 {
			return nil, 0, fmt.Errorf("should not be more than 1 arg: %v", processedArgs)
		}

		args[i] = processedArgs[0]

		if offset == 0 {
			return nil, 0, fmt.Errorf(
				"expected %d args, broke after %d with request: %s", numArgs-1, i, response,
			)
		}
	}
	return args, offset, nil
}

func processArg(response string, offset int) ([]string, int, error) {
	if offset >= len(response) {
		return nil, 0, fmt.Errorf("bad arguments, index %d > length %d", offset, len(response))
	}

	dataType := response[offset]
	fmt.Printf("processArg: dataType: %s, offset: %d\n", string(dataType), offset) // Debugging statement
	switch dataType {
	case BulkString:
		width, length, err := parseNumber(response, offset+1)
		if err != nil {
			return nil, 0, err
		}

		offset += DataTypeLength + NewLineLen + width
		s := response[offset : offset+length]
		return []string{s}, offset + length + NewLineLen, nil
	case SimpleString:
		str, until, err := parseLine(response, offset+1)
		if err != nil {
			return nil, 0, err
		}
		return []string{str}, until, nil
	case Error:
		errStr, until, err := parseLine(response, offset+1)
		if err != nil {
			return nil, 0, err
		}
		return []string{ERR, errStr}, until, nil
	default:
		return nil, 0, fmt.Errorf(InvalidBatchResponseErr, string(dataType), offset, response)
	}
}

func parseLine(response string, start int) (string, int, error) {
	until := start
	for ; until < len(response) && response[until] != '\r'; until++ {
	}

	if until >= len(response) || response[until] != '\r' {
		return "", 0, fmt.Errorf("line %s needs to end with %s", response[start:], NewLine)
	}

	return response[start:until], until + NewLineLen, nil
}

func parseNumber(str string, start int) (int, int, error) {
	var i int
	for i = start + 1; i < len(str) && str[i] != CarraigeReturn; i++ {
	}

	if i >= len(str) {
		return 0, 0, fmt.Errorf("failed to parse %s", str)
	}

	numStr := str[start:i]
	numWidth := len(numStr)
	num, err := strconv.Atoi(numStr)
	return numWidth, num, err
}
