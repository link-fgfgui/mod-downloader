package filetransfer

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
)

type BackendKind string

const (
	BackendStdlib BackendKind = "stdlib"

	DefaultConcurrency = 4
	DefaultChunkSize   = 4 << 20
	DefaultMemoryLimit = 15 << 20
	defaultFileMode    = 0o644
)

type Request struct {
	URL         string
	Destination string
	Headers     map[string]string
	TempDir     string
	Concurrency int

	// ChunkSize is retained for API compatibility. Ranged downloads derive
	// balanced chunk sizes from the remote size and segment count.
	ChunkSize            int64
	MemoryLimit          int64
	AdaptiveConcurrency  bool
	TargetBytesPerSecond int64
	OverwriteExisting    bool
}

type Progress struct {
	BytesComplete int64
	TotalBytes    int64
}

type ProgressCallback func(Progress)

type ProgressTracker struct {
	complete atomic.Int64
	total    atomic.Int64
	mu       sync.RWMutex
	callback ProgressCallback
}

func NewProgressTracker(callback ProgressCallback) *ProgressTracker {
	return &ProgressTracker{callback: callback}
}

func (p *ProgressTracker) Snapshot() Progress {
	if p == nil {
		return Progress{}
	}
	return Progress{BytesComplete: p.complete.Load(), TotalBytes: p.total.Load()}
}

func (p *ProgressTracker) Reset(total int64) {
	if p == nil {
		return
	}
	p.total.Store(total)
	p.complete.Store(0)
	p.notify()
}

func (p *ProgressTracker) SetComplete(complete int64) {
	if p == nil {
		return
	}
	p.complete.Store(complete)
	p.notify()
}

func (p *ProgressTracker) AddComplete(delta int64) {
	if p == nil || delta == 0 {
		return
	}
	p.complete.Add(delta)
	p.notify()
}

func (p *ProgressTracker) notify() {
	p.mu.RLock()
	callback := p.callback
	p.mu.RUnlock()
	if callback != nil {
		callback(p.Snapshot())
	}
}

type Result struct {
	Path           string
	BytesComplete  int64
	TotalBytes     int64
	RangeSupported bool
}

type Backend interface {
	Kind() BackendKind
	Download(context.Context, Request, *ProgressTracker) (Result, error)
}

func New(kind BackendKind) (Backend, error) {
	switch BackendKind(strings.ToLower(strings.TrimSpace(string(kind)))) {
	case "", BackendStdlib:
		return NewStdlibBackend(nil), nil
	default:
		return nil, fmt.Errorf("unknown download backend: %s", kind)
	}
}

func normalizeRequest(req Request) (Request, error) {
	req.URL = strings.TrimSpace(req.URL)
	req.Destination = strings.TrimSpace(req.Destination)
	if req.URL == "" {
		return Request{}, fmt.Errorf("download url is required")
	}
	if req.Destination == "" {
		return Request{}, fmt.Errorf("download destination is required")
	}
	if req.Concurrency <= 0 {
		req.Concurrency = DefaultConcurrency
	}
	if req.ChunkSize <= 0 {
		req.ChunkSize = DefaultChunkSize
	}
	if req.MemoryLimit <= 0 {
		req.MemoryLimit = DefaultMemoryLimit
	}
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	return req, nil
}

func applyHeaders(req *http.Request, headers map[string]string) {
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if req.Header.Get("Accept-Encoding") == "" {
		req.Header.Set("Accept-Encoding", "identity")
	}
}
