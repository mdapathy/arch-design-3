package server_heap

import (
	"container/heap"
	"sync"
)

type SafeServer struct {
	v   ServerHeap
	mux sync.Mutex
}

type Server struct {
	ServerName      string
	ConnectionCount int
	IsHealthy       bool
	index           int
}

type ServerHeap []*Server

func (h *ServerHeap) Len() int { return len(*h) }

func (h ServerHeap) Less(i, j int) bool {
	return (h[i].ConnectionCount < h[j].ConnectionCount && h[i].IsHealthy) || (h[i].IsHealthy && !h[j].IsHealthy)
}

func (h ServerHeap) Swap(i, j int) { 
	h[i], h[j] = h[j], h[i] 
	h[i].index = i
	h[j].index = j
}

func (h *ServerHeap) Push(x interface{}) {
	item := x.(*Server)
	item.index = len(*h)
	*h = append(*h, item)
}

func (h *ServerHeap) Decrease(item *Server) {
	item.ConnectionCount--;
	heap.Fix(h, item.index)
}


func (h *ServerHeap) Pop() interface{} {
	if len(*h) >= 1 {
		x := (*h)[len(*h)-1]
		x.index = -1
		*h = (*h)[0 : len(*h)-1]
		if x.IsHealthy {
			return x
		}
	}
	return &Server{}
}
