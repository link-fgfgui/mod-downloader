package storage

import (
	"strings"

	"github.com/link-fgfgui/mod-downloader-core/models"
)

type PlatformAssociation struct {
	ID                  string `json:"id"`
	CurseForgeProjectID string `json:"curseforgeProjectId,omitempty"`
	ModrinthProjectID   string `json:"modrinthProjectId,omitempty"`
}

type CachedModVersionCandidate struct {
	Project models.ModProject
	Version models.ModVersion
}

type PinnedMod struct {
	ID               string `json:"id"`
	Platform         string `json:"platform"`
	ModID            string `json:"modId"`
	VersionID        string `json:"versionId"`
	MinecraftVersion string `json:"minecraftVersion"`
	ModLoader        string `json:"modLoader"`
}

type FavoriteList struct {
	ID               string `json:"id"`
	GroupID          string `json:"groupId,omitempty"`
	Name             string `json:"name"`
	MinecraftVersion string `json:"minecraftVersion,omitempty"`
	ModLoader        string `json:"modLoader,omitempty"`
	IconKind         string `json:"iconKind,omitempty"`
	IconValue        string `json:"iconValue,omitempty"`
	IconURL          string `json:"iconUrl,omitempty"`
	Pinned           bool   `json:"pinned,omitempty"`
	CreatedAt        int64  `json:"createdAt"`
	UpdatedAt        int64  `json:"updatedAt"`
	SortOrder        int    `json:"sortOrder"`
	System           bool   `json:"system,omitempty"`
}

type FavoriteGroup struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
	SortOrder int    `json:"sortOrder"`
}

type FavoriteListRef struct {
	ID           string `json:"id"`
	ParentListID string `json:"parentListId"`
	ChildListID  string `json:"childListId"`
	CreatedAt    int64  `json:"createdAt"`
	UpdatedAt    int64  `json:"updatedAt"`
	SortOrder    int    `json:"sortOrder"`
}

type FavoriteMod struct {
	ID               string   `json:"id"`
	ListID           string   `json:"listId"`
	Platform         string   `json:"platform"`
	ModID            string   `json:"modId"`
	VersionID        string   `json:"versionId,omitempty"`
	MinecraftVersion string   `json:"minecraftVersion,omitempty"`
	ModLoader        string   `json:"modLoader,omitempty"`
	Title            string   `json:"title,omitempty"`
	Slug             string   `json:"slug,omitempty"`
	IconURL          string   `json:"iconUrl,omitempty"`
	Description      string   `json:"description,omitempty"`
	Categories       []string `json:"categories,omitempty"`
	CreatedAt        int64    `json:"createdAt"`
	UpdatedAt        int64    `json:"updatedAt"`
}

type FavoriteModEntry struct {
	FavoriteMod
	SourceListID   string `json:"sourceListId,omitempty"`
	SourceListName string `json:"sourceListName,omitempty"`
	Referenced     bool   `json:"referenced,omitempty"`
}

type FavoriteListContents struct {
	ListID string             `json:"listId"`
	Mods   []FavoriteModEntry `json:"mods"`
	Refs   []FavoriteListRef  `json:"refs,omitempty"`
}

type UsageStatKey string

const (
	UsageStatDownloadsCompleted UsageStatKey = "downloads_completed"
	UsageStatModsEnabled        UsageStatKey = "mods_enabled"
	UsageStatModsDisabled       UsageStatKey = "mods_disabled"
	UsageStatModsDeleted        UsageStatKey = "mods_deleted"
	UsageStatFavoritesAdded     UsageStatKey = "favorites_added"
	UsageStatPackwizExports     UsageStatKey = "packwiz_exports"
)

type UsageStats struct {
	DownloadsCompleted int64 `json:"downloadsCompleted"`
	ModsEnabled        int64 `json:"modsEnabled"`
	ModsDisabled       int64 `json:"modsDisabled"`
	ModsDeleted        int64 `json:"modsDeleted"`
	FavoritesAdded     int64 `json:"favoritesAdded"`
	PackwizExports     int64 `json:"packwizExports"`
}

type ModPlatformVersionScope struct {
	MinecraftVersion string
	ModLoader        string
}

type storedVersionScope struct {
	Platform         string `json:"platform"`
	ProjectID        string `json:"projectId"`
	MinecraftVersion string `json:"minecraftVersion"`
	ModLoader        string `json:"modLoader"`
	UpdatedAt        int64  `json:"updatedAt"`
}

