package networking

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNewTransportUsesEnvironmentProxyResolver(t *testing.T) {
	previous := proxyFromEnvironment
	t.Cleanup(func() { proxyFromEnvironment = previous })
	want, err := url.Parse("http://127.0.0.1:18080")
	if err != nil {
		t.Fatal(err)
	}
	called := false
	proxyFromEnvironment = func(req *http.Request) (*url.URL, error) {
		called = true
		if req.URL.Host != "example.test" {
			t.Fatalf("request host = %q", req.URL.Host)
		}
		return want, nil
	}

	transport := NewTransport()
	req, err := http.NewRequest(http.MethodGet, "https://example.test/file", nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := transport.Proxy(req)
	if err != nil {
		t.Fatal(err)
	}
	if !called || got.String() != want.String() {
		t.Fatalf("proxy = %v, called = %v", got, called)
	}
}

func TestNewClientRoutesRequestsThroughResolvedProxy(t *testing.T) {
	previous := proxyFromEnvironment
	t.Cleanup(func() { proxyFromEnvironment = previous })

	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Host != "origin.invalid" || req.URL.Path != "/resource" {
			t.Fatalf("proxy request URL = %s", req.URL.String())
		}
		_, _ = w.Write([]byte("proxied"))
	}))
	defer proxy.Close()
	proxyURL, err := url.Parse(proxy.URL)
	if err != nil {
		t.Fatal(err)
	}
	proxyFromEnvironment = func(*http.Request) (*url.URL, error) { return proxyURL, nil }

	resp, err := NewClient(0).Get("http://origin.invalid/resource")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(body)) != "proxied" {
		t.Fatalf("body = %q", body)
	}
}

func TestNewClientWithUserAgent(t *testing.T) {
	tests := []struct {
		name      string
		clientUA  string
		requestUA string
		want      string
	}{
		{name: "default", want: "mod-downloader/dev"},
		{name: "versioned", clientUA: "mod-downloader/v1.2.3", want: "mod-downloader/v1.2.3"},
		{name: "request override", clientUA: "mod-downloader/v1.2.3", requestUA: "custom-client/1.0", want: "custom-client/1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if got := req.Header.Get("User-Agent"); got != tt.want {
					t.Errorf("User-Agent = %q, want %q", got, tt.want)
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			req, err := http.NewRequest(http.MethodGet, server.URL, nil)
			if err != nil {
				t.Fatal(err)
			}
			if tt.requestUA != "" {
				req.Header.Set("User-Agent", tt.requestUA)
			}
			resp, err := NewClientWithUserAgent(0, tt.clientUA).Do(req)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
		})
	}
}
