package main

import (
	"fmt"
	"time"
)

func (m *Monitor) displayTable() {
	m.clearScreen()
	m.renderTable()
}

func (m *Monitor) displayFinalTable() {
	fmt.Println("\nFinal Statistics:")
	m.renderTable()
}

func (m *Monitor) renderTable() {

	// Table header
	fmt.Printf("%-30s %-12s %-12s %-12s %-10s %-10s %-10s %-15s\n",
		"URL", "Duration Min", "Duration Avg", "Duration Max",
		"Size Min", "Size Avg", "Size Max", "OK")

	// Header separator
	fmt.Printf("%-30s %-12s %-12s %-12s %-10s %-10s %-10s %-15s\n",
		"────────────────────────────", "────────────", "────────────", "────────────",
		"─────────", "─────────", "─────────", "──────────────")

	// Data rows
	m.statsMu.RLock()
	for _, url := range m.urls {
		stat := m.stats[url]
		snapshot := stat.GetSnapshot()

		displayURL := url
		if len(displayURL) > 28 {
			displayURL = displayURL[:25] + "..."
		}

		// Format durations
		minDur := formatDuration(snapshot.MinDuration)
		avgDur := formatDuration(snapshot.AverageDuration())
		maxDur := formatDuration(snapshot.MaxDuration)

		// Format sizes
		minSize := formatSize(snapshot.MinSize)
		avgSize := formatSize(snapshot.AverageSize())
		maxSize := formatSize(snapshot.MaxSize)

		// Format success ratio
		okRatio := fmt.Sprintf("%d/%d", snapshot.SuccessCount, snapshot.TotalRequests)

		fmt.Printf("%-30s %-12s %-12s %-12s %-10s %-10s %-10s %-15s\n",
			displayURL, minDur, avgDur, maxDur, minSize, avgSize, maxSize, okRatio)
	}
	m.statsMu.RUnlock()
}

func (m *Monitor) clearScreen() {
	fmt.Print("\033[2J\033[H")
}

func formatDuration(d time.Duration) string {
	if d == 0 || d == time.Duration(^uint64(0)>>1) {
		return "-"
	}

	if d >= time.Second {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
	return fmt.Sprintf("%dms", d.Milliseconds())
}

func formatSize(size int64) string {
	if size == 0 || size == ^int64(0)>>1 {
		return "-"
	}

	if size >= 1024*1024 {
		return fmt.Sprintf("%.1fMB", float64(size)/(1024*1024))
	}
	if size >= 1024 {
		return fmt.Sprintf("%.1fKB", float64(size)/1024)
	}
	return fmt.Sprintf("%dB", size)
}
