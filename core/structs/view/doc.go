// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

// Package view holds the UI-facing projections shared by the GUI and TUI
// adapters: display names, adapter ordering, streamed search state and button
// state. Types here are derived snapshots, not domain inputs. The package must
// not import core/structs; requests, domain results and protocol payloads stay
// in the parent package.
package view
