package main

import (
	"fmt"
	"log"
	"os"

	"github.com/kevindweb/cache/pkg/client"
	"github.com/kevindweb/cache/pkg/server"
	"github.com/kevindweb/cache/pkg/util"

	"github.com/google/uuid"
)

func main() {
	logger := log.New(os.Stdout, "", 0)
	defaultParameters(logger)
	customParameters(logger)
}

func defaultParameters(logger *log.Logger) {
	logger.Println("Starting default server and client")
	client, server, err := util.StartDefaultClientServer()
	if err != nil {
		panic(fmt.Errorf("error starting default client & server: %w", err))
	}

	defer cleanup(client, server)
	setOperation(logger, client)
}

func customParameters(logger *log.Logger) {
	host := "0.0.0.0"
	port := 8080
	network := "tcp"
	serverOptions := server.Options{
		Host:    host,
		Port:    port,
		Network: network,
	}
	customServer, err := server.StartOptions(serverOptions)
	if err != nil {
		panic(fmt.Errorf("error starting custom server: %w", err))
	}

	logger.Printf("Starting custom server at %s\n", customServer.Address)

	clientOptions := client.Options{
		Host:    host,
		Port:    port,
		Network: network,
	}
	customClient, err := client.StartOptions(clientOptions)
	if err != nil {
		panic(fmt.Errorf("error creating custom client: %w", err))
	}

	defer cleanup(customClient, customServer)
	setOperation(logger, customClient)
}

func setOperation(logger *log.Logger, client *client.Client) {
	logger.Println("Running example set operation")
	key := uuid.New().String()
	val := uuid.New().String()
	err := client.Set(key, val)
	if err != nil {
		panic(fmt.Errorf("error setting key (%s) and val (%s): %w", key, val, err))
	}
}

func cleanup(client *client.Client, server *server.Server) {
	if err := server.Stop(); err != nil {
		panic(fmt.Errorf("error stopping server: %w", err))
	}
	if err := client.Stop(); err != nil {
		panic(fmt.Errorf("error stopping client: %w", err))
	}
}
