package networking

import (
	"net/http"
	"strings"
	"time"
)

var proxyFromEnvironment = http.ProxyFromEnvironment

// NewTransport returns an independent transport that follows the process proxy environment.
func NewTransport() *http.Transport {
	if base, ok := http.DefaultTransport.(*http.Transport); ok {
		transport := base.Clone()
		transport.Proxy = proxyFromEnvironment
		return transport
	}
	return &http.Transport{Proxy: proxyFromEnvironment}
}

func NewClient(timeout time.Duration) *http.Client {
	return NewClientWithUserAgent(timeout, "")
}

// NewClientWithUserAgent returns an HTTP client that supplies the application
// identity when the caller has not explicitly provided a User-Agent header.
func NewClientWithUserAgent(timeout time.Duration, userAgent string) *http.Client {
	userAgent = strings.TrimSpace(userAgent)
	if userAgent == "" {
		userAgent = "mod-downloader/dev"
	}
	return &http.Client{Transport: userAgentTransport{base: NewTransport(), userAgent: userAgent}, Timeout: timeout}
}

type userAgentTransport struct {
	base      http.RoundTripper
	userAgent string
}

func (t userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header = req.Header.Clone()
	if clone.Header.Get("User-Agent") == "" {
		clone.Header.Set("User-Agent", t.userAgent)
	}
	return t.base.RoundTrip(clone)
}
