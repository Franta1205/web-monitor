package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
)

func TestURLValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		urls          []string
		shouldError   bool
		errorContains string
	}{
		{
			name:        "valid HTTP URLs",
			urls:        []string{"http://example.com", "https://google.com"},
			shouldError: false,
		},
		{
			name:        "valid single HTTPS URL",
			urls:        []string{"https://seznam.cz"},
			shouldError: false,
		},
		{
			name:          "empty input",
			urls:          []string{},
			shouldError:   true,
			errorContains: "at least one URL is required",
		},
		{
			name:          "invalid scheme - FTP",
			urls:          []string{"ftp://example.com"},
			shouldError:   true,
			errorContains: "must have http or https scheme",
		},
		{
			name:          "invalid scheme - file",
			urls:          []string{"file:///etc/passwd"},
			shouldError:   true,
			errorContains: "must have http or https scheme",
		},
		{
			name:        "malformed URL",
			urls:        []string{"not-a-url-at-all"},
			shouldError: true,
		},
		{
			name:          "empty URL string",
			urls:          []string{""},
			shouldError:   true,
			errorContains: "is empty",
		},
		{
			name:          "URL without host",
			urls:          []string{"http://"},
			shouldError:   true,
			errorContains: "must have a valid host",
		},
		{
			name:        "mixed valid and invalid URLs",
			urls:        []string{"https://example.com", "invalid-url"},
			shouldError: true,
		},
		{
			name:        "URL with whitespace",
			urls:        []string{"  https://example.com  "},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := validateURLs(tt.urls)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if tt.errorContains != "" && err != nil {
					if !strings.Contains(err.Error(), tt.errorContains) {
						t.Errorf("Expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestStatsCalculations(t *testing.T) {
	t.Parallel()

	urlStats := NewURLStats("http://example.com")

	testData := []struct {
		duration time.Duration
		size     int64
		success  bool
	}{
		{100 * time.Millisecond, 1000, true},
		{200 * time.Millisecond, 2000, true},
		{300 * time.Millisecond, 1500, false},
		{150 * time.Millisecond, 500, true},
	}

	for _, data := range testData {
		urlStats.Update(data.duration, data.size, data.success)
	}

	snapshot := urlStats.GetSnapshot()

	if snapshot.TotalRequests != 4 {
		t.Errorf("Expected 4 total requests, got %d", snapshot.TotalRequests)
	}

	if snapshot.SuccessCount != 3 {
		t.Errorf("Expected 3 successful requests, got %d", snapshot.SuccessCount)
	}

	if snapshot.MinDuration != 100*time.Millisecond {
		t.Errorf("Expected min duration 100ms, got %v", snapshot.MinDuration)
	}

	if snapshot.MaxDuration != 300*time.Millisecond {
		t.Errorf("Expected max duration 300ms, got %v", snapshot.MaxDuration)
	}

	expectedAvg := 187500 * time.Microsecond
	actualAvg := snapshot.AverageDuration()
	if actualAvg != expectedAvg {
		t.Errorf("Expected average duration %v, got %v", expectedAvg, actualAvg)
	}

	if snapshot.MinSize != 500 {
		t.Errorf("Expected min size 500, got %d", snapshot.MinSize)
	}

	if snapshot.MaxSize != 2000 {
		t.Errorf("Expected max size 2000, got %d", snapshot.MaxSize)
	}

	expectedAvgSize := int64(1250)
	actualAvgSize := snapshot.AverageSize()
	if actualAvgSize != expectedAvgSize {
		t.Errorf("Expected average size %d, got %d", expectedAvgSize, actualAvgSize)
	}
}

func TestThreadSafety(t *testing.T) {
	t.Parallel()

	urlStats := NewURLStats("http://example.com")

	const numGoroutines = 10
	const updatesPerGoroutine = 100

	done := make(chan bool, numGoroutines)

	// updating stats concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			for j := 0; j < updatesPerGoroutine; j++ {
				duration := time.Millisecond * time.Duration(j+routineID)
				size := int64(j + routineID*100)
				success := (j+routineID)%2 == 0

				urlStats.Update(duration, size, success)
			}
			done <- true
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	snapshot := urlStats.GetSnapshot()

	expectedTotal := int64(numGoroutines * updatesPerGoroutine)
	if snapshot.TotalRequests != expectedTotal {
		t.Errorf("Expected %d total requests, got %d", expectedTotal, snapshot.TotalRequests)
	}

	if snapshot.SuccessCount > snapshot.TotalRequests {
		t.Errorf("Success count (%d) cannot exceed total requests (%d)",
			snapshot.SuccessCount, snapshot.TotalRequests)
	}

	if snapshot.MinDuration > snapshot.MaxDuration && snapshot.TotalRequests > 0 {
		t.Errorf("Min duration (%v) cannot be greater than max duration (%v)",
			snapshot.MinDuration, snapshot.MaxDuration)
	}

	if snapshot.MinSize > snapshot.MaxSize && snapshot.TotalRequests > 0 {
		t.Errorf("Min size (%d) cannot be greater than max size (%d)",
			snapshot.MinSize, snapshot.MaxSize)
	}
}

func TestHTTPSuccessDetection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		statusCode      int
		expectedSuccess bool
	}{
		{"200 OK", 200, true},
		{"201 Created", 201, true},
		{"299 Custom 2xx", 299, true},
		{"300 Multiple Choices", 300, true},
		{"301 Moved Permanently", 301, true},
		{"302 Found", 302, true},
		{"399 Custom 3xx", 399, true},
		{"400 Bad Request", 400, false},
		{"401 Unauthorized", 401, false},
		{"404 Not Found", 404, false},
		{"500 Internal Server Error", 500, false},
		{"503 Service Unavailable", 503, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			url := fmt.Sprintf("http://test.example.com/status%d", tt.statusCode)

			httpmock.RegisterResponder("GET", url,
				httpmock.NewStringResponder(tt.statusCode, "Test response body"))

			monitor := NewMonitor([]string{url})
			monitor.makeRequest(url)

			stats := monitor.stats[url].GetSnapshot()

			if stats.TotalRequests != 1 {
				t.Errorf("Expected 1 request, got %d", stats.TotalRequests)
			}

			expectedSuccessCount := int64(0)
			if tt.expectedSuccess {
				expectedSuccessCount = 1
			}

			if stats.SuccessCount != expectedSuccessCount {
				t.Errorf("Status %d: Expected success count %d, got %d",
					tt.statusCode, expectedSuccessCount, stats.SuccessCount)
			}
		})
	}
}

