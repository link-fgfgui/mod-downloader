# mod-downloader

A desktop Minecraft mod downloader and local mod manager built with Go, Wails, Vue, Pinia, and Vuetify.

`mod-downloader` helps you search Modrinth and CurseForge, filter releases by Minecraft version and mod loader, install matching jars into a selected Minecraft instance, and inspect the mods already present in that instance. It focuses on downloading and managing mod files; it is not a launcher.

## Features

- Search Minecraft mods across Modrinth and CurseForge.
- Filter results by Minecraft version and loader.
- Install matching mod files into the selected instance's `mods` directory.
- Queue downloads and show per-result install/update/conflict states.
- Resolve required platform dependencies where provider metadata exposes them.
- Pin a specific project version for future installs.
- Scan local `mods` folders and display installed mod IDs, versions, files, and enabled state.
- Parse mod metadata directly from jars, including Fabric, Forge, NeoForge, jar-in-jar metadata, nested jars, and manifest-backed `${file.jarVersion}` values.
- Cache platform and jar metadata locally with gob + zstd.

## Supported Sources And Loaders

Sources:

- Modrinth
- CurseForge, when a CurseForge API key is configured

Loaders:

- Fabric
- Forge
- NeoForge

## Configuration

Configuration is read from `mod-downloader.toml` in the working directory and can also be provided through environment variables.

```toml
[keys]
curseforge_api_key = "your-curseforge-api-key"
modrinth_api_key = ""

[preferences]
theme = "dark"
minecraft_dir = "/path/to/.minecraft"
mcim_enabled = false

[downloads]
file_concurrency = 4
concurrent_downloads = 1

[api]
requests_per_second = 0
```

Environment variables:

- `KEYS_CF_API_KEY`
- `KEYS_MODRINTH_API_KEY`
- `PREFERS_MINECRAFT_DIR`
- `PREFERS_THEME`
- `PREFERS_MCIM_ENABLED`
- `DOWNLOADS_FILE_CONCURRENCY`
- `DOWNLOADS_CONCURRENT_DOWNLOADS`
- `API_REQUESTS_PER_SECOND`
- `HTTP_PROXY` / `HTTPS_PROXY`
- `NO_PROXY`

`theme` supports `dark`, `light`, or `system`. The Modrinth key field is currently reserved for future use; Modrinth requests are made with the app user agent, while CurseForge requires `curseforge_api_key` or `KEYS_CF_API_KEY` to enable that source.

`file_concurrency` controls the number of ranged chunks used for one file, and `concurrent_downloads` controls how many files may download at once. `requests_per_second` limits the combined CurseForge and Modrinth API request rate; `0` disables rate limiting.

All outbound API, metadata, and file requests automatically use the process system proxy environment. `NO_PROXY` exclusions are honored; lowercase proxy variable names are also supported by Go.

The app also lets you choose the `.minecraft` directory and switch the Modrinth/CurseForge API and file sources to MCIM from the UI.

## Development

Requirements:

- Go 1.24+
- Node.js and npm
- Wails v2

Install frontend dependencies:

```bash
npm ci --prefix frontend
```

Run in development mode from the repository root:

```bash
wails dev
```

Build a production binary:

```bash
wails build
```

Inject a version into the production binary:

```bash
export APP_VERSION=v1.2.3
wails build -ldflags "-X main.appVersion=${APP_VERSION}"
```

Build the CLI binary:

```bash
go build ./cmd/mod-downloader-cli
```

Run the CLI during development:

```bash
go run ./cmd/mod-downloader-cli --help
```

Run tests:

```bash
go test ./...
```

Build the frontend only:

```bash
npm run build --prefix frontend
```

## Data Files

The app stores configuration and local runtime data separately:

- `mod-downloader.toml` in the working directory for configuration
- `mod-metadata.tmp` in the configured cache directory for rebuildable mod platform metadata cache
- `mod-favs.sqlite` in the configured cache directory for user-owned data such as pinned mod versions and favorite lists
- `mod-downloader.log` in the working directory when stderr is unavailable or file logging is forced

Logging defaults to stderr. Configure automatic output behavior in
`mod-downloader.toml`:

```toml
[logging]
disabled = false
force_file = false
```

`disabled = true` suppresses all application logs and takes precedence over
`force_file`. `force_file = true` appends logs to `mod-downloader.log` while
continuing to use stderr when it is available. Without either option, the app
falls back to the log file only when stderr cannot be used. Environment
equivalents are `LOGGING_DISABLED` and `LOGGING_FORCE_FILE`.

## CLI

The CLI reuses the same backend logic as the desktop app. It reads
`mod-downloader.toml` and the same environment variables as the UI. Global
flags such as `--minecraft-dir`, `--curseforge-api-key`, and
`--modrinth-api-key` override config for the current command.

Commands:

```bash
go run ./cmd/mod-downloader-cli config --json
go run ./cmd/mod-downloader-cli --minecraft-dir /path/to/.minecraft versions
go run ./cmd/mod-downloader-cli search sodium --version 1.21.1 --loader fabric
go run ./cmd/mod-downloader-cli install --instance fabric-loader-1.21.1 --project modrinth:sodium
go run ./cmd/mod-downloader-cli mods --instance fabric-loader-1.21.1 --json
```
