#!/usr/bin/env bash
# Production build wrapper: injects app version and default CurseForge API key via ldflags.
# CI / gitignore for this script are intentionally left to the repository owner.
set -euo pipefail

# Prefer an explicit env override; fall back to the project's known default key.
# Single-quoted so '$' characters in the key are not expanded by the shell.
if [[ -z "${DEFAULT_CF_API_KEY:-}" ]]; then
  DEFAULT_CF_API_KEY='$2a$10$wuAJuNZuted3NORVmpgUC.m8sI.pv1tOPKZyBgLFGjxFp/br0lZCC'
fi

APP_VERSION="${APP_VERSION:-}"

CF_KEY_PATH='github.com/link-fgfgui/mod-downloader-core/configs.DefaultCurseforgeAPIKey'
LDFLAGS="-X main.appVersion=${APP_VERSION} -X ${CF_KEY_PATH}=${DEFAULT_CF_API_KEY}"

exec wails build -ldflags "${LDFLAGS}" "$@"
