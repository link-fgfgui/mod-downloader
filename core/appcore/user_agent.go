package appcore

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/link-fgfgui/mod-downloader-core/networking"
)

type requestRateLimiter struct {
	mu       sync.Mutex
	interval time.Duration
	next     time.Time
}

func newRequestRateLimiter(requestsPerSecond int) *requestRateLimiter {
	if requestsPerSecond <= 0 {
		return &requestRateLimiter{}
	}
	return &requestRateLimiter{interval: time.Second / time.Duration(requestsPerSecond)}
}

func (l *requestRateLimiter) wait(ctx context.Context) error {
	if l == nil || l.interval <= 0 {
		return nil
	}
	l.mu.Lock()
	now := time.Now()
	readyAt := now
	if l.next.After(now) {
		readyAt = l.next
	}
	l.next = readyAt.Add(l.interval)
	l.mu.Unlock()

	delay := time.Until(readyAt)
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type rateLimitedTransport struct {
	limiter *requestRateLimiter
	base    http.RoundTripper
}

func (t rateLimitedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.limiter.wait(req.Context()); err != nil {
		return nil, err
	}
	base := t.base
	if base == nil {
		base = networking.NewTransport()
	}
	return base.RoundTrip(req)
}

const defaultAppVersion = "dev"

const (
	mcimCurseForgeBaseURL = "https://mod.mcimirror.top/curseforge"
	mcimModrinthBaseURL   = "https://mod.mcimirror.top/modrinth/v2/"
)

func userAgentForVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		version = defaultAppVersion
	}
	return "mod-downloader/" + version
}

type curseForgeTransport struct {
	apiKey     string
	userAgent  string
	apiBaseURL string
	base       http.RoundTripper
}

func (t curseForgeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header = req.Header.Clone()
	if strings.TrimSpace(t.apiBaseURL) != "" {
		baseURL, err := url.Parse(t.apiBaseURL)
		if err != nil {
			return nil, err
		}
		clone.URL.Scheme = baseURL.Scheme
		clone.URL.Host = baseURL.Host
		clone.Host = baseURL.Host
		clone.URL.Path = strings.TrimRight(baseURL.Path, "/") + "/" + strings.TrimLeft(clone.URL.Path, "/")
		clone.URL.RawPath = ""
	}
	clone.Header.Set("Accept", "application/json")
	if strings.TrimSpace(t.apiKey) != "" {
		clone.Header.Set("x-api-key", t.apiKey)
	} else {
		clone.Header.Del("x-api-key")
	}
	clone.Header.Set("User-Agent", t.userAgent)
	base := t.base
	if base == nil {
		base = networking.NewTransport()
	}
	resp, err := base.RoundTrip(clone)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, errors.New(resp.Status)
	}
	return resp, nil
}
