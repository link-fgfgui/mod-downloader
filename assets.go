package main

import "embed"

// Embedded frontend assets are shared by the desktop and server entry points.
//
//go:embed all:frontend/dist
var assets embed.FS
