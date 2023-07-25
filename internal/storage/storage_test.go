package storage

import (
	"math/rand"
	"testing"

	"github.com/google/go-cmp/cmp"
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
		for _, store := range Caches {
			store := store
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				store := store.New()
				err := store.Set(key, val)
				assert.NoError(t, err)
				got, err := store.Get(key)
				assert.NoError(t, err)
				if diff := cmp.Diff(tc.val, string(got)); diff != "" {
					t.Fatal(diff)
				}
				err = store.Del(key)
				assert.NoError(t, err)
				_, err = store.Get(key)
				assert.Error(t, err)
				err = store.Free()
				assert.NoError(t, err)
			})
		}
	}
}
