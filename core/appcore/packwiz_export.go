package appcore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/link-fgfgui/mod-downloader-core/logging"
	"github.com/link-fgfgui/mod-downloader-core/modbridge"
	"github.com/link-fgfgui/mod-downloader-core/models"
	"github.com/link-fgfgui/mod-downloader-core/packwiz"
	"github.com/link-fgfgui/mod-downloader-core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader-core/structs"
)

type ExportFavoriteListPackwizResult struct {
	Path string `json:"path"`
}

type favoriteExportScope struct {
	MinecraftVersion string
	ModLoader        string
}

func (s *Service) FavoriteListPackwizDefaultFilename(listID string) string {
	if list, ok := storage.GetFavoriteList(listID); ok {
		return packwiz.SafeArchiveName(list.Name)
	}
	return packwiz.SafeArchiveName("packwiz-export")
}

func (s *Service) ExportFavoriteListPackwizZip(listID, destinationPath string) (ExportFavoriteListPackwizResult, error) {
	return s.ExportFavoriteListPackwizZipForScope(listID, destinationPath, "", "")
}

func (s *Service) ExportFavoriteListPackwizZipForScope(listID, destinationPath, minecraftVersion, modLoader string) (ExportFavoriteListPackwizResult, error) {
	listID = strings.TrimSpace(listID)
	if listID == "" {
		return ExportFavoriteListPackwizResult{}, fmt.Errorf("favorite list is required")
	}
	list, ok := storage.GetFavoriteList(listID)
	if !ok {
		return ExportFavoriteListPackwizResult{}, fmt.Errorf("favorite list not found")
	}
	contents := storage.ListFavoriteContents(listID)
	items := make([]storage.FavoriteMod, 0, len(contents.Mods))
	for _, entry := range contents.Mods {
		items = append(items, entry.FavoriteMod)
	}
	if len(items) == 0 {
		return ExportFavoriteListPackwizResult{}, fmt.Errorf("favorite list is empty")
	}
	items = favoritePackwizItemsForScope(items, minecraftVersion, modLoader)
	if len(items) == 0 {
		return ExportFavoriteListPackwizResult{}, fmt.Errorf("favorite list has no items for selected minecraft version and mod loader")
	}
	scope, err := favoritePackwizScope(items)
	if err != nil {
		return ExportFavoriteListPackwizResult{}, err
	}

	mods := make([]packwiz.Mod, 0, len(items))
	for _, item := range items {
		version, found := resolveFavoritePackwizVersion(item, scope)
		if !found {
			return ExportFavoriteListPackwizResult{}, fmt.Errorf("%s: no matching downloadable version", favoriteDisplayName(item))
		}
		mod, err := packwizModFromFavorite(item, version)
		if err != nil {
			return ExportFavoriteListPackwizResult{}, err
		}
		mods = append(mods, mod)
	}

	destinationPath = ensureZipPath(destinationPath)
	if destinationPath == "" {
		return ExportFavoriteListPackwizResult{}, fmt.Errorf("destination path is required")
	}
	if err := writePackwizZipAtomic(destinationPath, packwiz.Pack{
		Name:             list.Name,
		MinecraftVersion: scope.MinecraftVersion,
		ModLoader:        scope.ModLoader,
		Mods:             mods,
	}); err != nil {
		return ExportFavoriteListPackwizResult{}, err
	}
	logging.Info("favorite list exported as packwiz zip", "listID", listID, "path", destinationPath, "modCount", len(mods))
	s.incrementUsageStat(storage.UsageStatPackwizExports, 1)
	return ExportFavoriteListPackwizResult{Path: destinationPath}, nil
}

