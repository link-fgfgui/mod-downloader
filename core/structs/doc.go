// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

// Package structs defines the adapter-neutral request, result and settings
// projections exchanged across the core/UI boundary.
//
// These types are the single data contract shared by the Wails GUI and the
// tview TUI. JSON tags are wire names. Adapters must not declare parallel
// copies; domain models live in package models, and UI-facing derived
// projections live in package view.
//
// StartupStatus and the StartupStage* constants define the typed readiness
// contract returned by the host. ExtensionProjectPayload is the decoded browser
// extension payload consumed by appcore.Service.AcceptExtensionProjects.
package structs
