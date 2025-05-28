package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	urls, err := validateURLs(os.Args[1:])
	if err != nil {
		usageExample()
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	monitor := NewMonitor(urls)

	var wg sync.WaitGroup

	monitor.Start(ctx, &wg)

	<-ctx.Done()
	fmt.Println("\nShutting down gracefully...")

	wg.Wait()

	monitor.DisplayFinalTable()
}

func validateURLs(args []string) ([]string, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("at least one URL is required")
	}

	var validURLs []string

	for i, arg := range args {
		arg = strings.TrimSpace(arg)

		if arg == "" {
			return nil, fmt.Errorf("argument %d is empty", i+1)
		}

		parsedURL, err := url.Parse(arg)
		if err != nil {
			return nil, fmt.Errorf("invalid URL '%s': %v", arg, err)
		}

		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			return nil, fmt.Errorf("URL '%s' must have http or https scheme", arg)
		}

		if parsedURL.Host == "" {
			return nil, fmt.Errorf("URL '%s' must have a valid host", arg)
		}

		validURLs = append(validURLs, arg)
	}

	return validURLs, nil
}

func usageExample() {
	fmt.Fprintf(os.Stderr, "Usage: go run main.go <url1> [url2] ...\n")
	fmt.Fprintf(os.Stderr, "   or: ./web-monitor <url1> [url2] ...\n")
	fmt.Fprintf(os.Stderr, "\nExample: go run main.go https://example.com https://seznam.cz\n")
	fmt.Fprintf(os.Stderr, "Example: go run main.go https://google.com https://github.com\n")
}
