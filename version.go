package main

import "strings"

const defaultAppVersion = "dev"

var appVersion = defaultAppVersion

func currentAppVersion() string {
	version := strings.TrimSpace(appVersion)
	if version == "" {
		return defaultAppVersion
	}
	return version
}
