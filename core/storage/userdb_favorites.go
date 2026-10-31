package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

func (s *userStore) createFavoriteList(name, minecraftVersion, modLoader string) (FavoriteList, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return FavoriteList{}, nil
	}
	now := time.Now().Unix()
	var maxSortOrder int
	if err := s.db.QueryRow(`SELECT COALESCE(MAX(sort_order), 0) FROM favorite_lists`).Scan(&maxSortOrder); err != nil {
		return FavoriteList{}, err
	}
	list := FavoriteList{
		ID:               NewID(),
		Name:             name,
		MinecraftVersion: strings.TrimSpace(minecraftVersion),
		ModLoader:        strings.ToLower(strings.TrimSpace(modLoader)),
		CreatedAt:        now,
		UpdatedAt:        now,
		SortOrder:        maxSortOrder + 1,
	}
	_, err := s.db.Exec(
		`INSERT INTO favorite_lists(id, name, minecraft_version, mod_loader, created_at, updated_at, sort_order, system)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		list.ID, list.Name, list.MinecraftVersion, list.ModLoader, list.CreatedAt, list.UpdatedAt, list.SortOrder, boolToInt(list.System),
	)
	if err != nil {
		if isFavoriteListNameConflictError(err) {
			return FavoriteList{}, ErrFavoriteListNameConflict
		}
		return FavoriteList{}, err
	}
	return list, nil
}

func (s *userStore) getFavoriteListByNameAndScope(name, minecraftVersion, modLoader string) (FavoriteList, bool, error) {
	list := normalizeFavoriteList(FavoriteList{Name: name, MinecraftVersion: minecraftVersion, ModLoader: modLoader})
	if list.Name == "" {
		return FavoriteList{}, false, nil
	}
	found, err := scanFavoriteList(s.db.QueryRow(
		`SELECT id, group_id, name, minecraft_version, mod_loader, icon_kind, icon_value, icon_url, pinned, created_at, updated_at, sort_order, system
		 FROM favorite_lists WHERE name = ? AND minecraft_version = ? AND mod_loader = ?`,
		list.Name, list.MinecraftVersion, list.ModLoader,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return FavoriteList{}, false, nil
	}
	if err != nil {
		return FavoriteList{}, false, err
	}
	return found, true, nil
}

func (s *userStore) createFavoriteListWithMods(name, minecraftVersion, modLoader string, mods []FavoriteMod) (FavoriteList, []FavoriteMod, error) {
	list := normalizeFavoriteList(FavoriteList{Name: name, MinecraftVersion: minecraftVersion, ModLoader: modLoader})
	if list.Name == "" || list.MinecraftVersion == "" || list.ModLoader == "" || len(mods) == 0 {
		return FavoriteList{}, nil, errors.New("favorite list name, scope, and mods are required")
	}

	normalized := make([]FavoriteMod, 0, len(mods))
	seen := make(map[favoriteModKey]struct{}, len(mods))
	for _, mod := range mods {
		mod.ListID = ""
		mod.MinecraftVersion = list.MinecraftVersion
		mod.ModLoader = list.ModLoader
		mod = normalizeFavoriteMod(mod)
		if mod.Platform == "" || mod.ModID == "" {
			return FavoriteList{}, nil, errors.New("favorite mod platform and mod id are required")
		}
		key := makeFavoriteModKey("", mod.Platform, mod.ModID, mod.MinecraftVersion, mod.ModLoader)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, mod)
	}
	if len(normalized) == 0 {
		return FavoriteList{}, nil, errors.New("at least one unique favorite mod is required")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return FavoriteList{}, nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var conflict int
	err = tx.QueryRow(
		`SELECT 1 FROM favorite_lists WHERE name = ? AND minecraft_version = ? AND mod_loader = ? LIMIT 1`,
		list.Name, list.MinecraftVersion, list.ModLoader,
	).Scan(&conflict)
	if err == nil {
		return FavoriteList{}, nil, ErrFavoriteListNameConflict
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return FavoriteList{}, nil, err
	}

	now := time.Now().Unix()
	var maxSortOrder int
	if err := tx.QueryRow(`SELECT COALESCE(MAX(sort_order), 0) FROM favorite_lists`).Scan(&maxSortOrder); err != nil {
		return FavoriteList{}, nil, err
	}
	list.ID = NewID()
	list.CreatedAt = now
	list.UpdatedAt = now
	list.SortOrder = maxSortOrder + 1
	if _, err := tx.Exec(
		`INSERT INTO favorite_lists(id, group_id, name, minecraft_version, mod_loader, icon_kind, icon_value, icon_url, pinned, created_at, updated_at, sort_order, system)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		list.ID, list.GroupID, list.Name, list.MinecraftVersion, list.ModLoader, list.IconKind, list.IconValue, list.IconURL,
		boolToInt(list.Pinned), list.CreatedAt, list.UpdatedAt, list.SortOrder, boolToInt(list.System),
	); err != nil {
		if isFavoriteListNameConflictError(err) {
			return FavoriteList{}, nil, ErrFavoriteListNameConflict
		}
		return FavoriteList{}, nil, err
	}

	saved := make([]FavoriteMod, 0, len(normalized))
	for _, mod := range normalized {
		mod.ListID = list.ID
		mod.ID = NewID()
		mod.CreatedAt = now
		mod.UpdatedAt = now
		categoriesJSON, err := favoriteCategoriesJSON(mod.Categories)
		if err != nil {
			return FavoriteList{}, nil, err
		}
		if _, err := tx.Exec(
			`INSERT INTO favorite_mods(
				id, list_id, platform, mod_id, version_id, minecraft_version, mod_loader,
				title, slug, icon_url, description, categories_json, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			mod.ID, mod.ListID, mod.Platform, mod.ModID, mod.VersionID, mod.MinecraftVersion, mod.ModLoader,
			mod.Title, mod.Slug, mod.IconURL, mod.Description, categoriesJSON, mod.CreatedAt, mod.UpdatedAt,
		); err != nil {
			return FavoriteList{}, nil, err
		}
		saved = append(saved, copyFavoriteMod(mod))
	}
	if err := tx.Commit(); err != nil {
		return FavoriteList{}, nil, err
	}
	committed = true
	return list, saved, nil
}

func (s *userStore) updateFavoriteListName(id, name string) (FavoriteList, bool, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	if id == "" || name == "" {
		return FavoriteList{}, false, nil
	}
	now := time.Now().Unix()
	result, err := s.db.Exec(`UPDATE favorite_lists SET name = ?, updated_at = ? WHERE id = ?`, name, now, id)
	if err != nil {
		if isFavoriteListNameConflictError(err) {
			return FavoriteList{}, false, ErrFavoriteListNameConflict
		}
		return FavoriteList{}, false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return FavoriteList{}, false, err
	}
	if affected == 0 {
		return FavoriteList{}, false, nil
	}
	list, ok, err := s.getFavoriteList(id)
	return list, ok, err
}

func isFavoriteListNameConflictError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "unique constraint failed: favorite_lists.name")
}

func (s *userStore) getFavoriteList(id string) (FavoriteList, bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return FavoriteList{}, false, nil
	}
	list, err := scanFavoriteList(s.db.QueryRow(
		`SELECT id, group_id, name, minecraft_version, mod_loader, icon_kind, icon_value, icon_url, pinned, created_at, updated_at, sort_order, system
		 FROM favorite_lists WHERE id = ?`,
		id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return FavoriteList{}, false, nil
	}
	if err != nil {
		return FavoriteList{}, false, err
	}
	return list, true, nil
}

func (s *userStore) deleteFavoriteList(id string) (bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}
	result, err := s.db.Exec(`DELETE FROM favorite_lists WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (s *userStore) listFavoriteLists() ([]FavoriteList, error) {
	rows, err := s.db.Query(
		`SELECT id, group_id, name, minecraft_version, mod_loader, icon_kind, icon_value, icon_url, pinned, created_at, updated_at, sort_order, system
		 FROM favorite_lists
		 ORDER BY pinned DESC, sort_order ASC, created_at ASC, name ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lists []FavoriteList
	for rows.Next() {
		list, err := scanFavoriteList(rows)
		if err != nil {
			return nil, err
		}
		lists = append(lists, list)
	}
	return lists, rows.Err()
}

func (s *userStore) updateFavoriteListMetadata(list FavoriteList) (FavoriteList, bool, error) {
	list = normalizeFavoriteList(list)
	if list.ID == "" {
		return FavoriteList{}, false, nil
	}
	existing, ok, err := s.getFavoriteList(list.ID)
	if err != nil {
		return FavoriteList{}, false, err
	}
	if !ok {
		return FavoriteList{}, false, nil
	}
	if list.SortOrder == 0 {
		list.SortOrder = existing.SortOrder
	}
	now := time.Now().Unix()
	result, err := s.db.Exec(
		`UPDATE favorite_lists
		 SET group_id = ?, icon_kind = ?, icon_value = ?, icon_url = ?, pinned = ?, sort_order = ?, updated_at = ?
		 WHERE id = ?`,
		list.GroupID, list.IconKind, list.IconValue, list.IconURL, boolToInt(list.Pinned), list.SortOrder, now, list.ID,
	)
	if err != nil {
		return FavoriteList{}, false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return FavoriteList{}, false, err
	}
	if affected == 0 {
		return FavoriteList{}, false, nil
	}
	return s.getFavoriteList(list.ID)
}

func (s *userStore) reorderFavoriteLists(ids []string) error {
	return s.reorderRows("favorite_lists", ids)
}

func scanFavoriteList(scanner favoriteListScanner) (FavoriteList, error) {
	var list FavoriteList
	var pinned, system int
	if err := scanner.Scan(
		&list.ID,
		&list.GroupID,
		&list.Name,
		&list.MinecraftVersion,
		&list.ModLoader,
		&list.IconKind,
		&list.IconValue,
		&list.IconURL,
		&pinned,
		&list.CreatedAt,
		&list.UpdatedAt,
		&list.SortOrder,
		&system,
	); err != nil {
		return FavoriteList{}, err
	}
	list.Pinned = pinned != 0
	list.System = system != 0
	return normalizeFavoriteList(list), nil
}

func (s *userStore) createFavoriteGroup(name string) (FavoriteGroup, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return FavoriteGroup{}, nil
	}
	now := time.Now().Unix()
	var maxSortOrder int
	if err := s.db.QueryRow(`SELECT COALESCE(MAX(sort_order), 0) FROM favorite_groups`).Scan(&maxSortOrder); err != nil {
		return FavoriteGroup{}, err
	}
	group := FavoriteGroup{
		ID:        NewID(),
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
		SortOrder: maxSortOrder + 1,
	}
	_, err := s.db.Exec(
		`INSERT INTO favorite_groups(id, name, created_at, updated_at, sort_order)
		 VALUES (?, ?, ?, ?, ?)`,
		group.ID, group.Name, group.CreatedAt, group.UpdatedAt, group.SortOrder,
	)
	if err != nil {
		return FavoriteGroup{}, err
	}
	return group, nil
}

func (s *userStore) updateFavoriteGroupName(id, name string) (FavoriteGroup, bool, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	if id == "" || name == "" {
		return FavoriteGroup{}, false, nil
	}
	now := time.Now().Unix()
	result, err := s.db.Exec(`UPDATE favorite_groups SET name = ?, updated_at = ? WHERE id = ?`, name, now, id)
	if err != nil {
		return FavoriteGroup{}, false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return FavoriteGroup{}, false, err
	}
	if affected == 0 {
		return FavoriteGroup{}, false, nil
	}
	return s.getFavoriteGroup(id)
}

func (s *userStore) getFavoriteGroup(id string) (FavoriteGroup, bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return FavoriteGroup{}, false, nil
	}
	group, err := scanFavoriteGroup(s.db.QueryRow(
		`SELECT id, name, created_at, updated_at, sort_order
		 FROM favorite_groups WHERE id = ?`,
		id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return FavoriteGroup{}, false, nil
	}
	if err != nil {
		return FavoriteGroup{}, false, err
	}
	return group, true, nil
}

func (s *userStore) deleteFavoriteGroup(id string) (bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.Exec(`UPDATE favorite_lists SET group_id = '', updated_at = ? WHERE group_id = ?`, time.Now().Unix(), id); err != nil {
		return false, err
	}
	result, err := tx.Exec(`DELETE FROM favorite_groups WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	committed = true
	return affected > 0, nil
}

func (s *userStore) listFavoriteGroups() ([]FavoriteGroup, error) {
	rows, err := s.db.Query(
		`SELECT id, name, created_at, updated_at, sort_order
		 FROM favorite_groups
		 ORDER BY sort_order ASC, created_at ASC, name ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []FavoriteGroup
	for rows.Next() {
		group, err := scanFavoriteGroup(rows)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, rows.Err()
}

func (s *userStore) reorderFavoriteGroups(ids []string) error {
	return s.reorderRows("favorite_groups", ids)
}

func scanFavoriteGroup(scanner favoriteGroupScanner) (FavoriteGroup, error) {
	var group FavoriteGroup
	if err := scanner.Scan(&group.ID, &group.Name, &group.CreatedAt, &group.UpdatedAt, &group.SortOrder); err != nil {
		return FavoriteGroup{}, err
	}
	return normalizeFavoriteGroup(group), nil
}

func (s *userStore) upsertFavoriteMod(mod FavoriteMod) (FavoriteMod, bool, error) {
	mod = normalizeFavoriteMod(mod)
	if mod.ListID == "" || mod.Platform == "" || mod.ModID == "" {
		return FavoriteMod{}, false, nil
	}
	list, ok, err := s.getFavoriteList(mod.ListID)
	if err != nil {
		return FavoriteMod{}, false, err
	}
	if !ok {
		return FavoriteMod{}, false, nil
	}
	mod.MinecraftVersion = list.MinecraftVersion
	mod.ModLoader = list.ModLoader

	if existing, ok, err := s.getFavoriteModByKey(mod.ListID, mod.Platform, mod.ModID, mod.MinecraftVersion, mod.ModLoader); err != nil {
		return FavoriteMod{}, false, err
	} else if ok {
		if mod.ID == "" {
			mod.ID = existing.ID
		}
		if mod.CreatedAt == 0 {
			mod.CreatedAt = existing.CreatedAt
		}
	}

	now := time.Now().Unix()
	if mod.ID == "" {
		mod.ID = NewID()
	}
	if mod.CreatedAt == 0 {
		mod.CreatedAt = now
	}
	mod.UpdatedAt = now
	categoriesJSON, err := favoriteCategoriesJSON(mod.Categories)
	if err != nil {
		return FavoriteMod{}, false, err
	}
	_, err = s.db.Exec(
		`INSERT INTO favorite_mods(
			id, list_id, platform, mod_id, version_id, minecraft_version, mod_loader,
			title, slug, icon_url, description, categories_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(list_id, platform, mod_id, minecraft_version, mod_loader)
		DO UPDATE SET
			id = excluded.id,
			version_id = excluded.version_id,
			title = excluded.title,
			slug = excluded.slug,
			icon_url = excluded.icon_url,
			description = excluded.description,
			categories_json = excluded.categories_json,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at`,
		mod.ID, mod.ListID, mod.Platform, mod.ModID, mod.VersionID, mod.MinecraftVersion, mod.ModLoader,
		mod.Title, mod.Slug, mod.IconURL, mod.Description, categoriesJSON, mod.CreatedAt, mod.UpdatedAt,
	)
	if err != nil {
		return FavoriteMod{}, false, err
	}
	return copyFavoriteMod(mod), true, nil
}

func (s *userStore) getFavoriteModByKey(listID, platform, modID, mcVersion, modLoader string) (FavoriteMod, bool, error) {
	key := makeFavoriteModKey(listID, platform, modID, mcVersion, modLoader)
	if key.ListID == "" || key.Platform == "" || key.ModID == "" {
		return FavoriteMod{}, false, nil
	}
	return s.getFavoriteMod(
		`SELECT id, list_id, platform, mod_id, version_id, minecraft_version, mod_loader,
			title, slug, icon_url, description, categories_json, created_at, updated_at
		 FROM favorite_mods
		 WHERE list_id = ? AND platform = ? AND mod_id = ? AND minecraft_version = ? AND mod_loader = ?`,
		key.ListID, key.Platform, key.ModID, key.MinecraftVersion, key.ModLoader,
	)
}

func (s *userStore) deleteFavoriteMod(listID, platform, modID, mcVersion, modLoader string) (bool, error) {
	key := makeFavoriteModKey(listID, platform, modID, mcVersion, modLoader)
	if key.ListID == "" || key.Platform == "" || key.ModID == "" {
		return false, nil
	}
	list, ok, err := s.getFavoriteList(key.ListID)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	key.MinecraftVersion = list.MinecraftVersion
	key.ModLoader = list.ModLoader
	result, err := s.db.Exec(
		`DELETE FROM favorite_mods
		 WHERE list_id = ? AND platform = ? AND mod_id = ? AND minecraft_version = ? AND mod_loader = ?`,
		key.ListID, key.Platform, key.ModID, key.MinecraftVersion, key.ModLoader,
	)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (s *userStore) listFavoriteMods(listID string) ([]FavoriteMod, error) {
	listID = strings.TrimSpace(listID)
	if listID == "" {
		return nil, nil
	}
	rows, err := s.db.Query(
		`SELECT id, list_id, platform, mod_id, version_id, minecraft_version, mod_loader,
			title, slug, icon_url, description, categories_json, created_at, updated_at
		 FROM favorite_mods
		 WHERE list_id = ?`,
		listID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mods []FavoriteMod
	for rows.Next() {
		mod, err := scanFavoriteMod(rows)
		if err != nil {
			return nil, err
		}
		mods = append(mods, mod)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(mods, func(i, j int) bool {
		if mods[i].CreatedAt != mods[j].CreatedAt {
			return mods[i].CreatedAt < mods[j].CreatedAt
		}
		if mods[i].Platform != mods[j].Platform {
			return mods[i].Platform < mods[j].Platform
		}
		return mods[i].ModID < mods[j].ModID
	})
	return mods, nil
}

func (s *userStore) getFavoriteMod(query string, args ...any) (FavoriteMod, bool, error) {
	row := s.db.QueryRow(query, args...)
	mod, err := scanFavoriteMod(row)
	if errors.Is(err, sql.ErrNoRows) {
		return FavoriteMod{}, false, nil
	}
	if err != nil {
		return FavoriteMod{}, false, err
	}
	return mod, true, nil
}

func scanFavoriteMod(scanner favoriteModScanner) (FavoriteMod, error) {
	var mod FavoriteMod
	var categoriesJSON string
	if err := scanner.Scan(
		&mod.ID,
		&mod.ListID,
		&mod.Platform,
		&mod.ModID,
		&mod.VersionID,
		&mod.MinecraftVersion,
		&mod.ModLoader,
		&mod.Title,
		&mod.Slug,
		&mod.IconURL,
		&mod.Description,
		&categoriesJSON,
		&mod.CreatedAt,
		&mod.UpdatedAt,
	); err != nil {
		return FavoriteMod{}, err
	}
	if categoriesJSON != "" {
		if err := json.Unmarshal([]byte(categoriesJSON), &mod.Categories); err != nil {
			return FavoriteMod{}, fmt.Errorf("decode favorite categories: %w", err)
		}
	}
	mod.Categories = copyStringSlice(mod.Categories)
	return mod, nil
}

func favoriteCategoriesJSON(categories []string) (string, error) {
	categories = copyStringSlice(categories)
	if len(categories) == 0 {
		return "[]", nil
	}
	data, err := json.Marshal(categories)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
