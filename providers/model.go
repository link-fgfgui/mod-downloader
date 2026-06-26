package providers

import "mod-downloader/models"

// Re-export types from models package for backward compatibility
type ModProject = models.ModProject
type ModVersion = models.ModVersion
type ModDependency = models.ModDependency

// Re-export helper functions
var ProjectKey = models.ProjectKey
var ParseProjectKey = models.ParseProjectKey
var VersionKey = models.VersionKey
var ParseVersionKey = models.ParseVersionKey

