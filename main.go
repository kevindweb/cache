package main

import (
	"fmt"
)

func main() {
	fmt.Println("hello")
}

// func main() {
// 	rand.Seed(time.Now().Unix())
// 	runBenchmark()
// }

// func runBenchmark() {
// 	server := startServer()
// 	defer server.Stop()

// 	client := createClient()
// 	defer func() {
// 		if err := client.Stop(); err != nil {
// 			panic(err)
// 		}
// 	}()

// 	benchmarkRandom(client)
// }

// func startServer() *redis.Server {
// 	s, err := redis.NewServer(redis.DefaultHost, redis.DefaultPort)
// 	if err != nil {
// 		panic(err)
// 	}

// 	go func() {
// 		err := s.Start()
// 		if err != nil {
// 			panic(err)
// 		}
// 	}()
// 	return s
// }

// func createClient() *redis.Client {
// 	c, err := redis.NewClient(redis.DefaultHost, redis.DefaultPort)
// 	if err != nil {
// 		panic(err)
// 	}

// 	return c
// }

// func benchmarkRandom(c *redis.Client) {
// 	set(c)

// 	var highest float64 = 0
// 	highestGroup := -1
// 	var lowest float64 = 10000
// 	lowestGroup := -1

// 	var wg sync.WaitGroup
// 	wg.Add(1)
// 	for i := 1; i < 10; i++ {
// 		now := time.Now()
// 		group := rand.Intn(100000) + 50
// 		executeRandom(group, c)
// 		duration := time.Since(now)
// 		ms := float64(duration.Nanoseconds()) / 1000000.0
// 		if ms > highest {
// 			highest = ms
// 			highestGroup = group
// 		}

// 		if ms < lowest {
// 			lowest = ms
// 			lowestGroup = group
// 		}
// 	}

// 	fmt.Printf("Lowest time: %fms with group %d\n", lowest, lowestGroup)
// 	fmt.Printf("Highest time: %fms with group %d\n", highest, highestGroup)
// }

// func executeRandom(n int, c *redis.Client) {
// 	var wg sync.WaitGroup
// 	fns := []func(*redis.Client){get, set}
// 	for i := 0; i < n; i++ {
// 		fn := fns[rand.Intn(len(fns))]
// 		wg.Add(1)
// 		go func() {
// 			fn(c)
// 			wg.Done()
// 		}()
// 	}
// 	wg.Wait()
// }

// func set(c *redis.Client) {
// 	if err := c.Set("key", "val"); err != nil {
// 		panic(err)
// 	}
// }

// func get(c *redis.Client) {
// 	if _, err := c.Get("key"); err != nil {
// 		panic(err)
// 	}
// }
