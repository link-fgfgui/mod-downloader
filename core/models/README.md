# Canonical Models

`models.go` is the single source of truth for provider-neutral project,
version, dependency, and composite-key types. Providers, storage, appcore, and
the Wails adapter import these types directly. Do not create provider aliases
or parallel model files for convenience.
