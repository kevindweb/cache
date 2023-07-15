package main

import (
	"app/pkg/redis"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

func main() {
	rand.Seed(time.Now().Unix())

	s, err := redis.NewServer(redis.DefaultHost, redis.DefaultPort)
	handleErr(err)

	go func() {
		err := s.Start()
		if err != nil {
			panic(err)
		}
	}()

	c, err := redis.NewClient(redis.DefaultHost, redis.DefaultPort)
	handleErr(err)

	set(c)

	var highest float64 = 0
	highestGroup := -1
	var lowest float64 = 10000
	lowestGroup := -1

	var wg sync.WaitGroup
	wg.Add(1)
	for i := 9999; i < 10000; i++ {
		now := time.Now()
		executeRandom(i, c)
		duration := time.Since(now)
		ms := float64(duration.Nanoseconds()) / 1000000.0
		if ms > highest {
			highest = ms
			highestGroup = i
		}

		if ms < lowest {
			lowest = ms
			lowestGroup = i
		}
	}

	fmt.Printf("Lowest time: %fms with group %d\n", lowest, lowestGroup)
	fmt.Printf("Highest time: %fms with group %d\n", highest, highestGroup)

	s.Stop()

	err = c.Stop()
	handleErr(err)
}

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}

func executeRandom(n int, c *redis.Client) {
	var wg sync.WaitGroup
	fns := []func(*redis.Client){get}
	// fns := []func(*redis.Client){ping, set, get}
	for i := 0; i < n; i++ {
		fn := fns[rand.Intn(len(fns))]
		wg.Add(1)
		go func() {
			fn(c)
			wg.Done()
		}()
	}
	wg.Wait()
}

func set(c *redis.Client) {
	if err := c.Set("key", "val"); err != nil {
		panic(err)
	}
}

func get(c *redis.Client) {
	if _, err := c.Get("key"); err != nil {
		panic(err)
	}
}

func ping(c *redis.Client) {
	if err := c.Ping(); err != nil {
		panic(err)
	}
}
