package tools

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/golog"
)

func newTestLogger(t *testing.T) golog.MyLogger {
	t.Helper()
	l, err := golog.NewLogger("simple", os.Stderr, golog.ErrorLevel, "test:")
	if err != nil {
		t.Fatalf("cannot create logger: %v", err)
	}
	return l
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatalf("cannot encode png: %v", err)
	}
	return buf.Bytes()
}

// newFlakyServer answers each request with the next status in statuses (the last one is repeated),
// sending a png on 200 and counting the requests.
func newFlakyServer(t *testing.T, contentType string, statuses ...int) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	img := pngBytes(t)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := int(calls.Add(1)) - 1
		status := statuses[min(n, len(statuses)-1)]
		if status != http.StatusOK {
			http.Error(w, "upstream error", status)
			return
		}
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write(img)
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

func TestFetchImageWithRetry(t *testing.T) {
	retryBaseDelay = time.Millisecond
	tests := []struct {
		name        string
		contentType string
		statuses    []int
		maxRetries  int
		wantErr     bool
		wantCalls   int32
	}{
		{"ok first time", "image/png", []int{200}, 3, false, 1},
		{"502 then ok", "image/png", []int{502, 502, 200}, 3, false, 3},
		{"always 502 gives up after retries", "image/png", []int{502}, 3, true, 4},
		{"no retry when maxRetries is 0", "image/png", []int{502, 200}, 0, true, 1},
		{"400 is not retried", "image/png", []int{400, 200}, 3, true, 1},
		{"200 with xml exception is not retried", "text/xml", []int{200}, 3, true, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, calls := newFlakyServer(t, tt.contentType, tt.statuses...)
			body, err := FetchImageWithRetry(srv.Client(), srv.URL, tt.maxRetries, newTestLogger(t))
			if (err != nil) != tt.wantErr {
				t.Fatalf("got err %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(body) == 0 {
				t.Errorf("got empty body")
			}
			if got := calls.Load(); got != tt.wantCalls {
				t.Errorf("got %d requests, want %d", got, tt.wantCalls)
			}
		})
	}
}

func TestGetPngFromUrlRetriesAndSaves(t *testing.T) {
	retryBaseDelay = time.Millisecond
	for _, buffer := range []int{0, 1} {
		srv, _ := newFlakyServer(t, "image/png", 502, 200)
		path := filepath.Join(t.TempDir(), "7", "3009", "1893.png")
		if err := GetPngFromUrl(srv.Client(), srv.URL, path, buffer, 2, newTestLogger(t)); err != nil {
			t.Fatalf("buffer %d: unexpected error: %v", buffer, err)
		}
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("buffer %d: tile not saved: %v", buffer, err)
		}
		if _, err := png.Decode(f); err != nil {
			t.Errorf("buffer %d: saved tile is not a valid png: %v", buffer, err)
		}
		f.Close()
	}
}

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "tile.png")
	if err := WriteFileAtomic(path, func(w io.Writer) error {
		_, err := w.Write([]byte("hello"))
		return err
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if info.Mode().Perm() != tileFileMode {
		t.Errorf("got mode %v, want %v", info.Mode().Perm(), os.FileMode(tileFileMode))
	}

	// a failing writer must neither replace the existing file nor leave a temporary file
	if err := WriteFileAtomic(path, func(w io.Writer) error {
		_, _ = w.Write([]byte("partial"))
		return os.ErrClosed
	}); err == nil {
		t.Fatalf("expected an error")
	}
	if got, _ := os.ReadFile(path); string(got) != "hello" {
		t.Errorf("existing file was modified: %q", got)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Errorf("temporary file left behind: %v", entries)
	}
}
