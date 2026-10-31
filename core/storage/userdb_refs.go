package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (s *userStore) createFavoriteListRef(parentListID, childListID string) (FavoriteListRef, bool, error) {
	ref := normalizeFavoriteListRef(FavoriteListRef{
		ParentListID: parentListID,
		ChildListID:  childListID,
	})
	if ref.ParentListID == "" || ref.ChildListID == "" {
		return FavoriteListRef{}, false, nil
	}
	if ref.ParentListID == ref.ChildListID {
		return FavoriteListRef{}, false, errFavoriteListRefCycle
	}
	if _, ok, err := s.getFavoriteList(ref.ParentListID); err != nil {
		return FavoriteListRef{}, false, err
	} else if !ok {
		return FavoriteListRef{}, false, nil
	}
	if _, ok, err := s.getFavoriteList(ref.ChildListID); err != nil {
		return FavoriteListRef{}, false, err
	} else if !ok {
		return FavoriteListRef{}, false, nil
	}
	if existing, ok, err := s.getFavoriteListRef(ref.ParentListID, ref.ChildListID); err != nil {
		return FavoriteListRef{}, false, err
	} else if ok {
		return existing, true, nil
	}
	reachesParent, err := s.favoriteListCanReach(ref.ChildListID, ref.ParentListID)
	if err != nil {
		return FavoriteListRef{}, false, err
	}
	if reachesParent {
		return FavoriteListRef{}, false, errFavoriteListRefCycle
	}

	now := time.Now().Unix()
	var maxSortOrder int
	if err := s.db.QueryRow(`SELECT COALESCE(MAX(sort_order), 0) FROM favorite_list_refs WHERE parent_list_id = ?`, ref.ParentListID).Scan(&maxSortOrder); err != nil {
		return FavoriteListRef{}, false, err
	}
	ref.ID = NewID()
	ref.CreatedAt = now
	ref.UpdatedAt = now
	ref.SortOrder = maxSortOrder + 1
	_, err = s.db.Exec(
		`INSERT INTO favorite_list_refs(id, parent_list_id, child_list_id, created_at, updated_at, sort_order)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		ref.ID, ref.ParentListID, ref.ChildListID, ref.CreatedAt, ref.UpdatedAt, ref.SortOrder,
	)
	if err != nil {
		return FavoriteListRef{}, false, err
	}
	return ref, true, nil
}

func (s *userStore) getFavoriteListRef(parentListID, childListID string) (FavoriteListRef, bool, error) {
	parentListID = strings.TrimSpace(parentListID)
	childListID = strings.TrimSpace(childListID)
	if parentListID == "" || childListID == "" {
		return FavoriteListRef{}, false, nil
	}
	ref, err := scanFavoriteListRef(s.db.QueryRow(
		`SELECT id, parent_list_id, child_list_id, created_at, updated_at, sort_order
		 FROM favorite_list_refs
		 WHERE parent_list_id = ? AND child_list_id = ?`,
		parentListID, childListID,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return FavoriteListRef{}, false, nil
	}
	if err != nil {
		return FavoriteListRef{}, false, err
	}
	return ref, true, nil
}

func (s *userStore) deleteFavoriteListRef(parentListID, childListID string) (bool, error) {
	parentListID = strings.TrimSpace(parentListID)
	childListID = strings.TrimSpace(childListID)
	if parentListID == "" || childListID == "" {
		return false, nil
	}
	result, err := s.db.Exec(
		`DELETE FROM favorite_list_refs
		 WHERE parent_list_id = ? AND child_list_id = ?`,
		parentListID, childListID,
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

func (s *userStore) listFavoriteListRefs(parentListID string) ([]FavoriteListRef, error) {
	parentListID = strings.TrimSpace(parentListID)
	if parentListID == "" {
		return nil, nil
	}
	rows, err := s.db.Query(
		`SELECT id, parent_list_id, child_list_id, created_at, updated_at, sort_order
		 FROM favorite_list_refs
		 WHERE parent_list_id = ?
		 ORDER BY sort_order ASC, created_at ASC`,
		parentListID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var refs []FavoriteListRef
	for rows.Next() {
		ref, err := scanFavoriteListRef(rows)
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, rows.Err()
}

func (s *userStore) favoriteListCanReach(startListID, targetListID string) (bool, error) {
	startListID = strings.TrimSpace(startListID)
	targetListID = strings.TrimSpace(targetListID)
	if startListID == "" || targetListID == "" {
		return false, nil
	}
	var found int
	err := s.db.QueryRow(
		`WITH RECURSIVE reachable(list_id, depth, path) AS (
			SELECT ? AS list_id, 0 AS depth, ',' || ? || ',' AS path
			UNION ALL
			SELECT refs.child_list_id, reachable.depth + 1, reachable.path || refs.child_list_id || ','
			FROM favorite_list_refs refs
			JOIN reachable ON refs.parent_list_id = reachable.list_id
			WHERE reachable.depth < ?
			  AND instr(reachable.path, ',' || refs.child_list_id || ',') = 0
		)
		SELECT 1 FROM reachable WHERE list_id = ? LIMIT 1`,
		startListID, startListID, favoriteRefMaxDepth, targetListID,
	).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return found == 1, nil
}

func (s *userStore) listFavoriteContents(listID string) (FavoriteListContents, error) {
	listID = strings.TrimSpace(listID)
	if listID == "" {
		return FavoriteListContents{}, nil
	}
	rows, err := s.db.Query(
		`WITH RECURSIVE reachable(list_id, depth, path) AS (
			SELECT ? AS list_id, 0 AS depth, ',' || ? || ',' AS path
			UNION ALL
			SELECT refs.child_list_id, reachable.depth + 1, reachable.path || refs.child_list_id || ','
			FROM favorite_list_refs refs
			JOIN reachable ON refs.parent_list_id = reachable.list_id
			WHERE reachable.depth < ?
			  AND instr(reachable.path, ',' || refs.child_list_id || ',') = 0
		)
		SELECT fm.id, fm.list_id, fm.platform, fm.mod_id, fm.version_id, fm.minecraft_version, fm.mod_loader,
			fm.title, fm.slug, fm.icon_url, fm.description, fm.categories_json, fm.created_at, fm.updated_at,
			reachable.depth, fl.name
		FROM favorite_mods fm
		JOIN reachable ON fm.list_id = reachable.list_id
		JOIN favorite_lists fl ON fl.id = fm.list_id
		ORDER BY reachable.depth ASC, fm.created_at ASC, fm.platform ASC, fm.mod_id ASC`,
		listID, listID, favoriteRefMaxDepth,
	)
	if err != nil {
		return FavoriteListContents{}, err
	}
	defer rows.Close()

	seen := make(map[favoriteModKey]struct{})
	var mods []FavoriteModEntry
	for rows.Next() {
		entry, err := scanFavoriteModEntry(rows)
		if err != nil {
			return FavoriteListContents{}, err
		}
		key := makeFavoriteModKey(entry.ListID, entry.Platform, entry.ModID, entry.MinecraftVersion, entry.ModLoader)
		key.ListID = listID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		mods = append(mods, entry)
	}
	if err := rows.Err(); err != nil {
		return FavoriteListContents{}, err
	}
	refs, err := s.listFavoriteListRefs(listID)
	if err != nil {
		return FavoriteListContents{}, err
	}
	return copyFavoriteListContents(FavoriteListContents{ListID: listID, Mods: mods, Refs: refs}), nil
}

func scanFavoriteListRef(scanner favoriteListRefScanner) (FavoriteListRef, error) {
	var ref FavoriteListRef
	if err := scanner.Scan(&ref.ID, &ref.ParentListID, &ref.ChildListID, &ref.CreatedAt, &ref.UpdatedAt, &ref.SortOrder); err != nil {
		return FavoriteListRef{}, err
	}
	return normalizeFavoriteListRef(ref), nil
}

func scanFavoriteModEntry(scanner favoriteModScanner) (FavoriteModEntry, error) {
	var entry FavoriteModEntry
	var categoriesJSON string
	var depth int
	if err := scanner.Scan(
		&entry.ID,
		&entry.ListID,
		&entry.Platform,
		&entry.ModID,
		&entry.VersionID,
		&entry.MinecraftVersion,
		&entry.ModLoader,
		&entry.Title,
		&entry.Slug,
		&entry.IconURL,
		&entry.Description,
		&categoriesJSON,
		&entry.CreatedAt,
		&entry.UpdatedAt,
		&depth,
		&entry.SourceListName,
	); err != nil {
		return FavoriteModEntry{}, err
	}
	if categoriesJSON != "" {
		if err := json.Unmarshal([]byte(categoriesJSON), &entry.Categories); err != nil {
			return FavoriteModEntry{}, fmt.Errorf("decode favorite categories: %w", err)
		}
	}
	entry.Categories = copyStringSlice(entry.Categories)
	entry.SourceListID = entry.ListID
	entry.Referenced = depth > 0
	return copyFavoriteModEntry(entry), nil
}
