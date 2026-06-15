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
```

Environment variables:

- `KEYS_CF_API_KEY`
- `KEYS_MODRINTH_API_KEY`
- `PREFERS_MINECRAFT_DIR`
- `PREFERS_THEME`

`theme` supports `dark`, `light`, or `system`. The Modrinth key field is currently reserved for future use; Modrinth requests are made with the app user agent, while CurseForge requires `curseforge_api_key` or `KEYS_CF_API_KEY` to enable that source.

The app also lets you choose the `.minecraft` directory from the UI.

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

Run tests:

```bash
go test ./...
```

Build the frontend only:

```bash
npm run build --prefix frontend
```

## Data Files

The app stores local runtime data in the working directory:

- `mod-downloader.toml` for configuration
- `mods.gob.zst` for cached mod platform and jar metadata
