// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package storage

import (
	"strings"

	"github.com/link-fgfgui/mod-downloader/core/logging"
)

// UpsertPinnedMod stores or replaces a pinned mod version.
func UpsertPinnedMod(p PinnedMod) error {
	d, err := readyUserDB()
	if err != nil {
		return err
	}
	p = normalizePinnedMod(p)
	if p.Platform == "" || p.ModID == "" || p.MinecraftVersion == "" || p.ModLoader == "" {
		return nil
	}

	if err := d.upsertPinnedMod(p); err != nil {
		logging.Error("upsert pinned mod failed", "platform", p.Platform, "modID", p.ModID, "versionID", p.VersionID, "minecraftVersion", p.MinecraftVersion, "modLoader", p.ModLoader, "error", err)
		return err
	}
	logging.Info("pinned mod upserted", "platform", p.Platform, "modID", p.ModID, "versionID", p.VersionID, "minecraftVersion", p.MinecraftVersion, "modLoader", p.ModLoader)
	return nil
}

// GetPinnedMod returns the pinned version for a mod in one scope.
func GetPinnedMod(platform, modID, minecraftVersion, modLoader string) (PinnedMod, bool) {
	d, err := readyUserDB()
	if err != nil {
		return PinnedMod{}, false
	}
	key := makePinnedModKey(platform, modID, minecraftVersion, modLoader)
	if key.Platform == "" || key.ModID == "" || key.MinecraftVersion == "" || key.ModLoader == "" {
		return PinnedMod{}, false
	}

	p, found, err := d.getPinnedMod(key.Platform, key.ModID, key.MinecraftVersion, key.ModLoader)
	if err != nil || !found {
		logging.Debug("pinned mod miss", "platform", key.Platform, "modID", key.ModID, "minecraftVersion", key.MinecraftVersion, "modLoader", key.ModLoader)
		return PinnedMod{}, false
	}
	logging.Debug("pinned mod hit", "platform", key.Platform, "modID", key.ModID, "versionID", p.VersionID, "minecraftVersion", key.MinecraftVersion, "modLoader", key.ModLoader)
	return p, true
}

// DeletePinnedMod removes the pin for a mod in one scope.
func DeletePinnedMod(platform, modID, minecraftVersion, modLoader string) error {
	d, err := readyUserDB()
	if err != nil {
		return err
	}
	key := makePinnedModKey(platform, modID, minecraftVersion, modLoader)
	if key.Platform == "" || key.ModID == "" || key.MinecraftVersion == "" || key.ModLoader == "" {
		return nil
	}

	if err := d.deletePinnedMod(key.Platform, key.ModID, key.MinecraftVersion, key.ModLoader); err != nil {
		logging.Error("delete pinned mod failed", "platform", key.Platform, "modID", key.ModID, "minecraftVersion", key.MinecraftVersion, "modLoader", key.ModLoader, "error", err)
		return err
	}
	logging.Info("pinned mod deleted", "platform", key.Platform, "modID", key.ModID, "minecraftVersion", key.MinecraftVersion, "modLoader", key.ModLoader)
	return nil
}

// ListPinnedMods returns a snapshot of all pinned mods, sorted in ascending stable order by Platform -> ModID -> MinecraftVersion -> ModLoader.
// The returned value is a copy; callers may safely modify it.
func ListPinnedMods() []PinnedMod {
	d, err := readyUserDB()
	if err != nil {
		return nil
	}

	pins, err := d.listPinnedMods()
	if err != nil {
		logging.Error("list pinned mods failed", "error", err)
		return nil
	}
	return pins
}

// --- favorite lists ---

// CreateFavoriteList creates an unscoped favorite list.
func CreateFavoriteList(name string) (FavoriteList, error) {
	return CreateFavoriteListForScope(name, "", "")
}

