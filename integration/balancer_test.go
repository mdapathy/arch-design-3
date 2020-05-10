package integration

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"math"
	"net/http"
	"sync"
	"testing"
	"time"
)

const baseAddress = "http://balancer:8090"

var client = http.Client{
	Timeout: 5 * time.Second,
}

var serversPool = []string{
	"http://server1:8080",
	"http://server2:8080",
	"http://server3:8080",
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
	amount := 100
	for i := 0; i < amount; i++ {
		wg.Add(1)
		go worker(&wg, client, &m, i, t)
	}
	wg.Wait()

	for _, val := range m.m {
		assert.LessOrEqual(t, math.Abs(float64(val-amount/len(serversPool))), float64(2))
	}

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