func TestMakeRequestBasic(t *testing.T) {
	t.Parallel()

	monitor := NewMonitor([]string{"http://test.example.com"})

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("makeRequest panicked: %v", r)
		}
	}()

	monitor.makeRequest("http://test.example.com")

	stats := monitor.stats["http://test.example.com"].GetSnapshot()
	if stats.TotalRequests != 1 {
		t.Errorf("Expected 1 request recorded, got %d", stats.TotalRequests)
	}
}

func TestMonitorCreation(t *testing.T) {
	t.Parallel()

	urls := []string{
		"http://example.com",
		"https://google.com",
		"https://github.com",
	}

	monitor := NewMonitor(urls)

	if len(monitor.urls) != len(urls) {
		t.Errorf("Expected %d URLs, got %d", len(urls), len(monitor.urls))
	}

	if len(monitor.stats) != len(urls) {
		t.Errorf("Expected %d stats entries, got %d", len(urls), len(monitor.stats))
	}

	if monitor.httpClient.Timeout != 10*time.Second {
		t.Errorf("Expected 10s timeout, got %v", monitor.httpClient.Timeout)
	}

	for _, url := range urls {
		stats := monitor.stats[url]
		if stats == nil {
			t.Errorf("Stats not initialized for URL: %s", url)
			continue
		}

		if stats.URL != url {
			t.Errorf("Expected stats URL %s, got %s", url, stats.URL)
		}

		if stats.TotalRequests != 0 {
			t.Errorf("Expected 0 initial requests for %s, got %d", url, stats.TotalRequests)
		}
	}
}

