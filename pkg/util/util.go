package util

import (
	"cache/pkg/client"
	"cache/pkg/server"
)

func StartDefaultClientServer() (*client.Client, *server.Server, error) {
	s, err := server.StartDefault()
	if err != nil {
		return nil, nil, err
	}
	c, err := client.StartDefault()
	return c, s, err
}
