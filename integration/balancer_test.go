package integration

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

const baseAddress = "http://localhost:8090"

var client = http.Client{
	Timeout: 5 * time.Second,
}

var serversPool = []string{
	"http://localhost:8080",
	"http://localhost:8081",
	"http://localhost:8082",
}

type sum struct {
	mux sync.Mutex
	m   map[string]int
}

func worker(wg *sync.WaitGroup, client http.Client, m *sum, i int, t *testing.T) {
	defer wg.Done()
	resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
	if err != nil {
		t.Logf("Response error%s", err)
	}

	server := resp.Header.Get("lb-from")
	m.mux.Lock()
	m.m[server] += 1
	m.mux.Unlock()
}

func TestBalancer(t *testing.T) {
	m := sum{m: make(map[string]int)}
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go worker(&wg, client, &m, i, t)
	}
	wg.Wait()

}

func BenchmarkBalancer(b *testing.B) {
	var wg sync.WaitGroup
	for n := 0; n < b.N; n++ {
		wg.Add(1)
		go func(group sync.WaitGroup) {
			defer wg.Done()
			_, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
			if err != nil {
				b.Error(err)
			}
		}(wg)
	}
	wg.Wait()
}

func BenchmarkServer(b *testing.B) {
	var wg sync.WaitGroup
	for n := 0; n < b.N; n++ {
		wg.Add(1)
		go func(group sync.WaitGroup) {
			defer wg.Done()
			_, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", serversPool[n%3]))
			if err != nil {
				b.Error(err)
			}
		}(wg)
	}
	wg.Wait()
}
