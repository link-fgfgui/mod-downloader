package downloader

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"mod-downloader/logging"
)

const discardBufferSize = 32 * 1024

func DiscardFetchFromNetwork(downloadURL string, curseForgeAPIKey string) {
	downloadURL = strings.TrimSpace(downloadURL)
	if downloadURL == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		logging.Debug("discard fetch request creation failed", "url", downloadURL, "error", err)
		return
	}
	req.Header.Set("User-Agent", "mod-downloader")

	if curseForgeAPIKey != "" && isCurseForgeCDNURL(downloadURL) {
		req.Header.Set("x-api-key", curseForgeAPIKey)
	}

	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		logging.Debug("discard fetch request failed (ignored)", "url", downloadURL, "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logging.Debug("discard fetch non-success status (ignored)", "url", downloadURL, "status", resp.StatusCode)
		return
	}

	buf := make([]byte, discardBufferSize)
	_, _ = io.CopyBuffer(io.Discard, resp.Body, buf)
	logging.Debug("discard fetch completed", "url", downloadURL)
}
