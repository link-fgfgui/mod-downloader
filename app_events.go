package main

import (
	"github.com/link-fgfgui/mod-downloader-core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader-core/structs"
	minecraft "github.com/link-fgfgui/mod-downloader-core/structs/minecraft"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func init() {
	application.RegisterEvent[string](minecraftDirChangedEvent)
	application.RegisterEvent[minecraft.VersionInfo](selectedVersionChangedEvent)
	application.RegisterEvent[appstructs.SearchModsUpdate](searchModsUpdatedEvent)
	application.RegisterEvent[appstructs.SearchModsUpdate](extensionModsAcceptedEvent)
	application.RegisterEvent[application.Void](downloadStatesUpdatedEvent)
	application.RegisterEvent[appstructs.DownloadQueueState](downloadQueueUpdatedEvent)
	application.RegisterEvent[appstructs.DownloadFailedEvent](downloadFailedEvent)
	application.RegisterEvent[application.Void](downloadCompletedEvent)
	application.RegisterEvent[storage.UsageStats](usageStatsUpdatedEvent)
}
