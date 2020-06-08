package main

import (
	"container/heap"
	"context"
	"flag"
	"fmt"
	"github.com/mdapathy/arch-design-3/cmd/lb/server-heap"
	"github.com/mdapathy/arch-design-3/httptools"
	"github.com/mdapathy/arch-design-3/signal"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type SafeServer struct {
	v   server_heap.ServerHeap
	mux sync.Mutex
}

var (
	port         = flag.Int("port", 8090, "load balancer port")
	timeoutSec   = flag.Int("timeout-sec", 3, "request timeout time in seconds")
	https        = flag.Bool("https", false, "whether backends support HTTPs")
	traceEnabled = flag.Bool("trace", false, "whether to include tracing information into responses")
)

var (
	timeout     = time.Duration(*timeoutSec) * time.Second
	serversPool = []string{
		"server1:8080",
		"server2:8080",
		"server3:8080",
	}

	safeServer = &SafeServer{v: make(server_heap.ServerHeap, 0, len(serversPool))}
)

func scheme() string {
	if *https {
		return "https"
	}
	return "http"
}

func health(dst string) bool {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	req, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s://%s/health", scheme(), dst), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	if resp.StatusCode != http.StatusOK {
		return false
	}
	return true
}

func forward(rw http.ResponseWriter, r *http.Request) error {
	safeServer.mux.Lock()
	dst := heap.Pop(&safeServer.v).(*server_heap.Server)
	if dst == nil {
		return fmt.Errorf("no healthy servers found")
	}
	dst.ConnectionCount++
	heap.Push(&safeServer.v, dst)
	safeServer.mux.Unlock()

	ctx, _ := context.WithTimeout(r.Context(), timeout)
	fwdRequest := r.Clone(ctx)
	fwdRequest.RequestURI = ""
	fwdRequest.URL.Host = dst.ServerName
	fwdRequest.URL.Scheme = scheme()
	fwdRequest.Host = dst.ServerName

	resp, err := http.DefaultClient.Do(fwdRequest)
	if err == nil {
		for k, values := range resp.Header {
			for _, value := range values {
				rw.Header().Add(k, value)
			}
		}
		if *traceEnabled {
			rw.Header().Set("lb-from", dst.ServerName)
		}
		log.Println("fwd", resp.StatusCode, resp.Request.URL)
		rw.WriteHeader(resp.StatusCode)
		defer resp.Body.Close()
		_, err = io.Copy(rw, resp.Body)
		if err != nil {
			log.Printf("Failed to write response: %s", err)
		}

	} else {
		log.Printf("Failed to get response from %s: %s", dst.ServerName, err)
		rw.WriteHeader(http.StatusServiceUnavailable)
	}

	safeServer.mux.Lock()
	//dst.ConnectionCount--
	//heap.Decrease(dst)
	safeServer.v.Decrease(dst)
	safeServer.mux.Unlock()
	return err
}

func main() {
	flag.Parse()

	heap.Init(&safeServer.v)
	for i, server := range serversPool {
		currentServer := server_heap.Server{ServerName: server, ConnectionCount: 0, IsHealthy: false}
		safeServer.mux.Lock()
		safeServer.v.Push(&currentServer)
		safeServer.mux.Unlock()
		go func() {
			for range time.Tick(10 * time.Second) {
				log.Println(server, health(server))
			}
		}()
		go func() {
			for range time.Tick(1 * time.Second) {
				safeServer.mux.Lock()
				currentServer.IsHealthy = health(server)
				heap.Fix(&safeServer.v, i)
				safeServer.mux.Unlock()
			}
		}()
	}

	frontend := httptools.CreateServer(*port, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		forward(rw, r)

	}))

	log.Println("Starting load balancer...")
	log.Printf("Tracing support enabled: %t", *traceEnabled)
	frontend.Start()
	signal.WaitForTerminationSignal()
}
