package main

import (
	"app/pkg/redis"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

func Main1() {
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

type Connection struct {
	ID        int
	createdAt time.Time
}

type ConnectionPool struct {
	connections chan *Connection
	// mu          sync.Mutex
}

func NewConnectionPool(size int) *ConnectionPool {
	pool := &ConnectionPool{
		connections: make(chan *Connection, size),
	}

	for i := 1; i <= size; i++ {
		pool.connections <- &Connection{
			ID:        i,
			createdAt: time.Now(),
		}
	}

	return pool
}

func (p *ConnectionPool) Acquire() *Connection {
	return <-p.connections
}

func (p *ConnectionPool) Release(conn *Connection) {
	p.connections <- conn
}

func Main2() {
	pool := NewConnectionPool(5)

	// Acquire connections
	conn1 := pool.Acquire()
	conn2 := pool.Acquire()
	conn3 := pool.Acquire()

	fmt.Println("Connections acquired:", conn1.ID, conn2.ID, conn3.ID)

	// Simulate some work with the connections
	time.Sleep(2 * time.Second)

	// Release connections
	pool.Release(conn1)
	pool.Release(conn2)
	pool.Release(conn3)

	// Acquire more connections
	conn4 := pool.Acquire()
	conn5 := pool.Acquire()

	fmt.Println("Connections acquired:", conn4.ID, conn5.ID)

	// Release connections
	pool.Release(conn4)
	pool.Release(conn5)
}

const (
	maxBatchSize = 10                     // Maximum number of requests in a batch
	baseWaitTime = 100 * time.Millisecond // Initial wait time for batching
)

type Request struct {
	ID int
}

type Batch struct {
	Requests []*Request
}

func processBatch(batch *Batch) time.Duration {
	fmt.Println("Processing batch with", len(batch.Requests), "requests")
	// Simulate batch processing
	time.Sleep(500 * time.Millisecond)
	fmt.Println("Batch processing completed")

	// Return the processing duration
	return 500 * time.Millisecond
}

func adjustWaitTime(duration time.Duration) time.Duration {
	// Modify the wait time based on the processing duration
	newWaitTime := baseWaitTime

	if duration > 500*time.Millisecond {
		newWaitTime = 200 * time.Millisecond
	} else if duration > 200*time.Millisecond {
		newWaitTime = 100 * time.Millisecond
	}

	fmt.Println("Adjusted wait time:", newWaitTime)

	return newWaitTime
}

func Main3() {
	var (
		mu    sync.Mutex
		batch *Batch
		timer *time.Timer
	)

	addRequest := func(req *Request) {
		mu.Lock()
		defer mu.Unlock()

		if batch == nil {
			batch = &Batch{}
			timer = time.AfterFunc(baseWaitTime, func() {
				mu.Lock()
				defer mu.Unlock()
				duration := processBatch(batch)
				batch = nil
				timer.Reset(adjustWaitTime(duration))
			})
		}

		batch.Requests = append(batch.Requests, req)

		if len(batch.Requests) >= maxBatchSize {
			timer.Stop()
			duration := processBatch(batch)
			batch = nil
			timer.Reset(adjustWaitTime(duration))
		}
	}

	// Simulate adding requests
	for i := 1; i <= 15; i++ {
		req := &Request{ID: i}
		addRequest(req)
	}

	// Wait for the last batch to process
	time.Sleep(baseWaitTime)

	fmt.Println("Done")
}

func parseRedisResponse(response string, index int) [][]string {
	var results [][]string
	var currentResponse []string
	var numArgs int
	var err error

	for i := index; i < len(response); i++ {
		if response[i] == '*' {
			// This is the start of a new multi-bulk response
			if numArgs != 0 {
				// There is an invalid response before this multi-bulk response
				return nil
			}
			numArgs, err = strconv.Atoi(string(response[i+1 : i+2]))
			if err != nil {
				fmt.Printf("invalid during array: %s\n", string(response[i+1:i+2]))
				return nil
			}

			currentResponse = []string{}
		} else if response[i] == '$' {
			// This is a single-bulk response
			if numArgs == 0 {
				// There is an invalid response before this single-bulk response
				return nil
			}

			length, err := strconv.Atoi(string(response[i+1 : i+2]))
			if err != nil {
				fmt.Printf("invalid: %s\n", string(response[i+1:i+2]))
				return nil
			}

			arg := strings.TrimSpace(response[i+3 : i+3+length])
			currentResponse = append(currentResponse, arg)
			numArgs--
		} else {
			// This is an invalid character
			return nil
		}
	}

	if numArgs != 0 {
		// There are not enough arguments in the response
		return nil
	}

	results = append(results, currentResponse)
	return results
}

func main() {
	response := "*2\r\n*1\r\n$2\r\nOK\r\n*3\r\n$3\r\nset\r\n$5\r\nworld\r\n$5\r\nhello\r\n"
	results := parseRedisResponse(response, 0)
	fmt.Println(results)
}
