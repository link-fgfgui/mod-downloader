package appcore

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/link-fgfgui/mod-downloader-core/configs"
	"github.com/link-fgfgui/mod-downloader-core/global"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestRateLimitedTransportHonorsCanceledWait(t *testing.T) {
	calls := 0
	transport := rateLimitedTransport{
		limiter: newRequestRateLimiter(1),
		base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			return &http.Response{
				Status:     "200 OK",
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("{}")),
				Header:     make(http.Header),
			}, nil
		}),
	}
	first, err := http.NewRequest(http.MethodGet, "https://example.test/first", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := transport.RoundTrip(first)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	second, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.test/second", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transport.RoundTrip(second); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("RoundTrip() error = %v, want deadline exceeded", err)
	}
	if calls != 1 {
		t.Fatalf("base transport calls = %d, want 1", calls)
	}
}

func TestUserAgentForVersion(t *testing.T) {
	if got := userAgentForVersion(" v1.2.3 "); got != "mod-downloader/v1.2.3" {
		t.Fatalf("userAgentForVersion() = %q", got)
	}
	if got := userAgentForVersion(""); got != "mod-downloader/dev" {
		t.Fatalf("default userAgentForVersion() = %q", got)
	}
}

func TestCurseForgeTransportSetsHeaders(t *testing.T) {
	transport := curseForgeTransport{
		apiKey:    "secret",
		userAgent: "mod-downloader/v1.2.3",
		base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Header.Get("Accept"); got != "application/json" {
				t.Fatalf("Accept = %q", got)
			}
			if got := req.Header.Get("x-api-key"); got != "secret" {
				t.Fatalf("x-api-key = %q", got)
			}
			if got := req.Header.Get("User-Agent"); got != "mod-downloader/v1.2.3" {
				t.Fatalf("User-Agent = %q", got)
			}
			return &http.Response{
				Status:     "200 OK",
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("{}")),
				Header:     make(http.Header),
			}, nil
		}),
	}
	req, err := http.NewRequest(http.MethodGet, "https://api.curseforge.com/v1/mods/1", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if req.Header.Get("User-Agent") != "" {
		t.Fatal("transport mutated the caller request")
	}
}

func TestCurseForgeTransportRewritesMCIMURLWithoutAPIKey(t *testing.T) {
	transport := curseForgeTransport{
		userAgent:  "mod-downloader/v1.2.3",
		apiBaseURL: mcimCurseForgeBaseURL,
		base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.URL.String(); got != "https://mod.mcimirror.top/curseforge/v1/mods/1?test=true" {
				t.Fatalf("request URL = %q", got)
			}
			if got := req.Host; got != "mod.mcimirror.top" {
				t.Fatalf("request Host = %q, want %q", got, "mod.mcimirror.top")
			}
			if got := req.Header.Get("User-Agent"); got != "mod-downloader/v1.2.3" {
				t.Fatalf("User-Agent = %q", got)
			}
			if got := req.Header.Get("x-api-key"); got != "" {
				t.Fatalf("x-api-key = %q, want empty", got)
			}
			return &http.Response{
				Status:     "200 OK",
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("{}")),
				Header:     make(http.Header),
			}, nil
		}),
	}
	req, err := http.NewRequest(http.MethodGet, "https://api.curseforge.com/v1/mods/1?test=true", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if got := req.URL.String(); got != "https://api.curseforge.com/v1/mods/1?test=true" {
		t.Fatalf("caller request URL mutated to %q", got)
	}
}

func TestConfigureProviderClientsUsesVersionedUserAgent(t *testing.T) {
	svc := New(Options{Config: &configs.Config{}, Version: "v1.2.3"})
	svc.configureProviderClients()
	t.Cleanup(func() {
		global.SetModrinthClient(nil)
		global.SetCurseForgeClient(nil)
	})

	client := global.GetModrinthClient()
	if client == nil || client.UserAgent != "mod-downloader/v1.2.3" {
		t.Fatalf("Modrinth client = %#v", client)
	}
}

func TestConfigureProviderClientsUsesMCIMWithoutCurseForgeKey(t *testing.T) {
	svc := New(Options{Config: &configs.Config{Prefers: configs.Preferences{MCIMEnabled: true}}, Version: "v1.2.3"})
	svc.configureProviderClients()
	t.Cleanup(func() {
		global.SetModrinthClient(nil)
		global.SetCurseForgeClient(nil)
	})

	if global.GetCurseForgeClient() == nil {
		t.Fatal("CurseForge client is nil with MCIM enabled")
	}
	client := global.GetModrinthClient()
	if client == nil || client.BaseURL == nil || client.BaseURL.String() != mcimModrinthBaseURL {
		t.Fatalf("Modrinth MCIM client = %#v", client)
	}
	if client.UserAgent != "mod-downloader/v1.2.3" {
		t.Fatalf("Modrinth MCIM User-Agent = %q", client.UserAgent)
	}
}
