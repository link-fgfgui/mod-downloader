package filetransfer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type remoteInfo struct {
	size           int64
	rangeSupported bool
}

func probeRemote(ctx context.Context, client *http.Client, req Request) (remoteInfo, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, req.URL, nil)
	if err != nil {
		return remoteInfo{}, err
	}
	applyHeaders(httpReq, req.Headers)
	httpReq.Header.Set("Range", "bytes=0-0")
	resp, err := client.Do(httpReq)
	if err != nil {
		return remoteInfo{}, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusPartialContent:
		size, err := contentRangeSize(resp.Header.Get("Content-Range"))
		if err != nil {
			return remoteInfo{}, err
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		return remoteInfo{size: size, rangeSupported: true}, nil
	case http.StatusOK:
		return remoteInfo{size: resp.ContentLength}, nil
	case http.StatusRequestedRangeNotSatisfiable:
		size, err := contentRangeSize(resp.Header.Get("Content-Range"))
		if err == nil && size == 0 {
			return remoteInfo{size: 0, rangeSupported: true}, nil
		}
	}
	return remoteInfo{}, fmt.Errorf("probe download failed: %s", resp.Status)
}

func contentRangeSize(value string) (int64, error) {
	_, sizeText, ok := strings.Cut(strings.TrimSpace(value), "/")
	if !ok {
		return 0, fmt.Errorf("invalid content-range: %s", value)
	}
	sizeText = strings.TrimSpace(sizeText)
	if sizeText == "" || sizeText == "*" {
		return 0, fmt.Errorf("unknown content size")
	}
	size, err := strconv.ParseInt(sizeText, 10, 64)
	if err != nil || size < 0 {
		return 0, fmt.Errorf("invalid content size %q", sizeText)
	}
	return size, nil
}