func favoritePackwizItemsForScope(items []storage.FavoriteMod, minecraftVersion, modLoader string) []storage.FavoriteMod {
	minecraftVersion = strings.TrimSpace(minecraftVersion)
	modLoader = strings.ToLower(strings.TrimSpace(modLoader))
	if minecraftVersion == "" && modLoader == "" {
		return items
	}
	if minecraftVersion == "" || modLoader == "" {
		return nil
	}
	filtered := make([]storage.FavoriteMod, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.MinecraftVersion) == minecraftVersion && strings.ToLower(strings.TrimSpace(item.ModLoader)) == modLoader {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func favoritePackwizScope(items []storage.FavoriteMod) (favoriteExportScope, error) {
	var scope favoriteExportScope
	for _, item := range items {
		mcVersion := strings.TrimSpace(item.MinecraftVersion)
		modLoader := strings.ToLower(strings.TrimSpace(item.ModLoader))
		name := favoriteDisplayName(item)
		if mcVersion == "" || modLoader == "" {
			return favoriteExportScope{}, fmt.Errorf("%s: missing minecraft version or mod loader", name)
		}
		if scope.MinecraftVersion == "" {
			scope.MinecraftVersion = mcVersion
		} else if scope.MinecraftVersion != mcVersion {
			return favoriteExportScope{}, fmt.Errorf("favorite list contains multiple minecraft versions")
		}
		if scope.ModLoader == "" {
			scope.ModLoader = modLoader
		} else if scope.ModLoader != modLoader {
			return favoriteExportScope{}, fmt.Errorf("favorite list contains multiple mod loaders")
		}
	}
	if scope.MinecraftVersion == "" || scope.ModLoader == "" {
		return favoriteExportScope{}, fmt.Errorf("favorite list is empty")
	}
	return scope, nil
}

func resolveFavoritePackwizVersion(item storage.FavoriteMod, scope favoriteExportScope) (models.ModVersion, bool) {
	platform := strings.ToLower(strings.TrimSpace(item.Platform))
	projectID := strings.TrimSpace(item.ModID)
	ref := models.ProjectKey(platform, projectID)
	if ref == "" {
		return models.ModVersion{}, false
	}
	result := models.ModProject{
		ID:        ref,
		Platform:  platform,
		ProjectID: projectID,
		Slug:      strings.TrimSpace(item.Slug),
		Title:     strings.TrimSpace(item.Title),
	}
	return modbridge.ResolveVersion(appstructs.ModDownloadRequest{
		ProjectID:        ref,
		Result:           result,
		MinecraftVersion: scope.MinecraftVersion,
		ModLoader:        scope.ModLoader,
		VersionID:        strings.TrimSpace(item.VersionID),
	})
}

func packwizModFromFavorite(item storage.FavoriteMod, version models.ModVersion) (packwiz.Mod, error) {
	name := favoriteDisplayName(item)
	version.FileName = strings.TrimSpace(version.FileName)
	version.DownloadURL = strings.TrimSpace(version.DownloadURL)
	version.SHA1 = strings.ToLower(strings.TrimSpace(version.SHA1))
	if version.FileName == "" {
		return packwiz.Mod{}, fmt.Errorf("%s: resolved version is missing filename", name)
	}
	if version.DownloadURL == "" {
		return packwiz.Mod{}, fmt.Errorf("%s: resolved version is missing download URL", name)
	}
	if version.SHA1 == "" {
		return packwiz.Mod{}, fmt.Errorf("%s: resolved version is missing sha1", name)
	}

	platform := strings.ToLower(strings.TrimSpace(version.Platform))
	if platform == "" {
		platform = strings.ToLower(strings.TrimSpace(item.Platform))
	}
	projectID := strings.TrimSpace(version.ProjectID)
	if projectID == "" {
		projectID = strings.TrimSpace(item.ModID)
	}
	versionID := strings.TrimSpace(version.VersionID)
	if versionID == "" {
		versionID = strings.TrimSpace(version.ID)
	}
	return packwiz.Mod{
		Name:        firstNonEmptyString(name, version.Name, version.FileName),
		FileName:    version.FileName,
		DownloadURL: version.DownloadURL,
		SHA1:        version.SHA1,
		Platform:    platform,
		ProjectID:   projectID,
		VersionID:   versionID,
	}, nil
}

func writePackwizZipAtomic(destinationPath string, pack packwiz.Pack) error {
	dir := filepath.Dir(destinationPath)
	base := filepath.Base(destinationPath)
	tmp, err := os.CreateTemp(dir, "."+base+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := packwiz.WriteZip(tmp, pack); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, destinationPath); err != nil {
		if removeErr := os.Remove(destinationPath); removeErr != nil && !os.IsNotExist(removeErr) {
			return err
		}
		if retryErr := os.Rename(tmpPath, destinationPath); retryErr != nil {
			return retryErr
		}
	}
	committed = true
	return nil
}

func ensureZipPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.EqualFold(filepath.Ext(value), ".zip") {
		return value
	}
	return value + ".zip"
}

func favoriteDisplayName(item storage.FavoriteMod) string {
	return firstNonEmptyString(item.Title, item.Slug, item.ModID, item.ID, "favorite mod")
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
