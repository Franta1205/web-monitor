package main

import (
	"sync"
	"time"
)

type URLStats struct {
	URL           string
	TotalRequests int64
	SuccessCount  int64

	MinDuration   time.Duration
	MaxDuration   time.Duration
	TotalDuration time.Duration

	MinSize   int64
	MaxSize   int64
	TotalSize int64

	mu sync.RWMutex
}

func NewURLStats(url string) *URLStats {
	return &URLStats{
		URL: url,
		MinDuration: time.Duration(^uint64(0) >> 1),
		MinSize:     ^int64(0) >> 1,
	}
}

func (s *URLStats) Update(duration time.Duration, bodySize int64, success bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TotalRequests++
	if success {
		s.SuccessCount++
	}

	if s.TotalRequests == 1 || duration < s.MinDuration {
		s.MinDuration = duration
	}
	if duration > s.MaxDuration {
		s.MaxDuration = duration
	}
	s.TotalDuration += duration

	if s.TotalRequests == 1 || bodySize < s.MinSize {
		s.MinSize = bodySize
	}
	if bodySize > s.MaxSize {
		s.MaxSize = bodySize
	}
	s.TotalSize += bodySize
}

func (s *URLStats) GetSnapshot() URLStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return URLStats{
		URL:           s.URL,
		TotalRequests: s.TotalRequests,
		SuccessCount:  s.SuccessCount,
		MinDuration:   s.MinDuration,
		MaxDuration:   s.MaxDuration,
		TotalDuration: s.TotalDuration,
		MinSize:       s.MinSize,
		MaxSize:       s.MaxSize,
		TotalSize:     s.TotalSize,
	}
}

func (s *URLStats) AverageDuration() time.Duration {
	if s.TotalRequests == 0 {
		return 0
	}
	return s.TotalDuration / time.Duration(s.TotalRequests)
}

func (s *URLStats) AverageSize() int64 {
	if s.TotalRequests == 0 {
		return 0
	}
	return s.TotalSize / s.TotalRequests
}
