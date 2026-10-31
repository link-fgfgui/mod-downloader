package storage

import (
	"strings"

	"github.com/link-fgfgui/mod-downloader-core/logging"
)

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

func GetPinnedMod(platform, modID, mcVersion, modLoader string) (PinnedMod, bool) {
	d, err := readyUserDB()
	if err != nil {
		return PinnedMod{}, false
	}
	key := makePinnedModKey(platform, modID, mcVersion, modLoader)
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

func DeletePinnedMod(platform, modID, mcVersion, modLoader string) error {
	d, err := readyUserDB()
	if err != nil {
		return err
	}
	key := makePinnedModKey(platform, modID, mcVersion, modLoader)
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

func CreateFavoriteList(name string) (FavoriteList, error) {
	return CreateFavoriteListForScope(name, "", "")
}

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

func DeleteFavoriteMod(listID, platform, modID, mcVersion, modLoader string) (bool, error) {
	d, err := readyUserDB()
	if err != nil {
		return false, err
	}
	key := makeFavoriteModKey(listID, platform, modID, mcVersion, modLoader)
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
