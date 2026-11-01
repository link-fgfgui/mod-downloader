// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"github.com/link-fgfgui/mod-downloader/core/appcore"
	"github.com/link-fgfgui/mod-downloader/core/host"
	"github.com/link-fgfgui/mod-downloader/core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader/core/structs"
	minecraft "github.com/link-fgfgui/mod-downloader/core/structs/minecraft"
	appview "github.com/link-fgfgui/mod-downloader/core/structs/view"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Frontend event registration. The wire names are appcore constants so the
// renderer contract has a single source of truth and Wails can evaluate them.
func init() {
	application.RegisterEvent[string](appcore.EventNameMinecraftDirChanged)
	application.RegisterEvent[minecraft.VersionInfo](appcore.EventNameSelectedVersionChanged)
	application.RegisterEvent[appview.SearchModsUpdate](appcore.EventNameSearchModsUpdated)
	application.RegisterEvent[appview.SearchModsUpdate](appcore.EventNameExtensionModsAccepted)
	application.RegisterEvent[application.Void](appcore.EventNameDownloadStatesUpdated)
	application.RegisterEvent[appview.DownloadQueueState](appcore.EventNameDownloadQueueUpdated)
	application.RegisterEvent[appstructs.DownloadFailedEvent](appcore.EventNameDownloadFailed)
	application.RegisterEvent[application.Void](appcore.EventNameDownloadCompleted)
	application.RegisterEvent[storage.UsageStats](appcore.EventNameUsageStatsUpdated)
}

// registerCoreEvents maps every typed core event onto the Wails event bus.
func (a *App) registerCoreEvents(h *host.Host) {
	host.On(h, appcore.EventSearchModsUpdated, func(update appview.SearchModsUpdate) {
		a.emit(appcore.EventNameSearchModsUpdated, update)
	})
	host.On(h, appcore.EventExtensionModsAccepted, func(update appview.SearchModsUpdate) {
		a.emit(appcore.EventNameExtensionModsAccepted, update)
	})
	host.OnEvent(h, appcore.EventDownloadStatesUpdated, func() {
		a.emit(appcore.EventNameDownloadStatesUpdated, nil)
	})
	host.On(h, appcore.EventDownloadQueueUpdated, func(state appview.DownloadQueueState) {
		a.emit(appcore.EventNameDownloadQueueUpdated, state)
	})
	host.On(h, appcore.EventDownloadFailed, func(failed appstructs.DownloadFailedEvent) {
		a.emit(appcore.EventNameDownloadFailed, failed)
	})
	host.OnEvent(h, appcore.EventDownloadCompleted, func() {
		a.emit(appcore.EventNameDownloadCompleted, nil)
	})
	host.On(h, appcore.EventMinecraftDirChanged, func(dir string) {
		a.emit(appcore.EventNameMinecraftDirChanged, dir)
	})
	host.On(h, appcore.EventSelectedVersionChanged, func(version minecraft.VersionInfo) {
		a.emit(appcore.EventNameSelectedVersionChanged, version)
	})
	host.OnEvent(h, appcore.EventUsageStatsUpdated, func() {
		a.emit(appcore.EventNameUsageStatsUpdated, nil)
	})
}
