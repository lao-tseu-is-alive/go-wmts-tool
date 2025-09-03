package tools

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/golog"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/imgTools"
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
func GetPngFromUrl(client *http.Client, url, path string, buffer, maxRetries int, l golog.MyLogger) error {
	var lastErr error
	l.Debug("GetPngFromUrl buffer: %d , url: %s", buffer, url)
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: wait 2^attempt seconds
			time.Sleep(time.Duration(1<<attempt) * time.Second)
		}

		// Create the directory structure
		if err := ensureDir(filepath.Dir(path)); err != nil {
			lastErr = fmt.Errorf("failed to create directory: %v", err)
			l.Error("💥💥 error creating dir %v", err)
			continue
		}

		// Make HTTP request
		resp, err := client.Get(url)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %v", err)
			l.Error("💥💥 error doing request  %v", err)
			continue
		}

		// Check status code
		if resp.StatusCode != http.StatusOK {
			err := resp.Body.Close()
			if err != nil {
				return fmt.Errorf("error occured doing Body.Close : %x", err)
			}
			lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			l.Error("💥💥 error unexpected status code %d doing request", resp.StatusCode)
			continue
		}

		if buffer == 0 {

			// Create output file
			file, err := os.Create(path)
			if err != nil {
				resp.Body.Close()
				if err != nil {
					return fmt.Errorf("error occured doing Body.Close : %x", err)
				}
				lastErr = fmt.Errorf("failed to create file: %v", err)
				l.Error("💥💥 error creating file %s, err: %v", path, err)
				continue
			}

			// Copy response body to file
			_, err = io.Copy(file, resp.Body)
			if err != nil {
				lastErr = fmt.Errorf("error occured doing io.Copy : %x", err)
				l.Error("💥💥 error doing io.Copy  %v", err)
				os.Remove(path) // Clean up partial file
				continue
			}
			err = file.Close()
			if err != nil {
				l.Error("💥💥 error doing file.Close  %v", err)
				return fmt.Errorf("error occured doing file.Close : %x", err)
			}
			resp.Body.Close()
			if err != nil {
				l.Error("💥💥 error doing Body.Close  %v", err)
				return fmt.Errorf("error occured doing Body.Close : %x", err)
			}
			return nil
		} else {
			l.Debug("buffer is not null(= %d) so we need to crop the image before saving it", buffer)
			// Decode the image from the response body.
			bufferedImage, _, err := image.Decode(resp.Body)
			if err != nil {
				l.Error("💥💥 error doing image.Decode(resp.Body)  %v", err)
				return fmt.Errorf("failed to decode meta-tile image: %w", err)
			}
			l.Debug("about to  imgTools.CropImage buffer:%d", buffer)
			img := imgTools.CropImage(bufferedImage, buffer, l)

			outFile, err := os.Create(path)
			if err != nil {
				return fmt.Errorf("failed to create tile image file: %w", err)
			}
			if err := png.Encode(outFile, img); err != nil {
				return fmt.Errorf("failed to encode tile image: %w", err)
			}
			outFile.Close()
			return nil
		}
	}

	return fmt.Errorf("# failed  after %d retries: %v", maxRetries, lastErr)
}
