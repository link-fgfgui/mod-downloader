// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package structs

// Latest names the newest release and snapshot versions in Mojang's manifest.
type Latest struct {
	Release  string `json:"release"`
	Snapshot string `json:"snapshot"`
}

// Version is one Minecraft version entry in Mojang's manifest.
type Version struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// VersionManifest is the subset of Mojang's version_manifest_v2.json that the
// launcher and version pickers consume.
type VersionManifest struct {
	Latest   Latest    `json:"latest"`
	Versions []Version `json:"versions"`
}