// CreateFavoriteListForScope creates a favorite list scoped to one Minecraft
// version and loader.
func CreateFavoriteListForScope(name, minecraftVersion, modLoader string) (FavoriteList, error) {
	d, err := readyUserDB()
	if err != nil {
		return FavoriteList{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return FavoriteList{}, nil
	}

	list, err := d.createFavoriteList(name, minecraftVersion, modLoader)
	if err != nil {
		logging.Error("create favorite list failed", "name", name, "error", err)
		return FavoriteList{}, err
	}
	logging.Info("favorite list created", "id", list.ID, "name", list.Name)
	return list, nil
}

// GetFavoriteListByNameAndScope looks up a list by its unique name within a
// scope.
func GetFavoriteListByNameAndScope(name, minecraftVersion, modLoader string) (FavoriteList, bool) {
	d, err := readyUserDB()
	if err != nil {
		return FavoriteList{}, false
	}
	list, found, err := d.getFavoriteListByNameAndScope(name, minecraftVersion, modLoader)
	if err != nil {
		logging.Error("get favorite list by name and scope failed", "name", name, "minecraftVersion", minecraftVersion, "modLoader", modLoader, "error", err)
		return FavoriteList{}, false
	}
	return list, found
}

// CreateFavoriteListWithModsForScope creates a scoped list and inserts mods in
// one transaction.
func CreateFavoriteListWithModsForScope(name, minecraftVersion, modLoader string, mods []FavoriteMod) (FavoriteList, []FavoriteMod, error) {
	d, err := readyUserDB()
	if err != nil {
		return FavoriteList{}, nil, err
	}
	list, saved, err := d.createFavoriteListWithMods(name, minecraftVersion, modLoader, mods)
	if err != nil {
		logging.Error("create favorite list with mods failed", "name", name, "minecraftVersion", minecraftVersion, "modLoader", modLoader, "error", err)
		return FavoriteList{}, nil, err
	}
	logging.Info("favorite list created with mods", "id", list.ID, "name", list.Name, "count", len(saved))
	return list, saved, nil
}

// UpdateFavoriteListName renames a list, preserving its group, scope and
// ordering.
func UpdateFavoriteListName(id, name string) (FavoriteList, bool, error) {
	d, err := readyUserDB()
	if err != nil {
		return FavoriteList{}, false, err
	}
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	if id == "" || name == "" {
		return FavoriteList{}, false, nil
	}

	list, found, err := d.updateFavoriteListName(id, name)
	if err != nil {
		logging.Error("rename favorite list failed", "id", id, "name", name, "error", err)
		return FavoriteList{}, false, err
	}
	if found {
		logging.Info("favorite list renamed", "id", id, "name", name)
	}
	return list, found, nil
}

// DeleteFavoriteList removes a list and reports whether it existed.
func DeleteFavoriteList(id string) (bool, error) {
	d, err := readyUserDB()
	if err != nil {
		return false, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}

	deleted, err := d.deleteFavoriteList(id)
	if err != nil {
		logging.Error("delete favorite list failed", "id", id, "error", err)
		return false, err
	}
	if deleted {
		logging.Info("favorite list deleted", "id", id)
	}
	return deleted, nil
}

// GetFavoriteList returns one list by ID.
func GetFavoriteList(id string) (FavoriteList, bool) {
	d, err := readyUserDB()
	if err != nil {
		return FavoriteList{}, false
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return FavoriteList{}, false
	}

	list, found, err := d.getFavoriteList(id)
	if err != nil || !found {
		logging.Debug("favorite list miss", "id", id)
		return FavoriteList{}, false
	}
	return list, true
}

// ListFavoriteLists returns every favorite list.
func ListFavoriteLists() []FavoriteList {
	d, err := readyUserDB()
	if err != nil {
		return nil
	}

	lists, err := d.listFavoriteLists()
	if err != nil {
		logging.Error("list favorite lists failed", "error", err)
		return nil
	}
	return lists
}

// UpdateFavoriteListMetadata updates a list's group, icon, pin and scope
// metadata.
func UpdateFavoriteListMetadata(list FavoriteList) (FavoriteList, bool, error) {
	d, err := readyUserDB()
	if err != nil {
		return FavoriteList{}, false, err
	}
	list = normalizeFavoriteList(list)
	if list.ID == "" {
		return FavoriteList{}, false, nil
	}

	updated, found, err := d.updateFavoriteListMetadata(list)
	if err != nil {
		logging.Error("update favorite list metadata failed", "id", list.ID, "error", err)
		return FavoriteList{}, false, err
	}
	if found {
		logging.Info("favorite list metadata updated", "id", updated.ID, "groupID", updated.GroupID, "pinned", updated.Pinned)
	}
	return updated, found, nil
}

// ReorderFavoriteLists persists the display order of lists.
func ReorderFavoriteLists(ids []string) error {
	d, err := readyUserDB()
	if err != nil {
		return err
	}
	if err := d.reorderFavoriteLists(ids); err != nil {
		logging.Error("reorder favorite lists failed", "error", err)
		return err
	}
	return nil
}

// CreateFavoriteGroup creates a favorite list group.
func CreateFavoriteGroup(name string) (FavoriteGroup, error) {
	d, err := readyUserDB()
	if err != nil {
		return FavoriteGroup{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return FavoriteGroup{}, nil
	}

	group, err := d.createFavoriteGroup(name)
	if err != nil {
		logging.Error("create favorite group failed", "name", name, "error", err)
		return FavoriteGroup{}, err
	}
	logging.Info("favorite group created", "id", group.ID, "name", group.Name)
	return group, nil
}

// UpdateFavoriteGroupName renames a group.
func UpdateFavoriteGroupName(id, name string) (FavoriteGroup, bool, error) {
	d, err := readyUserDB()
	if err != nil {
		return FavoriteGroup{}, false, err
	}
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	if id == "" || name == "" {
		return FavoriteGroup{}, false, nil
	}

	group, found, err := d.updateFavoriteGroupName(id, name)
	if err != nil {
		logging.Error("rename favorite group failed", "id", id, "name", name, "error", err)
		return FavoriteGroup{}, false, err
	}
	if found {
		logging.Info("favorite group renamed", "id", id, "name", name)
	}
	return group, found, nil
}

// DeleteFavoriteGroup removes a group and reports whether it existed.
func DeleteFavoriteGroup(id string) (bool, error) {
	d, err := readyUserDB()
	if err != nil {
		return false, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}

	deleted, err := d.deleteFavoriteGroup(id)
	if err != nil {
		logging.Error("delete favorite group failed", "id", id, "error", err)
		return false, err
	}
	if deleted {
		logging.Info("favorite group deleted", "id", id)
	}
	return deleted, nil
}

// ListFavoriteGroups returns every favorite list group.
func ListFavoriteGroups() []FavoriteGroup {
	d, err := readyUserDB()
	if err != nil {
		return nil
	}
	groups, err := d.listFavoriteGroups()
	if err != nil {
		logging.Error("list favorite groups failed", "error", err)
		return nil
	}
	return groups
}

// ReorderFavoriteGroups persists the display order of groups.
func ReorderFavoriteGroups(ids []string) error {
	d, err := readyUserDB()
	if err != nil {
		return err
	}
	if err := d.reorderFavoriteGroups(ids); err != nil {
		logging.Error("reorder favorite groups failed", "error", err)
		return err
	}
	return nil
}

// UpsertFavoriteMod inserts or updates a mod in a favorite list.
func UpsertFavoriteMod(mod FavoriteMod) (FavoriteMod, bool, error) {
	d, err := readyUserDB()
	if err != nil {
		return FavoriteMod{}, false, err
	}
	mod = normalizeFavoriteMod(mod)
	if mod.ListID == "" || mod.Platform == "" || mod.ModID == "" {
		return FavoriteMod{}, false, nil
	}

	saved, ok, err := d.upsertFavoriteMod(mod)
	if err != nil {
		logging.Error("upsert favorite mod failed", "listID", mod.ListID, "platform", mod.Platform, "modID", mod.ModID, "error", err)
		return FavoriteMod{}, false, err
	}
	if ok {
		logging.Info("favorite mod upserted", "listID", saved.ListID, "platform", saved.Platform, "modID", saved.ModID)
	}
	return copyFavoriteMod(saved), ok, nil
}

// DeleteFavoriteMod removes a mod from a list in one scope.
func DeleteFavoriteMod(listID, platform, modID, minecraftVersion, modLoader string) (bool, error) {
	d, err := readyUserDB()
	if err != nil {
		return false, err
	}
	key := makeFavoriteModKey(listID, platform, modID, minecraftVersion, modLoader)
	if key.ListID == "" || key.Platform == "" || key.ModID == "" {
		return false, nil
	}

	deleted, err := d.deleteFavoriteMod(key.ListID, key.Platform, key.ModID, key.MinecraftVersion, key.ModLoader)
	if err != nil {
		logging.Error("delete favorite mod failed", "listID", key.ListID, "platform", key.Platform, "modID", key.ModID, "error", err)
		return false, err
	}
	if deleted {
		logging.Info("favorite mod deleted", "listID", key.ListID, "platform", key.Platform, "modID", key.ModID)
	}
	return deleted, nil
}

// IncrementUsageStat atomically adds delta to a usage counter.
func IncrementUsageStat(key UsageStatKey, delta int64) error {
	if delta <= 0 || !validUsageStatKey(key) {
		return nil
	}
	d, err := readyUserDB()
	if err != nil {
		return err
	}
	if err := d.incrementUsageStat(key, delta); err != nil {
		logging.Error("increment usage stat failed", "key", key, "delta", delta, "error", err)
		return err
	}
	return nil
}

// GetUsageStats returns a snapshot of all usage counters.
func GetUsageStats() UsageStats {
	d, err := readyUserDB()
	if err != nil {
		return UsageStats{}
	}
	values, err := d.usageStatValues()
	if err != nil {
		logging.Error("list usage stats failed", "error", err)
		return UsageStats{}
	}
	return UsageStats{
		DownloadsCompleted: values[UsageStatDownloadsCompleted],
		ModsEnabled:        values[UsageStatModsEnabled],
		ModsDisabled:       values[UsageStatModsDisabled],
		ModsDeleted:        values[UsageStatModsDeleted],
		FavoritesAdded:     values[UsageStatFavoritesAdded],
		PackwizExports:     values[UsageStatPackwizExports],
	}
}

func validUsageStatKey(key UsageStatKey) bool {
	switch key {
	case UsageStatDownloadsCompleted,
		UsageStatModsEnabled,
		UsageStatModsDisabled,
		UsageStatModsDeleted,
		UsageStatFavoritesAdded,
		UsageStatPackwizExports:
		return true
	default:
		return false
	}
}

// ListFavoriteMods returns the mods stored directly in one list.
func ListFavoriteMods(listID string) []FavoriteMod {
	d, err := readyUserDB()
	if err != nil {
		return nil
	}
	listID = strings.TrimSpace(listID)
	if listID == "" {
		return nil
	}

	mods, err := d.listFavoriteMods(listID)
	if err != nil {
		logging.Error("list favorite mods failed", "listID", listID, "error", err)
		return nil
	}
	return mods
}

// CreateFavoriteListRef adds a parent-to-child list reference, rejecting
// cycles.
func CreateFavoriteListRef(parentListID, childListID string) (FavoriteListRef, bool, error) {
	d, err := readyUserDB()
	if err != nil {
		return FavoriteListRef{}, false, err
	}
	parentListID = strings.TrimSpace(parentListID)
	childListID = strings.TrimSpace(childListID)
	if parentListID == "" || childListID == "" {
		return FavoriteListRef{}, false, nil
	}

	ref, ok, err := d.createFavoriteListRef(parentListID, childListID)
	if err != nil {
		logging.Error("create favorite list reference failed", "parentListID", parentListID, "childListID", childListID, "error", err)
		return FavoriteListRef{}, false, err
	}
	if ok {
		logging.Info("favorite list reference created", "parentListID", ref.ParentListID, "childListID", ref.ChildListID)
	}
	return ref, ok, nil
}

// DeleteFavoriteListRef removes a parent-to-child list reference.
func DeleteFavoriteListRef(parentListID, childListID string) (bool, error) {
	d, err := readyUserDB()
	if err != nil {
		return false, err
	}
	parentListID = strings.TrimSpace(parentListID)
	childListID = strings.TrimSpace(childListID)
	if parentListID == "" || childListID == "" {
		return false, nil
	}

	deleted, err := d.deleteFavoriteListRef(parentListID, childListID)
	if err != nil {
		logging.Error("delete favorite list reference failed", "parentListID", parentListID, "childListID", childListID, "error", err)
		return false, err
	}
	if deleted {
		logging.Info("favorite list reference deleted", "parentListID", parentListID, "childListID", childListID)
	}
	return deleted, nil
}

// ListFavoriteListRefs returns the direct child references of a list.
func ListFavoriteListRefs(parentListID string) []FavoriteListRef {
	d, err := readyUserDB()
	if err != nil {
		return nil
	}
	parentListID = strings.TrimSpace(parentListID)
	if parentListID == "" {
		return nil
	}

	refs, err := d.listFavoriteListRefs(parentListID)
	if err != nil {
		logging.Error("list favorite list references failed", "parentListID", parentListID, "error", err)
		return nil
	}
	return refs
}

// ListFavoriteContents resolves one list, including inherited mods from
// referenced lists.
func ListFavoriteContents(listID string) FavoriteListContents {
	d, err := readyUserDB()
	if err != nil {
		return FavoriteListContents{}
	}
	listID = strings.TrimSpace(listID)
	if listID == "" {
		return FavoriteListContents{}
	}

	contents, err := d.listFavoriteContents(listID)
	if err != nil {
		logging.Error("list favorite contents failed", "listID", listID, "error", err)
		return FavoriteListContents{}
	}
	return contents
}

// --- version mod IDs ---
