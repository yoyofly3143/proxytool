package node

import (
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"proxytool/internal/subscription"
)

// Result holds latency test result for a node
type Result struct {
	Node    subscription.Node
	Latency time.Duration
	Error   error
}

// TestAll tests all nodes concurrently and returns sorted results (fastest first)
func TestAll(nodes []subscription.Node, timeout time.Duration) []Result {
	results := make([]Result, len(nodes))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 50) // max 50 concurrent

	for i, n := range nodes {
		wg.Add(1)
		go func(idx int, node subscription.Node) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			latency, err := tcpPing(node.Server, node.Port, timeout)
			results[idx] = Result{Node: node, Latency: latency, Error: err}
		}(i, n)
	}
	wg.Wait()

	// Sort: reachable first (by latency), then unreachable
	sort.Slice(results, func(i, j int) bool {
		if results[i].Error != nil && results[j].Error != nil {
			return false
		}
		if results[i].Error != nil {
			return false
		}
		if results[j].Error != nil {
			return true
		}
		return results[i].Latency < results[j].Latency
	})
	return results
}

func tcpPing(server string, port int, timeout time.Duration) (time.Duration, error) {
	if server == "" || port == 0 {
		return 0, fmt.Errorf("invalid address")
	}
	addr := net.JoinHostPort(server, fmt.Sprintf("%d", port))
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return 0, err
	}
	conn.Close()
	return time.Since(start), nil
}

// BestNode returns the node with lowest latency
func BestNode(results []Result) *Result {
	for i := range results {
		if results[i].Error == nil {
			return &results[i]
		}
	}
	return nil
}
