package util

import (
	"fmt"
	"net"
	"sync/atomic"

	"cache/internal/constants"
)

var (
	uniquePort uint32 = constants.DefaultPort
)

func ErrResponse(err string) []string {
	return []string{fmt.Sprintf("%c%s", constants.ERR, err)}
}

func IsTimeout(err error) bool {
	if err == nil {
		return false
	}
	netErr, ok := err.(net.Error)
	return ok && netErr.Timeout()
}

func GetUniquePort() int {
	return int(atomic.AddUint32(&uniquePort, 1))
}
