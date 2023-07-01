package main

import (
	"app/pkg/redis"
	"log"
)

func main() {
	var (
		err error
	)
	s, err := redis.NewServer(redis.DefaultHost, redis.DefaultPort)
	handleErr(err)

	go func() {
		err := s.Start()
		handleErr(err)
	}()

	c, err := redis.NewClient(redis.DefaultHost, redis.DefaultPort)
	handleErr(err)

	key := "key"
	val := "val"
	err = c.Set(key, val)
	handleErr(err)

	err = c.Del(key)
	handleErr(err)

	s.Stop()
}

func handleErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
