package storage

import (
	"database/sql"
	"errors"
)

func (s *userStore) upsertPinnedMod(p PinnedMod) error {
	p = normalizePinnedMod(p)
	if p.Platform == "" || p.ModID == "" || p.VersionID == "" || p.MinecraftVersion == "" || p.ModLoader == "" {
		return nil
	}
	if existing, ok, err := s.getPinnedMod(p.Platform, p.ModID, p.MinecraftVersion, p.ModLoader); err != nil {
		return err
	} else if ok && existing.ID != "" {
		p.ID = existing.ID
	}
	if p.ID == "" {
		p.ID = NewID()
	}
	_, err := s.db.Exec(
		`INSERT INTO pinned_mods(id, platform, mod_id, version_id, minecraft_version, mod_loader)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(platform, mod_id, minecraft_version, mod_loader)
		 DO UPDATE SET version_id = excluded.version_id`,
		p.ID, p.Platform, p.ModID, p.VersionID, p.MinecraftVersion, p.ModLoader,
	)
	return err
}

func (s *userStore) getPinnedMod(platform, modID, mcVersion, modLoader string) (PinnedMod, bool, error) {
	key := makePinnedModKey(platform, modID, mcVersion, modLoader)
	if key.Platform == "" || key.ModID == "" || key.MinecraftVersion == "" || key.ModLoader == "" {
		return PinnedMod{}, false, nil
	}

	var p PinnedMod
	err := s.db.QueryRow(
		`SELECT id, platform, mod_id, version_id, minecraft_version, mod_loader
		 FROM pinned_mods
		 WHERE platform = ? AND mod_id = ? AND minecraft_version = ? AND mod_loader = ?`,
		key.Platform, key.ModID, key.MinecraftVersion, key.ModLoader,
	).Scan(&p.ID, &p.Platform, &p.ModID, &p.VersionID, &p.MinecraftVersion, &p.ModLoader)
	if errors.Is(err, sql.ErrNoRows) {
		return PinnedMod{}, false, nil
	}
	if err != nil {
		return PinnedMod{}, false, err
	}
	return p, true, nil
}

func (s *userStore) deletePinnedMod(platform, modID, mcVersion, modLoader string) error {
	key := makePinnedModKey(platform, modID, mcVersion, modLoader)
	if key.Platform == "" || key.ModID == "" || key.MinecraftVersion == "" || key.ModLoader == "" {
		return nil
	}
	_, err := s.db.Exec(
		`DELETE FROM pinned_mods
		 WHERE platform = ? AND mod_id = ? AND minecraft_version = ? AND mod_loader = ?`,
		key.Platform, key.ModID, key.MinecraftVersion, key.ModLoader,
	)
	return err
}

func (s *userStore) listPinnedMods() ([]PinnedMod, error) {
	rows, err := s.db.Query(
		`SELECT id, platform, mod_id, version_id, minecraft_version, mod_loader
		 FROM pinned_mods
		 ORDER BY platform ASC, mod_id ASC, minecraft_version ASC, mod_loader ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pins []PinnedMod
	for rows.Next() {
		var p PinnedMod
		if err := rows.Scan(&p.ID, &p.Platform, &p.ModID, &p.VersionID, &p.MinecraftVersion, &p.ModLoader); err != nil {
			return nil, err
		}
		pins = append(pins, p)
	}
	return pins, rows.Err()
}
