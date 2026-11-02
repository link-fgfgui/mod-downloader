# AGENTS

This file is for AI assistants and human developers working in this repository.
It defines the architecture boundary, the primary layering rule, build and
verification commands, coding red lines, and the documentation mechanism.

---

## 1. Project overview and architecture boundaries

This repository is a **Wails v3 desktop app** for searching, downloading, and
managing Minecraft mods. `core/` is a standalone Go module
(`github.com/link-fgfgui/mod-downloader/core`, wired through
`replace => ./core`). **The same `core` is consumed by the TUI repository
`../mod-downloader-cli` through the symlink `core -> ../mod-downloader-gui-v3/core`**.
The repository therefore has two frontends — GUI (React) and TUI (tview, in the
sibling repository) — sharing one Go domain layer.

```text
React frontend (pages/, stores/, hooks/, lib/)       ← rendering, local UI cache, input
        │  ① generated bindings frontend/bindings/ (App method RPC)
        │  ② Wails event bus (wire name = appcore EventKind.Name())
Root App adapter (main.go / main_server.go, app.go,
                app_contracts.go, app_events.go)     ← wiring, binding pass-through, native dialogs
        ▼
core/host    UI-agnostic runtime: core lifecycle + HTTP extension server + typed event dispatch
        ▼
core/appcore.Service  business facade (process-wide singleton)
        ├── providers    Modrinth / CurseForge integrations
        ├── downloader   download queue, chunking, retries, stall handling
        ├── modbridge    online metadata ↔ local mods
        ├── minecraft    instance/launcher discovery, jar parsing, JIJ analysis
        ├── storage      SQLite (favorites, pins, stats) + gob+zstd metadata cache
        ├── configs      TOML config, boundary constants, normalization
        ├── structs      cross-frontend projection types (single data contract)
        └── models       canonical domain models (single source of truth)
```

- Sibling layout: `../mod-downloader-cli` must exist next to this repository,
  otherwise its `core` symlink breaks.
- Domain structs are defined only in `core/models` / `core/structs`; adapters and
  the frontend must not duplicate them.
- Core startup must fail fast: `GetStartupStatus` returns a typed failure reason,
  and the frontend bootstrap checks it before mounting stores. Never hide a
  startup failure behind an empty result.

---

## 2. Primary rule: no UI-framework-agnostic logic outside core

Because the GUI and TUI share one `core`, any logic that still holds after
swapping the UI framework (React ↔ tview) must have **exactly one Go
implementation, inside core**. This is the most important layering rule in the
repository.

### 2.1 Classification test

Before changing any logic, answer:

1. **Would this rule still hold under a different UI framework?**
2. **Does it write core state, or must both UIs behave identically?**

If either answer is yes, put it in core and expose it through an
`appcore.Service` method or a `core/structs` projection. `app.go` only passes it
through; the frontend only consumes it. If both answers are no, it may stay in
the frontend (or the root adapter).

| Belongs in core (GUI/TUI must agree) | May stay in the frontend |
| :--- | :--- |
| Search expression parsing, sorting, grouping, compatibility | Pagination, focus, drawers, selection, form drafts |
| Settings boundary constants and clamp normalization | Animation timing, GSAP/FLIP, theme styling |
| Favorite ordering/scope filtering, pinned partitions | Tooltip copy, i18n, pinyin-initial search enhancement |
| Download queue transitions, chunking/retry rules | Imperative layout, per-frame formatting of core-computed values |
| Event kinds, wire names, lifecycle, HTTP server | Rendering projections directly |

### 2.2 Why this is strict (historical lesson)

After the GUI + TUI split, duplicated implementations were measured and
amplified: Manage grouping/JIJ/compatibility/search text had three copies,
settings boundary constants existed three times, favorite ordering twice (both
missing the `created_at` tie-breaker), and three event enums kept hand-written
wire names in sync. They were consolidated into `core/host`,
`appcore.EventKind.Name()`, and `core/structs`. **Do not reintroduce them.**
Before writing logic in `app.go`, `stores/*.ts`, or `lib/*.ts`, apply the 2.1
test.

