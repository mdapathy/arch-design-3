package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
	"sync"

	"github.com/masha-mcr/arch-design-3/httptools"
	"github.com/masha-mcr/arch-design-3/signal"
)

const UintSize = 32 << (^uint(0) >> 32 & 1) // 32 or 64

const MaxInt  = 1<<(UintSize-1) - 1

type Server struct {
	connectionCount int
	isHealthy bool
}

type SafeServer struct {
	v   map[string]Server
	mux sync.Mutex
}

var (
	port = flag.Int("port", 8090, "load balancer port")
	timeoutSec = flag.Int("timeout-sec", 3, "request timeout time in seconds")
	https = flag.Bool("https", false, "whether backends support HTTPs")
	traceEnabled = flag.Bool("trace", false, "whether to include tracing information into responses")
)

var (
	timeout = time.Duration(*timeoutSec) * time.Second
	serversPool = []string{
		"server1:8080",
		"server2:8080",
		"server3:8080",
	}
	// serverCount = make(map[string]Server, len(serversPool))
	safeServer = SafeServer{v: make(map[string]Server, len(serversPool))}
)

func min(m map[string] Server) string {
	min := MaxInt
	k := ""
	for s, c := range m {
		if min >= c.connectionCount && c.isHealthy {
			k = s
			min = c.connectionCount
		}
	}
	return k
}

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

func forward(dst string, rw http.ResponseWriter, r *http.Request) error {
	safeServer.mux.Lock()
	s1 := safeServer.v[dst]
	s1.connectionCount++
	safeServer.v[dst] = s1
	safeServer.mux.Unlock()
	ctx, _ := context.WithTimeout(r.Context(), timeout)
	fwdRequest := r.Clone(ctx)
	fwdRequest.RequestURI = ""
	fwdRequest.URL.Host = dst
	fwdRequest.URL.Scheme = scheme()
	fwdRequest.Host = dst

	resp, err := http.DefaultClient.Do(fwdRequest)
	if err == nil {
		for k, values := range resp.Header {
			for _, value := range values {
				rw.Header().Add(k, value)
			}
		}
		if *traceEnabled {
			rw.Header().Set("lb-from", dst)
		}
		log.Println("fwd", resp.StatusCode, resp.Request.URL)
		rw.WriteHeader(resp.StatusCode)
		defer resp.Body.Close()
		_, err := io.Copy(rw, resp.Body)
		if err != nil {
			log.Printf("Failed to write response: %s", err)
		}
		safeServer.mux.Lock()
		s2 := safeServer.v[dst]
		s2.connectionCount--
		safeServer.v[dst] = s2
		safeServer.mux.Unlock()
		return nil
	} else {
		log.Printf("Failed to get response from %s: %s", dst, err)
		rw.WriteHeader(http.StatusServiceUnavailable)
		safeServer.mux.Lock()
		s2 := safeServer.v[dst]
		s2.connectionCount--
		safeServer.v[dst] = s2
		safeServer.mux.Unlock()
		return err
	}
}

func main() {
	flag.Parse()
	// TODO: Використовуйте дані про стан сервреа, щоб підтримувати список тих серверів, яким можна відправляти ззапит.
	for _, server := range serversPool {
		server := server
		//serverCount[server] = Server{0, true}
		safeServer.mux.Lock()
		safeServer.v[server] = Server{0, false}
		safeServer.mux.Unlock()
		go func() {
			for range time.Tick(10 * time.Second) {
				log.Println(server, health(server))
			}
		}()
		go func() {
			for range time.Tick(1 * time.Second) {
				safeServer.mux.Lock()
				s1 := safeServer.v[server]
				s1.isHealthy = health(server)
				safeServer.v[server] = s1
				safeServer.mux.Unlock()
			}
		}()
	}

	frontend := httptools.CreateServer(*port, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// TODO: Рееалізуйте свій алгоритм балансувальника.
		//log.Println(safeServer.v)
		forward(min(safeServer.v), rw, r)
	}))

	log.Println("Starting load balancer...")
	log.Printf("Tracing support enabled: %t", *traceEnabled)
	frontend.Start()
	signal.WaitForTerminationSignal()
}
