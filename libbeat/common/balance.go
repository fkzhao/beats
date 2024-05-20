// Package common
package common

import (
	"sync"
)

type Server struct {
	Address string
}

type LoadBalancer struct {
	servers []Server
	next    int
	mu      sync.Mutex
}

func NewLoadBalancer(targets []string) *LoadBalancer {
	lb := &LoadBalancer{
		servers: make([]Server, len(targets)),
	}
	for i, target := range targets {
		lb.servers[i] = Server{
			Address: target,
		}
	}
	return lb
}

func (sp *LoadBalancer) GetNextServer() Server {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	server := sp.servers[sp.next]
	sp.next += 1
	if sp.next >= len(sp.servers) {
		sp.next = 0
	}
	return server
}
