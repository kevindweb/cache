package storage

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func getRandomString(length int) string {
	characters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	randomString := ""
	for i := 0; i < length; i++ {
		randomIndex := rand.Intn(len(characters))
		randomString += string(characters[randomIndex])
	}
	return randomString
}

func TestStorageSetGetDel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		key  string
		val  string
	}{
		{
			name: "large value",
			key:  "key",
			val:  getRandomString(100 * 100),
		},
		{
			name: "empty value",
			key:  "key",
			val:  "",
		},
		{
			name: "key and value the same",
			key:  "key",
			val:  "key",
		},
	}
	for _, tc := range tests {
		tc := tc
		key := []byte(tc.key)
		val := []byte(tc.val)
		for _, cache := range caches() {
			cache := cache
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				err := cache.Set(key, val)
				assert.NoError(t, err)
				got, err := cache.Get(key)
				assert.NoError(t, err)
				assert.Equal(t, tc.val, string(got))
				err = cache.Del(key)
				assert.NoError(t, err)
				_, err = cache.Get(key)
				assert.Error(t, err)
				err = cache.Free()
				assert.NoError(t, err)
			})
		}
	}
}
