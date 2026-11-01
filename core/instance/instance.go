// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

// Package instance owns the process-wide single-instance guard. The guard is
// a bound TCP socket on the extension port: the first mod-downloader process
// (GUI or TUI) owns it, and every later launch fails to bind and aborts before
// creating a window or opening shared storage.
package instance

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/link-fgfgui/mod-downloader/core/httpserver"
)

// DefaultAddr is the sentinel address shared by every frontend. It is the same
// loopback port the browser extension posts to (httpserver.DefaultAddr), so the
// guard and the extension receiver cannot be owned by two processes.
const DefaultAddr = httpserver.DefaultAddr

// ErrAlreadyRunning reports that the singleton port is not available, which
// normally means another mod-downloader process already owns it. A foreign
// program holding the port produces the same error: the port is the lock, so
// this process must not start in either case.
var ErrAlreadyRunning = errors.New("mod-downloader is already running")

// Guard owns the bound singleton socket until Close is called.
type Guard struct {
	listener  net.Listener
	closeOnce sync.Once
	closeErr  error
}

// Acquire binds addr as the process-wide guard. An empty addr falls back to
// DefaultAddr. Every bind failure is reported as ErrAlreadyRunning so callers
// abort startup instead of opening the same state twice.
func Acquire(addr string) (*Guard, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		addr = DefaultAddr
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAlreadyRunning, err)
	}
	return &Guard{listener: listener}, nil
}

// Listener returns the bound socket. The extension HTTP server serves on it so
// the guard is the only bind of the singleton port.
func (g *Guard) Listener() net.Listener {
	if g == nil {
		return nil
	}
	return g.listener
}

// Addr returns the actual bound address, or "" before acquisition.
func (g *Guard) Addr() string {
	if g == nil || g.listener == nil {
		return ""
	}
	return g.listener.Addr().String()
}

// Close releases the guard. It is safe to call more than once.
func (g *Guard) Close() error {
	if g == nil || g.listener == nil {
		return nil
	}
	g.closeOnce.Do(func() {
		g.closeErr = g.listener.Close()
	})
	return g.closeErr
}
