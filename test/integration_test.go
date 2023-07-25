package test

import (
	"math/rand"
	"sync"
	"testing"

	"cache/pkg/client"
	"cache/pkg/server"
	"cache/pkg/util"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func cleanup(t *testing.T, client *client.Client, server *server.Server) {
	cleanupClient(t, client)
	cleanupServer(t, server)
}

func cleanupClient(t *testing.T, client *client.Client) {
	err := client.Stop()
	assert.NoError(t, err, "failed to stop client")
}

func cleanupServer(t *testing.T, server *server.Server) {
	err := server.Stop()
	assert.NoError(t, err, "failed to stop server")
}

func TestSetGetDel(t *testing.T) {
	t.Parallel()
	client, server, err := util.StartUniqueClientServer()
	assert.NoError(t, err, "unable to start client and server")
	defer cleanup(t, client, server)

	testKey := uuid.NewString()
	expectedVal := uuid.NewString()
	err = client.Set(testKey, expectedVal)
	assert.NoError(t, err)
	gotVal, err := client.Get(testKey)
	assert.NoError(t, err)
	assert.Equal(t, expectedVal, gotVal)
	err = client.Del(testKey)
	assert.NoError(t, err)
	_, err = client.Get(testKey)
	assert.Error(t, err)
}

func TestSetGetDelParallel(t *testing.T) {
	t.Parallel()
	client, server, err := util.StartUniqueClientServer()
	assert.NoError(t, err, "unable to start client and server")
	defer cleanup(t, client, server)

	keys := []string{}
	values := []string{}
	numKeys := 100
	for i := 0; i < numKeys; i++ {
		values = append(values, uuid.NewString())
		keys = append(keys, uuid.NewString())
	}

	var wg sync.WaitGroup
	wg.Add(numKeys)
	for i := 0; i < numKeys; i++ {
		i := i
		go func(i int) {
			err := client.Set(keys[i], values[i])
			assert.NoError(t, err)
			wg.Done()
		}(i)
	}
	wg.Wait()

	numOperations := 1000
	wg.Add(numOperations)
	for i := 0; i < numOperations; i++ {
		go func() {
			randomIndex := rand.Intn(numKeys)
			val, err := client.Get(keys[randomIndex])
			assert.NoError(t, err)
			assert.Equal(t, values[randomIndex], val)
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestAfterCleanup(t *testing.T) {
	t.Parallel()
	client, server, err := util.StartUniqueClientServer()
	assert.NoError(t, err, "unable to start client and server")
	cleanupServer(t, server)

	testKey := uuid.New().String()
	testVal := uuid.New().String()
	err = client.Set(testKey, testVal)
	assert.Error(t, err, "set should have failed when server is down")
	cleanupClient(t, client)
}
