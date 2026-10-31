package appcore

import (
	"errors"
	"strings"

	"github.com/link-fgfgui/mod-downloader-core/global"
	structs "github.com/link-fgfgui/mod-downloader-core/structs/minecraft"
)

const (
	connectorModID  = "connector"
	connectorLoader = "fabric"
)

func initializeActualModLoader(version structs.VersionInfo) structs.VersionInfo {
	actual := normalizeLoader(version.ActualModLoader)
	if actual == "" {
		actual = normalizeLoader(version.ModLoader)
	}
	version.ActualModLoader = actual
	version.ModLoader = normalizeLoader(version.ModLoader)
	return version
}

func resetConnectorVersion(version structs.VersionInfo) structs.VersionInfo {
	version = initializeActualModLoader(version)
	version.ModLoader = version.ActualModLoader
	version.ConnectorAvailable = false
	version.ConnectorVirtual = false
	return version
}

func reconcileConnectorVersion(version structs.VersionInfo) structs.VersionInfo {
	version = initializeActualModLoader(version)
	available := connectorHostLoader(version.ActualModLoader) && enabledConnector(version.Mods)
	version.ConnectorAvailable = available
	if !available {
		version.ModLoader = version.ActualModLoader
		version.ConnectorVirtual = false
		return version
	}
	if version.ConnectorVirtual {
		version.ModLoader = connectorLoader
		return version
	}
	version.ModLoader = version.ActualModLoader
	return version
}

func connectorHostLoader(loader string) bool {
	switch normalizeLoader(loader) {
	case "forge", "neoforge":
		return true
	default:
		return false
	}
}

func enabledConnector(mods []structs.ModInfo) bool {
	for _, mod := range mods {
		if mod.Enabled && strings.EqualFold(strings.TrimSpace(mod.ID), connectorModID) {
			return true
		}
	}
	return false
}

func normalizeLoader(loader string) string {
	return strings.ToLower(strings.TrimSpace(loader))
}

func (s *Service) ToggleConnectorLoader() (structs.VersionInfo, error) {
	selected := global.GetSelectedVersion()
	if selected.ID == "" && selected.Name == "" {
		return structs.VersionInfo{}, errors.New("no selected version")
	}
	selected = reconcileConnectorVersion(selected)
	if !selected.ConnectorAvailable {
		return structs.VersionInfo{}, errors.New("connector unavailable")
	}
	selected.ConnectorVirtual = !selected.ConnectorVirtual
	if selected.ConnectorVirtual {
		selected.ModLoader = connectorLoader
	} else {
		selected.ModLoader = selected.ActualModLoader
	}
	global.SetSelectedVersion(selected)
	s.emit(EventSelectedVersionChanged, selected)
	return selected, nil
}
