// Command loadtest sends GET-only traffic to the article API and reports latency percentiles.
package main

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type result struct {
	requests      int
	connectionErr int
	statuses      map[int]int
	latencies     []time.Duration
}

func main() {
	baseURL := flag.String("url", "http://localhost:18080", "base URL of the API")
	workers := flag.Int("c", 1, "number of workers")
	duration := flag.Duration("d", 10*time.Second, "test duration")
	mix := flag.String("mix", "burst", "request mix: burst or heavy")
	flag.Parse()
	if *workers < 1 || *duration <= 0 {
		panic("-c must be positive and -d must be positive")
	}

	paths, err := requestPaths(*mix)
	if err != nil {
		panic(err)
	}
	transport := &http.Transport{MaxIdleConns: *workers, MaxIdleConnsPerHost: *workers}
	client := &http.Client{Transport: transport}
	deadline := time.Now().Add(*duration)
	results := make(chan result, *workers)
	var wg sync.WaitGroup
	for worker := 0; worker < *workers; worker++ {
		wg.Add(1)
		go func(seed int64) {
			defer wg.Done()
			r := result{statuses: make(map[int]int)}
			rng := rand.New(rand.NewSource(seed))
			for time.Now().Before(deadline) {
				path := paths[rng.Intn(len(paths))]
				started := time.Now()
				response, requestErr := client.Get(strings.TrimRight(*baseURL, "/") + path)
				latency := time.Since(started)
				if requestErr != nil {
					r.connectionErr++
					continue
				}
				_, _ = io.Copy(io.Discard, response.Body)
				_ = response.Body.Close()
				r.requests++
				r.statuses[response.StatusCode]++
				r.latencies = append(r.latencies, latency)
			}
			results <- r
		}(time.Now().UnixNano() + int64(worker))
	}
	wg.Wait()
	close(results)

	aggregate := result{statuses: make(map[int]int)}
	for r := range results {
		aggregate.requests += r.requests
		aggregate.connectionErr += r.connectionErr
		for status, count := range r.statuses {
			aggregate.statuses[status] += count
		}
		aggregate.latencies = append(aggregate.latencies, r.latencies...)
	}
	sort.Slice(aggregate.latencies, func(i, j int) bool { return aggregate.latencies[i] < aggregate.latencies[j] })
	fmt.Printf("requests=%d\n", aggregate.requests)
	fmt.Printf("rps=%.2f\n", float64(aggregate.requests)/duration.Seconds())
	fmt.Printf("status_200=%d\nstatus_400=%d\nstatus_503=%d\nstatus_500=%d\n", aggregate.statuses[200], aggregate.statuses[400], aggregate.statuses[503], aggregate.statuses[500])
	fmt.Printf("connection_errors=%d\n", aggregate.connectionErr)
	fmt.Printf("p50=%s\np95=%s\np99=%s\n", percentile(aggregate.latencies, 50), percentile(aggregate.latencies, 95), percentile(aggregate.latencies, 99))
}

func requestPaths(mix string) ([]string, error) {
	switch mix {
	case "burst":
		paths := make([]string, 0, 10)
		paths = append(paths, "/articles?limit=20", "/articles?limit=20", "/articles?limit=20", "/articles?limit=20", "/articles?limit=20", "/articles?limit=20")
		paths = append(paths, "/articles?author=Syndication&limit=20", "/articles?author=Syndication&limit=20")
		paths = append(paths, "/articles?query=kebijakan&limit=20", "/articles?author=Budi%20Santoso&limit=20")
		return paths, nil
	case "heavy":
		return []string{"/articles?author=Syndication&query=zzxqv&limit=20"}, nil
	default:
		return nil, fmt.Errorf("unknown -mix %q (use burst or heavy)", mix)
	}
}

func percentile(latencies []time.Duration, value int) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	index := (len(latencies)*value + 99) / 100
	return latencies[index-1]
}
