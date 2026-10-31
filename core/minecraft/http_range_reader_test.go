package minecraft

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPRangeReaderResolvesRedirectBeforeRangeProbe(t *testing.T) {
	data := []byte("0123456789")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/edge.jar":
			if r.Header.Get("Range") != "" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			http.Redirect(w, r, "/media.jar", http.StatusFound)
		case "/media.jar":
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
			if r.Method == http.MethodHead {
				return
			}
			if r.Header.Get("Range") != "bytes=0-0" {
				t.Fatalf("unexpected range header: %q", r.Header.Get("Range"))
			}
			w.Header().Set("Content-Range", fmt.Sprintf("bytes 0-0/%d", len(data)))
			w.Header().Set("Content-Length", "1")
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write(data[:1])
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	reader, err := NewHTTPRangeReaderAt(server.URL + "/edge.jar")
	if err != nil {
		t.Fatalf("NewHTTPRangeReaderAt() error = %v", err)
	}
	if reader.url != server.URL+"/media.jar" {
		t.Fatalf("reader url = %q, want redirected media URL", reader.url)
	}
	if reader.Size() != int64(len(data)) {
		t.Fatalf("reader size = %d, want %d", reader.Size(), len(data))
	}
}

func TestHTTPRangeReaderConstructorHonorsTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Range", "bytes 0-0/10")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte("0"))
	}))
	defer server.Close()

	_, err := NewHTTPRangeReaderAtWithTimeout(server.URL+"/slow.jar", 30*time.Millisecond)
	if err == nil {
		t.Fatal("NewHTTPRangeReaderAtWithTimeout() error = nil, want timeout")
	}
}

func TestHTTPRangeReaderReadAtHonorsTimeout(t *testing.T) {
	data := []byte("0123456789")
	probed := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		if r.Method == http.MethodHead {
			return
		}
		if !probed {
			probed = true
			w.Header().Set("Content-Range", fmt.Sprintf("bytes 0-0/%d", len(data)))
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write(data[:1])
			return
		}
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(data)
	}))
	defer server.Close()

	reader, err := NewHTTPRangeReaderAtWithTimeout(server.URL+"/slow-read.jar", 80*time.Millisecond)
	if err != nil {
		t.Fatalf("NewHTTPRangeReaderAtWithTimeout() error = %v", err)
	}
	time.Sleep(90 * time.Millisecond)
	_, err = reader.ReadAt(make([]byte, 1), 0)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ReadAt() error = %v, want deadline exceeded", err)
	}
}

func TestHTTPRangeReaderHandlesOKResponseForReadAt(t *testing.T) {
	data := []byte("0123456789")
	probed := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		if r.Method == http.MethodHead {
			return
		}
		if !probed {
			probed = true
			w.Header().Set("Content-Range", fmt.Sprintf("bytes 0-0/%d", len(data)))
			w.Header().Set("Content-Length", "1")
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write(data[:1])
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}))
	defer server.Close()

	reader, err := NewHTTPRangeReaderAt(server.URL + "/file.jar")
	if err != nil {
		t.Fatalf("NewHTTPRangeReaderAt() error = %v", err)
	}

	buf := make([]byte, 4)
	n, err := reader.ReadAt(buf, 3)
	if err != nil && err != io.EOF {
		t.Fatalf("ReadAt() error = %v", err)
	}
	if n != len(buf) {
		t.Fatalf("ReadAt() n = %d, want %d", n, len(buf))
	}
	if string(buf) != "3456" {
		t.Fatalf("ReadAt() data = %q, want %q", string(buf), "3456")
	}
}