func normalizePinnedModKey(platform, modID, mcVersion, modLoader string) (string, string, string, string) {
	return strings.ToLower(strings.TrimSpace(platform)),
		strings.ToLower(strings.TrimSpace(modID)),
		strings.TrimSpace(mcVersion),
		strings.ToLower(strings.TrimSpace(modLoader))
}

func normalizePinnedMod(p PinnedMod) PinnedMod {
	p.Platform, p.ModID, p.MinecraftVersion, p.ModLoader = normalizePinnedModKey(p.Platform, p.ModID, p.MinecraftVersion, p.ModLoader)
	p.ID = strings.TrimSpace(p.ID)
	p.VersionID = strings.TrimSpace(p.VersionID)
	return p
}

func normalizeFavoriteList(list FavoriteList) FavoriteList {
	list.ID = strings.TrimSpace(list.ID)
	list.GroupID = strings.TrimSpace(list.GroupID)
	list.Name = strings.TrimSpace(list.Name)
	list.MinecraftVersion = strings.TrimSpace(list.MinecraftVersion)
	list.ModLoader = strings.ToLower(strings.TrimSpace(list.ModLoader))
	list.IconKind = strings.ToLower(strings.TrimSpace(list.IconKind))
	list.IconValue = strings.TrimSpace(list.IconValue)
	list.IconURL = strings.TrimSpace(list.IconURL)
	return list
}

func normalizeFavoriteGroup(group FavoriteGroup) FavoriteGroup {
	group.ID = strings.TrimSpace(group.ID)
	group.Name = strings.TrimSpace(group.Name)
	return group
}

func normalizeFavoriteListRef(ref FavoriteListRef) FavoriteListRef {
	ref.ID = strings.TrimSpace(ref.ID)
	ref.ParentListID = strings.TrimSpace(ref.ParentListID)
	ref.ChildListID = strings.TrimSpace(ref.ChildListID)
	return ref
}

func normalizeFavoriteMod(mod FavoriteMod) FavoriteMod {
	mod.ID = strings.TrimSpace(mod.ID)
	mod.ListID = strings.TrimSpace(mod.ListID)
	mod.Platform, mod.ModID, mod.MinecraftVersion, mod.ModLoader = normalizePinnedModKey(mod.Platform, mod.ModID, mod.MinecraftVersion, mod.ModLoader)
	mod.VersionID = strings.TrimSpace(mod.VersionID)
	mod.Title = strings.TrimSpace(mod.Title)
	mod.Slug = strings.TrimSpace(mod.Slug)
	mod.IconURL = strings.TrimSpace(mod.IconURL)
	mod.Description = strings.TrimSpace(mod.Description)
	mod.Categories = models.NormalizeCategories(mod.Categories)
	return mod
}

func makePlatformKey(platform, projectID string) platformKey {
	return platformKey{Platform: normalizePlatform(platform), ProjectID: strings.TrimSpace(projectID)}
}

func makeVersionKey(platform, projectID, versionID string) versionKey {
	return versionKey{Platform: normalizePlatform(platform), ProjectID: strings.TrimSpace(projectID), VersionID: strings.TrimSpace(versionID)}
}

func makePinnedModKey(platform, modID, mcVersion, modLoader string) pinnedModKey {
	platform, modID, mcVersion, modLoader = normalizePinnedModKey(platform, modID, mcVersion, modLoader)
	return pinnedModKey{Platform: platform, ModID: modID, MinecraftVersion: mcVersion, ModLoader: modLoader}
}

func makeFavoriteModKey(listID, platform, modID, mcVersion, modLoader string) favoriteModKey {
	platform, modID, mcVersion, modLoader = normalizePinnedModKey(platform, modID, mcVersion, modLoader)
	return favoriteModKey{
		ListID:           strings.TrimSpace(listID),
		Platform:         platform,
		ModID:            modID,
		MinecraftVersion: mcVersion,
		ModLoader:        modLoader,
	}
}

func makeVersionScopeKey(platform, projectID string, scope ModPlatformVersionScope) versionScopeKey {
	return versionScopeKey{
		Platform:         normalizePlatform(platform),
		ProjectID:        strings.TrimSpace(projectID),
		MinecraftVersion: scope.MinecraftVersion,
		ModLoader:        scope.ModLoader,
	}
}

func normalizePlatform(platform string) string {
	return strings.ToLower(strings.TrimSpace(platform))
}

// --- mod platforms ---