func TestStatsSnapshot(t *testing.T) {
	t.Parallel()

	stats := NewURLStats("http://example.com")

	stats.Update(100*time.Millisecond, 1000, true)
	stats.Update(200*time.Millisecond, 2000, false)

	snapshot := stats.GetSnapshot()

	if snapshot.TotalRequests != 2 {
		t.Errorf("Expected 2 requests in snapshot, got %d", snapshot.TotalRequests)
	}

	if snapshot.SuccessCount != 1 {
		t.Errorf("Expected 1 success in snapshot, got %d", snapshot.SuccessCount)
	}

	stats.Update(300*time.Millisecond, 3000, true)

	if snapshot.TotalRequests != 2 {
		t.Errorf("Snapshot should be independent, got %d requests", snapshot.TotalRequests)
	}
}

func TestFormatFunctions(t *testing.T) {
	t.Parallel()

	testDurations := []struct {
		input    time.Duration
		expected string
	}{
		{0, "-"},
		{50 * time.Millisecond, "50ms"},
		{1500 * time.Millisecond, "1.50s"},
		{2 * time.Second, "2.00s"},
	}

	for _, test := range testDurations {
		result := formatDuration(test.input)
		if result != test.expected {
			t.Errorf("formatDuration(%v) = %s, expected %s", test.input, result, test.expected)
		}
	}

	testSizes := []struct {
		input    int64
		expected string
	}{
		{0, "-"},
		{500, "500B"},
		{1536, "1.5KB"},
		{2097152, "2.0MB"},
	}

	for _, test := range testSizes {
		result := formatSize(test.input)
		if result != test.expected {
			t.Errorf("formatSize(%d) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestAverageCalculations(t *testing.T) {
	t.Parallel()

	stats := NewURLStats("http://example.com")

	if stats.AverageDuration() != 0 {
		t.Errorf("Expected 0 average duration with no requests, got %v", stats.AverageDuration())
	}

	if stats.AverageSize() != 0 {
		t.Errorf("Expected 0 average size with no requests, got %d", stats.AverageSize())
	}

	stats.Update(100*time.Millisecond, 1000, true)

	if stats.AverageDuration() != 100*time.Millisecond {
		t.Errorf("Expected 100ms average duration, got %v", stats.AverageDuration())
	}

	if stats.AverageSize() != 1000 {
		t.Errorf("Expected 1000 average size, got %d", stats.AverageSize())
	}

	stats.Update(200*time.Millisecond, 2000, false)

	expectedAvgDuration := 150 * time.Millisecond
	if stats.AverageDuration() != expectedAvgDuration {
		t.Errorf("Expected %v average duration, got %v", expectedAvgDuration, stats.AverageDuration())
	}

	expectedAvgSize := int64(1500)
	if stats.AverageSize() != expectedAvgSize {
		t.Errorf("Expected %d average size, got %d", expectedAvgSize, stats.AverageSize())
	}
}

func TestURLStatsUpdate(t *testing.T) {
	t.Parallel()

	stats := NewURLStats("http://example.com")

	stats.Update(100*time.Millisecond, 1000, true)

	snapshot := stats.GetSnapshot()
	if snapshot.MinDuration != 100*time.Millisecond {
		t.Errorf("Expected min duration 100ms after first update, got %v", snapshot.MinDuration)
	}

	if snapshot.MinSize != 1000 {
		t.Errorf("Expected min size 1000 after first update, got %d", snapshot.MinSize)
	}

	stats.Update(50*time.Millisecond, 500, false)

	snapshot = stats.GetSnapshot()
	if snapshot.MinDuration != 50*time.Millisecond {
		t.Errorf("Expected min duration 50ms after smaller update, got %v", snapshot.MinDuration)
	}

	if snapshot.MinSize != 500 {
		t.Errorf("Expected min size 500 after smaller update, got %d", snapshot.MinSize)
	}

	stats.Update(300*time.Millisecond, 3000, true)

	snapshot = stats.GetSnapshot()
	if snapshot.MaxDuration != 300*time.Millisecond {
		t.Errorf("Expected max duration 300ms, got %v", snapshot.MaxDuration)
	}

	if snapshot.MaxSize != 3000 {
		t.Errorf("Expected max size 3000, got %d", snapshot.MaxSize)
	}

	if snapshot.TotalRequests != 3 {
		t.Errorf("Expected 3 total requests, got %d", snapshot.TotalRequests)
	}

	if snapshot.SuccessCount != 2 {
		t.Errorf("Expected 2 successful requests, got %d", snapshot.SuccessCount)
	}
}
