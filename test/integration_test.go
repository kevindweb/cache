package test

import (
	"testing"

	"cache/pkg/client"
	"cache/pkg/server"
	"cache/pkg/util"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSetGetDel(t *testing.T) {
	t.Parallel()
	client, server, err := util.StartUniqueClientServer()
	assert.NoError(t, err, "unable to start client and server")
	defer cleanup(t, client, server)

	testKey := uuid.New().String()
	expectedVal := uuid.New().String()
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

	testKey := uuid.New().String()
	expectedVal := uuid.New().String()
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

func cleanup(t *testing.T, client *client.Client, server *server.Server) {
	err := server.Stop()
	assert.NoError(t, err, "failed to stop server")
	err = client.Stop()
	assert.NoError(t, err, "failed to stop client")
}