### 2.3 Dependency direction (mechanically checkable)

1. `core/` must not import adapter packages. Adapters (the root main package,
   TUI `internal/...`) may import `core/*`.
2. Event kinds and wire names are defined once in `core/appcore`
   (`EventKind.Name()`, kebab-case, locked by a contract test). Adapters must not
   define enums or a second set of names.
3. Subscribe to core events only through `core/host`'s typed `host.On` /
   `host.OnEvent` handlers. The single `any` assertion is allowed only inside
   `core/host`.
4. `app.go` does not own lifecycle: startup/shutdown/HTTP/events belong to
   `core/host`. The repository root only holds Wails wiring, binding
   pass-through, and native dialogs. `app_contracts.go` holds adapter-only types
   (for example native dialog results).
5. The standard path for a new GUI capability is **core API/projection type →
   `app.go` binding → frontend consumption**. If step 1 is impossible (the TUI
   needs the same rule), the logic never belonged in the frontend.

---

## 3. Build, test, and bindings

### Go

```bash
go build . && go build -tags server .   # both forms (window / headless) must compile
go vet ./...
go test ./...                           # root contract and version tests
(cd core && go test ./...)              # core is a separate module; run it separately
```

- `app_test.go` / `app_contract_test.go` lock the App binding surface
  (`expectedAppMethods`): when adding or removing an `App` method, update the
  contract test and make it pass.
- Core is shared: **any public core API or event change must be verified in the
  TUI repository** (see the 6.4 pre-commit checklist).

### Frontend

```bash
cd frontend
npm run build      # tsc type check + Vite production build
npx vitest run     # unit tests
```

- After Go `App` signatures or core structs change, regenerate
  `frontend/bindings/` with `wails3 generate bindings` (equivalent to
  `build/Taskfile.yml`'s `generate:bindings`, including `-clean=true -ts`). The
  generator analyzes the same code under the `server,production` tags.
- The frontend consumes generated binding types only; never hand-write parallel
  interfaces.

---

## 4. UI verification

All UI checks use the Playwright MCP browser tools. Do not use `spectacle`,
`xwininfo`, `uinput`, or other desktop screenshot/input automation: drive the
headless Wails server in a browser instead. It is faster, scriptable, and gives
DOM-level assertions.

### Headless server workflow

```bash
wails3 task build:server
./bin/mod-downloader-server
```

The server listens on `http://127.0.0.1:9245` by default; override with
`MOD_DOWNLOADER_SERVER_PORT`. It is subject to the global single-instance guard
(`127.0.0.1:18801`); close any running GUI/TUI instance first or it exits with
`instance.ErrAlreadyRunning` (see `docs/specs/core/single-instance.md`). It serves
the embedded production frontend with real Go bindings, so Playwright exercises
the real backend.

For source-level React diagnostics, start the server first and run Vite against
it; open `http://127.0.0.1:9246/`:

```bash
cd frontend && WAILS_SERVER_URL=http://127.0.0.1:9245 npm run dev -- --port 9246
```

Running `npm run dev` alone provides no Wails bindings; end-to-end UI
verification must go through the headless server.

---

## 5. Coding standards and anti-patterns

These red lines are the repository's coding standard (originally derived from the
old repository, now inlined here).

### Go

1. **No type-alias transition layers**: do not write
   `type SearchModResult = models.ModProject`; import and use
   `core/models.*` / `core/structs.*` directly.
2. **No re-export files**: no package may forward core-exported types.
3. **Replace means delete**: when replacing conversion/rule code, delete the old
   implementation in the same commit; leave no dead path.
4. **Derive test boundaries from production constants**: write `MaxLimit + 1`,
   not a magic number that rots when the constant changes.
5. **Return errors explicitly** and translate them into user-readable copy. Never
   swallow errors or store them unused; log through `core/logging`.
