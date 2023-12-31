package util

import (
	"github.com/kevindweb/cache/internal/constants"
	"github.com/kevindweb/cache/internal/util"
	"github.com/kevindweb/cache/pkg/client"
	"github.com/kevindweb/cache/pkg/server"
)

func StartDefaultClientServer() (*client.Client, *server.Server, error) {
	s, err := server.StartDefault()
	if err != nil {
		return nil, nil, err
	}

	c, err := client.StartDefault()
	return c, s, err
}

func StartUniqueClientServer() (*client.Client, *server.Server, error) {
	port := util.GetUniquePort()
	serverOptions := server.Options{
		Host:    constants.DefaultHost,
		Port:    port,
		Network: constants.DefaultNetwork,
	}
	s, err := server.StartOptions(serverOptions)
	if err != nil {
		return nil, nil, err
	}

	clientOptions := client.Options{
		Host:    constants.DefaultHost,
		Port:    port,
		Network: constants.DefaultNetwork,
	}
	c, err := client.StartOptions(clientOptions)
	return c, s, err
}
