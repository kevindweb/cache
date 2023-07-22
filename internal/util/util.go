package util

import (
	"fmt"

	"cache/internal/constants"
)

func ErrResponse(err string) []string {
	return []string{fmt.Sprintf("%c%s", constants.ERR, err)}
}
