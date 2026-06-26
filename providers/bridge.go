package providers

import (
	appstructs "mod-downloader/structs"
)

// Bridge functions to convert between old and new structures.
// These allow gradual migration from SearchModResult/ProjectVersionResult to ModProject/ModVersion.

// --- Old → New conversions ---

func SearchModResultToModProject(old appstructs.SearchModResult) ModProject {
	platform, projectID := ParseProjectKey(old.ID)
	if platform == "" {
		// Fallback: use Platform field
		platform = old.Platform
	}
	if projectID == "" {
		// Fallback: use Slug
		projectID = old.Slug
	}

	return ModProject{
		ID:          old.ID,
		Platform:    old.Platform,
		ProjectID:   projectID,
		Slug:        old.Slug,
		Title:       old.Title,
		Icon:        old.Icon,
		IconURL:     old.IconURL,
		Description: old.Description,
		Downloads:   old.Downloads,
		UpdatedAt:   0,
	}
}

func ProjectVersionResultToModVersion(old appstructs.ProjectVersionResult) ModVersion {
	deps := make([]ModDependency, 0, len(old.Dependencies))
	for _, oldDep := range old.Dependencies {
		deps = append(deps, ModDependency{
			DependencyProjectID: oldDep.DependencyProjectID,
			DependencyVersionID: oldDep.DependencyVersionID,
			DependencyType:      oldDep.DependencyType,
		})
	}

	return ModVersion{
		ID:           old.ID,
		Platform:     old.Platform,
		ProjectID:    old.ProjectID,
		VersionID:    old.ID, // Use ID as VersionID
		Name:         old.Name,
		Version:      old.Version,
		FileName:     old.FileName,
		DownloadURL:  old.DownloadURL,
		SHA1:         old.SHA1,
		PublishedAt:  old.PublishedAt,
		Downloads:    old.Downloads,
		GameVersions: append([]string(nil), old.GameVersions...),
		Loaders:      append([]string(nil), old.Loaders...),
		Dependencies: deps,
	}
}

// --- New → Old conversions ---

func ModProjectToSearchModResult(new ModProject) appstructs.SearchModResult {
	return appstructs.SearchModResult{
		ID:          new.ID,
		Platform:    new.Platform,
		Title:       new.Title,
		Icon:        new.Icon,
		IconURL:     new.IconURL,
		Description: new.Description,
		Downloads:   new.Downloads,
		Slug:        new.Slug,
	}
}

func ModVersionToProjectVersionResult(new ModVersion) appstructs.ProjectVersionResult {
	deps := make([]appstructs.ProjectDependency, 0, len(new.Dependencies))
	for _, newDep := range new.Dependencies {
		deps = append(deps, appstructs.ProjectDependency{
			DependencyProjectID: newDep.DependencyProjectID,
			DependencyVersionID: newDep.DependencyVersionID,
			DependencyType:      newDep.DependencyType,
		})
	}

	return appstructs.ProjectVersionResult{
		ID:           new.ID,
		Platform:     new.Platform,
		ProjectID:    new.ProjectID,
		Name:         new.Name,
		Version:      new.Version,
		FileName:     new.FileName,
		DownloadURL:  new.DownloadURL,
		SHA1:         new.SHA1,
		PublishedAt:  new.PublishedAt,
		Downloads:    new.Downloads,
		GameVersions: append([]string(nil), new.GameVersions...),
		Loaders:      append([]string(nil), new.Loaders...),
		Dependencies: deps,
	}
}

// --- Batch conversions ---

func ModProjectsToSearchModResults(projects []ModProject) []appstructs.SearchModResult {
	results := make([]appstructs.SearchModResult, 0, len(projects))
	for _, p := range projects {
		results = append(results, ModProjectToSearchModResult(p))
	}
	return results
}

func ModVersionsToProjectVersionResults(versions []ModVersion) []appstructs.ProjectVersionResult {
	results := make([]appstructs.ProjectVersionResult, 0, len(versions))
	for _, v := range versions {
		results = append(results, ModVersionToProjectVersionResult(v))
	}
	return results
}

func SearchModResultsToModProjects(results []appstructs.SearchModResult) []ModProject {
	projects := make([]ModProject, 0, len(results))
	for _, r := range results {
		projects = append(projects, SearchModResultToModProject(r))
	}
	return projects
}

func ProjectVersionResultsToModVersions(results []appstructs.ProjectVersionResult) []ModVersion {
	versions := make([]ModVersion, 0, len(results))
	for _, r := range results {
		versions = append(versions, ProjectVersionResultToModVersion(r))
	}
	return versions
}
