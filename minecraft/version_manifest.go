package minecraft

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"mod-downloader/global"
	"mod-downloader/logging"
	structs "mod-downloader/structs/minecraft"
)

const minecraftVersionManifestURL = "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json"

var pinnedMinecraftReleaseVersions = []string{
	"26.1.2",
	"1.21.1",
	"1.20.1",
	"1.19.2",
	"1.18.2",
	"1.16.5",
	"1.14.4",
	"1.12.2",
	"1.8.9",
	"1.7.10",
}

func FetchMinecraftReleaseVersions() ([]string, error) {
	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(minecraftVersionManifestURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var manifest structs.VersionManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, err
	}

	releaseVersions := make([]string, 0, len(manifest.Versions))
	for _, version := range manifest.Versions {
		if version.Type == "release" {
			releaseVersions = append(releaseVersions, version.ID)
		}
	}

	return pinMinecraftReleaseVersions(releaseVersions), nil
}

func pinMinecraftReleaseVersions(versions []string) []string {
	seen := make(map[string]bool, len(versions))
	for _, version := range versions {
		seen[version] = true
	}

	result := make([]string, 0, len(versions))
	for _, version := range pinnedMinecraftReleaseVersions {
		if seen[version] {
			result = append(result, version)
			delete(seen, version)
		}
	}

	for _, version := range versions {
		if seen[version] {
			result = append(result, version)
			delete(seen, version)
		}
	}

	return result
}

func GetMinecraftReleaseVersions() []string {
	releaseVersions := global.GetMinecraftReleaseVersions()
	if len(releaseVersions) > 0 {
		return releaseVersions
	}

	fetchedVersions, err := FetchMinecraftReleaseVersions()
	if err != nil {
		logging.Error("fetch minecraft release versions failed", "error", err)
		return nil
	}

	global.SetMinecraftReleaseVersions(fetchedVersions)
	return fetchedVersions
}
