package tools

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/golog"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/imgTools"
)

const (
	// DefaultMaxRetries is the default number of retries after a first failed WMS request.
	DefaultMaxRetries = 3
	// tileFileMode is the permission of saved tiles, they must stay readable by the web server.
	tileFileMode = 0644
)

// retryBaseDelay is the wait before the first retry, doubled at each following attempt.
// It is a variable only to allow tests to run fast.
var retryBaseDelay = 1 * time.Second

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

// permanentError marks a failure that will not go away by retrying the same request.
type permanentError struct{ err error }

func (e *permanentError) Error() string { return e.err.Error() }
func (e *permanentError) Unwrap() error { return e.err }

// isRetryableStatus reports whether an HTTP status code is likely to be transient.
// 502 typically happens when the proxy (nginx) loses its connection to the WMS upstream.
func isRetryableStatus(code int) bool {
	switch code {
	case http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}

// backoffDelay returns the wait before retry number attempt (1-based): 1s, 2s, 4s, ...
// plus up to 50% random jitter so that concurrent workers do not hit the server in sync.
func backoffDelay(attempt int) time.Duration {
	d := retryBaseDelay << (attempt - 1)
	return d + rand.N(d/2)
}

// fetchImageOnce does a single GET on url and returns the full response body.
// The body is always drained and closed, so the connection can be safely reused.
func fetchImageOnce(client *http.Client, url string) ([]byte, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body (status %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("WMS request returned non-OK status: %d", resp.StatusCode)
		if !isRetryableStatus(resp.StatusCode) {
			return nil, &permanentError{err}
		}
		return nil, err
	}
	// a WMS server may answer 200 with an XML ServiceException instead of an image
	if contentType := resp.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "image/") {
		return nil, &permanentError{fmt.Errorf("WMS response is not an image (Content-Type: %q): %.300s", contentType, body)}
	}
	return body, nil
}

// FetchImageWithRetry downloads the image at url, retrying up to maxRetries times
// with exponential backoff on network errors and transient HTTP status codes.
// It returns the raw image bytes.
func FetchImageWithRetry(client *http.Client, url string, maxRetries int, l golog.MyLogger) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := backoffDelay(attempt)
			l.Warn("⚠️ attempt %d/%d failed: %v, retrying in %s", attempt, maxRetries+1, lastErr, delay.Round(time.Millisecond))
			time.Sleep(delay)
		}
		body, err := fetchImageOnce(client, url)
		if err == nil {
			if attempt > 0 {
				l.Warn("✅ request succeeded after %d retries", attempt)
			}
			return body, nil
		}
		lastErr = err
		var permErr *permanentError
		if errors.As(err, &permErr) {
			return nil, fmt.Errorf("%w for url : [%s]", err, url)
		}
	}
	return nil, fmt.Errorf("failed after %d retries: %w for url : [%s]", maxRetries, lastErr, url)
}

// WriteFileAtomic creates path (and its parent directories) with the content produced by write.
// The content is first written to a temporary file in the same directory, then renamed,
// so a crash or kill never leaves a truncated file at path.
func WriteFileAtomic(path string, write func(w io.Writer) error) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file in %s: %w", dir, err)
	}
	tmpName := tmp.Name()
	// no-op once the rename succeeded
	defer os.Remove(tmpName)

	if err := write(tmp); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close %s: %w", tmpName, err)
	}
	if err := os.Chmod(tmpName, tileFileMode); err != nil {
		return fmt.Errorf("failed to chmod %s: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("failed to rename %s to %s: %w", tmpName, path, err)
	}
	return nil
}

// encoderBufferPool lets png encodes reuse their buffers (about 1 MB of zlib state each),
// it is safe for concurrent use by the workers.
type encoderBufferPool struct{ pool sync.Pool }

func (p *encoderBufferPool) Get() *png.EncoderBuffer {
	b, _ := p.pool.Get().(*png.EncoderBuffer)
	return b
}

func (p *encoderBufferPool) Put(b *png.EncoderBuffer) { p.pool.Put(b) }

// pngEncoder is shared by all tile writes, allocating a new zlib writer for each tile dominated the CPU usage.
var pngEncoder = &png.Encoder{BufferPool: &encoderBufferPool{}}

// SavePng encodes img as png and saves it atomically at path.
func SavePng(path string, img image.Image) error {
	return WriteFileAtomic(path, func(w io.Writer) error {
		return pngEncoder.Encode(w, img)
	})
}

// GetPngFromUrl downloads a single tile with retry logic and saves it to a file in path parameter
func GetPngFromUrl(client *http.Client, url, path string, buffer, maxRetries int, l golog.MyLogger) error {
	l.Debug("GetPngFromUrl buffer: %d , url: %s", buffer, url)
	body, err := FetchImageWithRetry(client, url, maxRetries, l)
	if err != nil {
		return err
	}

	if buffer == 0 {
		return WriteFileAtomic(path, func(w io.Writer) error {
			_, err := w.Write(body)
			return err
		})
	}

	l.Debug("buffer is not null(= %d) so we need to crop the image before saving it", buffer)
	bufferedImage, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to decode tile image: %w", err)
	}
	return SavePng(path, imgTools.CropImage(bufferedImage, buffer, l))
}
