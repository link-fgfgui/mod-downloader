// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package instance

import (
	"errors"
	"testing"

	"github.com/link-fgfgui/mod-downloader/core/httpserver"
)

// TestDefaultAddrIsExtensionPort locks the guard to the extension receiver's
// port: binding a second address would let two processes run side by side.
func TestDefaultAddrIsExtensionPort(t *testing.T) {
	if DefaultAddr != httpserver.DefaultAddr {
		t.Fatalf("DefaultAddr = %q, want extension address %q", DefaultAddr, httpserver.DefaultAddr)
	}
}

func TestAcquireAndRelease(t *testing.T) {
	guard, err := Acquire("127.0.0.1:0")
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	addr := guard.Addr()
	if addr == "" {
		t.Fatal("Addr() empty after Acquire")
	}
	if guard.Listener() == nil {
		t.Fatal("Listener() nil after Acquire")
	}
	if err := guard.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := guard.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}

	reacquired, err := Acquire(addr)
	if err != nil {
		t.Fatalf("Acquire after Close: %v", err)
	}
	if err := reacquired.Close(); err != nil {
		t.Fatalf("close re-acquired guard: %v", err)
	}
}

func TestAcquireRejectsTakenPort(t *testing.T) {
	first, err := Acquire("127.0.0.1:0")
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	defer first.Close()

	second, err := Acquire(first.Addr())
	if second != nil {
		t.Fatalf("second Acquire returned guard %p, want nil", second)
	}
	if !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("second Acquire error = %v, want ErrAlreadyRunning", err)
	}
}

func TestNilGuardIsSafe(t *testing.T) {
	var guard *Guard
	if guard.Addr() != "" {
		t.Fatalf("nil guard Addr() = %q, want empty", guard.Addr())
	}
	if guard.Listener() != nil {
		t.Fatal("nil guard Listener() should be nil")
	}
	if err := guard.Close(); err != nil {
		t.Fatalf("nil guard Close: %v", err)
	}
}
