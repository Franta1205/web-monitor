package main

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"
)

type Monitor struct {
	urls        []string
	stats       map[string]*URLStats
	httpClient  *http.Client
	statsMu     sync.RWMutex
	updatedData chan struct{}
}

func NewMonitor(urls []string) *Monitor {
	stats := make(map[string]*URLStats)

	for _, url := range urls {
		stats[url] = NewURLStats(url)
	}

	return &Monitor{
		urls:  urls,
		stats: stats,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		updatedData: make(chan struct{}, 100),
	}
}

func (m *Monitor) Start(ctx context.Context, wg *sync.WaitGroup) {
	for _, url := range m.urls {
		wg.Add(1)
		go m.monitorURL(ctx, wg, url)
	}

	wg.Add(1)
	go m.displayLoop(ctx, wg)
}

func (m *Monitor) monitorURL(ctx context.Context, wg *sync.WaitGroup, url string) {
	defer wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	m.makeRequest(ctx, url)

	for {
		select {
		case <-ticker.C:
			m.makeRequest(ctx, url)
		case <-ctx.Done():
			return
		}
	}
}

func (m *Monitor) makeRequest(ctx context.Context, url string) {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		m.updateStats(url, time.Since(start), 0, false)
		return
	}

	resp, err := m.httpClient.Do(req)
	duration := time.Since(start)

	var bodySize int64
	var success bool

	if err != nil {
		success = false
		bodySize = 0
	} else {
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			success = false
			bodySize = 0
		} else {
			bodySize = int64(len(body))
			success = resp.StatusCode >= 200 && resp.StatusCode < 400
		}
	}

	m.updateStats(url, duration, bodySize, success)
}

func (m *Monitor) updateStats(url string, duration time.Duration, bodySize int64, success bool) {
	m.statsMu.RLock()
	stat := m.stats[url]
	m.statsMu.RUnlock()

	stat.Update(duration, bodySize, success)

	select {
	case m.updatedData <- struct{}{}:
	default:
	}
}

func (m *Monitor) displayLoop(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	m.displayTable()

	for {
		select {
		case <-m.updatedData:
			m.displayTable()
		case <-ctx.Done():
			return
		}
	}
}

func (m *Monitor) DisplayFinalTable() {
	m.displayFinalTable()
}