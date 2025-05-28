package main

import (
	"io"
	"net/http"
	"sync"
	"time"
)

type Monitor struct {
	urls        []string
	stats       map[string]*URLStats
	httpClient  *http.Client
	shutdown    chan struct{}
	wg          sync.WaitGroup
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
		shutdown:    make(chan struct{}),
		updatedData: make(chan struct{}),
	}
}

func (m *Monitor) Start() {

	for _, url := range m.urls {
		m.wg.Add(1)
		go m.monitorURL(url)
	}

	m.wg.Add(1)
	go m.displayLoop()
}

func (m *Monitor) monitorURL(url string) {
	defer m.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	m.makeRequest(url)

	for {
		select {
		case <-ticker.C:
			m.makeRequest(url)
		case <-m.shutdown:
			return
		}
	}
}

func (m *Monitor) makeRequest(url string) {
	start := time.Now()

	resp, err := m.httpClient.Get(url)
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

func (m *Monitor) displayLoop() {
	defer m.wg.Done()

	m.displayTable()

	for {
		select {
		case <-m.updatedData:
			m.displayTable()
		case <-m.shutdown:
			return
		}
	}
}

func (m *Monitor) Shutdown() {
	close(m.shutdown)

	m.wg.Wait()

	m.displayFinalTable()
}
