package tools

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// CreateHTTPClient creates a configured HTTP client with timeouts
func CreateHTTPClient(maxTimeout, maxIdleConn, maxIdleConnPerHost, idleConnTimeout int) *http.Client {
	return &http.Client{
		Timeout: time.Duration(maxTimeout) * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        maxIdleConn,
			MaxIdleConnsPerHost: maxIdleConnPerHost,
			IdleConnTimeout:     time.Duration(idleConnTimeout) * time.Second,
		},
	}
}

// ensureDir creates the output directory if it doesn't exist
func ensureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

// GetPngFromUrl downloads a single tile with retry logic and saves it to a file in path parameter
func GetPngFromUrl(client *http.Client, url, path string, maxRetries int) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: wait 2^attempt seconds
			time.Sleep(time.Duration(1<<attempt) * time.Second)
		}

		// Create the directory structure
		if err := ensureDir(filepath.Dir(path)); err != nil {
			lastErr = fmt.Errorf("failed to create directory: %v", err)
			continue
		}

		// Make HTTP request
		resp, err := client.Get(url)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %v", err)
			continue
		}

		// Check status code
		if resp.StatusCode != http.StatusOK {
			err := resp.Body.Close()
			if err != nil {
				return fmt.Errorf("error occured doing Body.Close : %x", err)
			}
			lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			continue
		}

		// Create output file
		file, err := os.Create(path)
		if err != nil {
			resp.Body.Close()
			if err != nil {
				return fmt.Errorf("error occured doing Body.Close : %x", err)
			}
			lastErr = fmt.Errorf("failed to create file: %v", err)
			continue
		}

		// Copy response body to file
		_, err = io.Copy(file, resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("error occured doing io.Copy : %x", err)
			os.Remove(path) // Clean up partial file
			continue
		}
		err = file.Close()
		if err != nil {
			return fmt.Errorf("error occured doing file.Close : %x", err)
		}
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("error occured doing Body.Close : %x", err)
		}
		return nil
	}

	return fmt.Errorf("# failed  after %d retries: %v", maxRetries, lastErr)
}
