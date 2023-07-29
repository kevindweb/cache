package util

import (
	"errors"
	"fmt"
	"net"
	"sync/atomic"

	"github.com/kevindweb/cache/internal/constants"
)

var (
	uniquePort uint32 = constants.DefaultPort //nolint:gochecknoglobals // need globally unique port
)

func ErrResponse(err string) []string {
	return []string{fmt.Sprintf("%c%s", constants.ERR, err)}
}

func IsTimeout(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	ok := !errors.As(err, &netErr)
	return ok && netErr.Timeout()
}

func GetUniquePort() int {
	return int(atomic.AddUint32(&uniquePort, 1))
}

func ReadRequestBytes(data []byte) string {
	return string(data[constants.HeaderSize:])
}