6. Keep touched files clean under `gofmt` / `go vet`.
7. Keep the SPDX header on every source file: `GPL-3.0-or-later` outside core;
   `LGPL-3.0-or-later WITH LGPL-3.0-linking-exception` inside core. The weaker
   core license plus linking exception is the licensing expression of the
   replaceable-adapter (GUI/TUI) architecture.

### Frontend (React + TypeScript)

1. Transient UI state (dialogs, drawers) must be fully reset on close and cancel
   paths; never rely on a page reload to clear state.
2. Cross-frontend rules belong in core (see section 2). Minimize `lib/` helpers;
   existing display projections such as `modGroupDisplay.ts` must not reimplement
   core sorting/grouping/compatibility rules.
3. When revising i18n, update `locales/en.ts` and `locales/zh.ts` in the same
   change; do not leave missing keys.

---

## 6. Documentation mechanism

Documentation uses a lightweight, tool-free Markdown mechanism (no scripts, state
machines, or JSONL manifests) under `docs/specs/`. **The only source of truth for
specifications is this repository's `docs/specs/`**; the old repository is
archived (`../archived-legacy-mod-downloader.tar.gz`) and is historical reference
only. Reuse an existing file instead of creating a new one.

All project documentation is written in English.

### 6.1 Directory structure

```text
docs/specs/
  core/       core domain rules (storage, downloads, networking, compatibility, …)
  adapter/    Wails adapter (binding surface, events, native dialogs, server mode)
  frontend/   React UI (state lifecycle, animation, i18n, theme)
```

- Each subdirectory has an `index.md` listing every topic file. Create it when
  the first file lands.
- `core` exists physically only in this repository: **core-level specifications
  live in this repository's `docs/specs/core/`**. The TUI repository must link
  here rather than copy them.

### 6.2 Read before changing

Before modifying an existing layer or module, find the relevant specification in
`docs/specs/` and confirm the plan does not violate a recorded contract or repeat
a recorded anti-pattern. If the topic is not documented, add it using the
triggers in 6.3 before changing code.

### 6.3 When to record

Record immediately when any of the following occurs:

1. **Subtle or recurring bugs**: non-obvious traps (silently dropped fields,
   goroutine leaks, races, lost events, third-party API quirks). Record the
   problem, root cause, forbidden pattern (Don't), and correct approach (Instead).
2. **Architecture or contract boundary changes**: new modules, cross-layer
   payload changes, SQLite schema changes, `App` binding or core event changes.
   Update the relevant spec or create `docs/specs/<layer>/<topic>.md` and register
   it in `index.md`.
3. **Identified anti-patterns**: circular dependencies, duplicate
   implementations (especially those violating section 2), or three sources of
   duplicated state. Append to the relevant layer spec.
4. **Key design decisions**: non-trivial choices (complex dependency resolution,
   download chunk calculation, grouping/compatibility consolidation). Record the
   constraints and trade-offs.

### 6.4 One-minute pre-commit checklist

Before finishing a task or committing:

- [ ] `go build . && go build -tags server .`, `go vet ./...`, `go test ./...` pass.
- [ ] `(cd core && go test ./...)` passes.
- [ ] Frontend `npm run build` and `npx vitest run` pass.
- [ ] If `App` methods or core structs changed: bindings regenerated and
      `app_contract_test.go` updated.
- [ ] **When core public APIs or events change, verify the sibling repository**:
      `cd ../mod-downloader-cli && go build ./... && go vet ./... && go test ./...`
- [ ] Did this work fix a subtle bug or produce an architectural insight? If yes,
      update/create the relevant spec in `docs/specs/`, and commit the doc change
      in the same batch as the code change.
- [ ] Grep for revived "second implementations", for example:
      `rg -n "MinFileConcurrency|MinConcurrentDownloads|MaxTargetDownloadRateMiB" --glob '!core/**'`
      must return 0 hits in adapters and the frontend (boundary rules live in one
      place only).
