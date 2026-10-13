# Repository Guidelines & Agent Instructions

This document provides guidelines, architectural context, development commands, and historical knowledge maintenance requirements for AI assistants and human developers working on this project.

---

## 1. Project Overview & Architecture

This repository is a **Wails v2 desktop application** for searching, downloading, and managing Minecraft mods and mod dependencies.

### Key Architecture Boundaries

```text
Vue 3 Frontend (views, stores, components)
        │
        ▼ (Generated TypeScript bindings: frontend/wailsjs/)
Root App Adapter (main.go, app.go, app_contracts.go)
        │ - Handles Wails runtime events, native file dialogs, window state
        ▼ (Adapter-neutral Go interface)
core/appcore.Service (Business logic engine, NO Wails imports)
        ├── providers   Modrinth & CurseForge API integrations
        ├── downloader  Queue management, dynamic chunking, retry & stall handling
        ├── modbridge   Bridge between online metadata and local mod files
        ├── minecraft   Game directory/launcher detection, jar parsing, JIJ dependency analysis
        ├── storage     SQLite user data (favorites, tags) & gob+zstd metadata cache
        ├── configs     TOML configuration and environment fallbacks
        └── global      In-memory indexes and shared clients
```

- **`core/` as a submodule**: Domain logic lives in `core/` (`github.com/link-fgfgui/mod-downloader-core`), referenced in `go.mod` via `replace github.com/link-fgfgui/mod-downloader-core => ./core`.
- **Single Source of Truth for Data Models**: All canonical domain structs live in `core/models` (`models.ModProject`, `models.ModVersion`).
- **Detailed Map**: See [ARCHITECTURE.md](ARCHITECTURE.md) for full architecture and event flow details.

---

## 2. Build, Test, and Development Commands

### Backend (Go & Wails)
- `wails dev`: Run the desktop app in live-reloading development mode.
- `wails build`: Create a production release build.
- `wails generate module`: Regenerate frontend TypeScript bindings in `frontend/wailsjs/` whenever Go `App` methods or models change.
- `go test ./...`: Run all root backend unit tests.
- `(cd core && go test ./...)`: Run tests inside the `core/` submodule.

### Frontend (Vue 3, TypeScript, Vite, Vuetify)
- `npm ci --prefix frontend`: Install frontend dependencies from `package-lock.json`.
- `npm run build --prefix frontend`: Run `vue-tsc` type-checking and Vite production build.
- `npm run preview --prefix frontend`: Preview the built frontend standalone.

---

## 3. Core Coding Standards & Anti-Patterns

### Go Backend
1. **Never use type aliases as a transition layer**: Do not define `type SearchModResult = models.ModProject`. Import and use `models.*` directly.
2. **Never leave re-export files**: Packages must not re-export types from `core/models`.
3. **No parallel converter functions**: When replacing converter logic (e.g., `<sdkType>To<canonicalType>`), delete the old function in the same commit to avoid dead/buggy paths.
4. **Derive boundary test values from production constants**: Boundary checks in unit tests must use `MaxLimit + 1`, never hardcoded magic numbers that might become valid when limits change.
5. **Format Go code**: Ensure `gofmt -l .` is clean for modified files.

### Frontend
1. **Component conventions**: Vue Single-File Components with PascalCase names in `frontend/src/components/`.
2. **UI state cleanup**: Modals and dialogs must properly reset their local state upon closing/cancel.
3. **Number inputs**: When dealing with decimal inputs in Vuetify text fields, handle locale decimal formatting safely.

---

## 4. Historical Knowledge & Specifications Library (`docs/specs/`)

All accumulated technical specifications, domain rules, and hard-won bug prevention lessons are preserved under `docs/specs/`. Consult these before making changes in relevant areas:

### Backend Specifications (`docs/specs/backend/`)
- [Directory Structure](docs/specs/backend/directory-structure.md): Submodule layout, single-source-of-truth convention, adapter bridge.
- [Quality Guidelines](docs/specs/backend/quality-guidelines.md): Forbidden patterns, converter naming rules, code review checklist.
- [Storage Guidelines](docs/specs/backend/storage-guidelines.md): Metadata cache, SQLite persistence, and migration boundaries.
- [Error Handling](docs/specs/backend/error-handling.md): Go error propagation, UI-friendly translation, and logging rules.
- [Logging Guidelines](docs/specs/backend/logging-guidelines.md): Structured logging, file policies, log levels, and sensitive data redaction.

### Core Domain Specifications (`docs/specs/core/backend/`)
- [File Transfer & Download](docs/specs/core/backend/file-transfer.md): Queue concurrency, chunk sizing, cancellation, retry/stall logic.
- [Connector Compatibility](docs/specs/core/backend/connector-compatibility.md): Cross-loader compatibility rules (Fabric, Forge, NeoForge, Quilt).
- [Default CurseForge API Key](docs/specs/core/backend/default-curseforge-api-key.md): Effective API key fallback rules and build injection.
- [Local Mod Online Version](docs/specs/core/backend/local-mod-online-version.md): SHA1 hashing and online version matching.
- [Network Runtime](docs/specs/core/backend/network-runtime.md) & [System Proxy](docs/specs/core/backend/system-proxy.md): Proxy handling and connection timeouts.

### Frontend Specifications (`docs/specs/app/frontend/`)
- [UI State Lifecycle](docs/specs/app/frontend/ui-state-lifecycle.md): State cleanup, debounce, and modal lifecycles.
- [Download Queue Actions](docs/specs/app/frontend/download-queue-actions.md): Queue states, progress aggregation, and cancellation handling.
- [Animation Modes](docs/specs/app/frontend/animation-modes.md): CSS animations, hardware acceleration, and reduced-motion modes.
- [Language Preference](docs/specs/app/frontend/language-preference.md): Vue i18n and system locale synchronization.

### Thinking Guides (`docs/specs/guides/`)
- [Code Reuse Thinking Guide](docs/specs/guides/code-reuse-thinking-guide.md): When to extract helpers vs when to keep code localized.
- [Cross-Layer Thinking Guide](docs/specs/guides/cross-layer-thinking-guide.md): Checklists for data crossing Frontend ↔ Wails ↔ Core boundaries.

### Historical Records
- `docs/tasks/`: Historical task PRDs and design notes.
- `docs/journals/`: Past developer journals and session histories.

---

## 5. Lightweight Historical Experience Accumulation Requirements

To ensure the codebase continues to learn from mistakes and preserve architectural wisdom without the friction of complex framework tooling, all AI agents and developers must adhere to the following **lightweight experience accumulation requirements**:

### 5.1 Pre-Development Retrieval (开发前查阅)
- **Before modifying any existing layer or module**, locate and read the corresponding guide in `docs/specs/`.
- Verify that your proposed approach does not violate documented contracts or repeat previously documented anti-patterns.

### 5.2 When to Accumulate Experience (沉淀触发场景)
Record new experience immediately whenever work involves:
1. **Subtle or Recurring Bugs (隐蔽/反复出现的 Bug)**:
   - A bug caused by a non-obvious trap (e.g., silent field omissions, goroutine leaks, race conditions, Wails event loss, third-party API quirks).
   - *Requirement*: Document the problem, the root cause, the forbidden pattern (Don't), and the correct approach (Instead).
2. **Architecture & Contract Boundary Changes (架构与契约边界变更)**:
   - Introducing a new module, changing cross-layer payload formats, altering SQLite schema, or adding/modifying Wails adapter events.
   - *Requirement*: Update the relevant spec file or create a concise new topic file under `docs/specs/<layer>/`.
3. **Discovered Anti-Patterns (识别出的反模式)**:
   - Anti-patterns identified during refactoring (e.g., circular imports, redundant converters, state duplication).
   - *Requirement*: Append the rule to `docs/specs/backend/quality-guidelines.md` or the corresponding layer spec.
4. **Significant Feature Solutions (关键特性方案备忘)**:
   - Non-trivial feature decisions (e.g., complex dependency resolution, download chunk calculation).
   - *Requirement*: Document key constraints and design rationale in `docs/specs/` (or create a task summary under `docs/tasks/`).

### 5.3 Where & How to Record (沉淀归档规范)
- **Keep it lightweight and tool-free**: Experience is recorded in plain Markdown files in git. No custom Python scripts, no state machines, no JSONL manifests required.
- **Prefer Updating Existing Specs**:
  - Always prefer updating the smallest existing relevant file (e.g., adding a section to `quality-guidelines.md`, `storage-guidelines.md`, or `ui-state-lifecycle.md`) rather than creating fragmented small files.
  - If a new topic is needed, create `docs/specs/<layer>/<topic>.md` and link it in `docs/specs/<layer>/index.md`.
- **Use Actionable "Problem / Why / Instead" Format**:
  ```markdown
  ### Don't: [Anti-pattern title]
  **Problem**: Code snippet or description of the bad pattern.
  **Why it's bad**: Concrete consequences (bugs caused, memory leaks, silent failures).
  **Instead**: Preferred code snippet and convention to follow.
  ```

### 5.4 Pre-Commit 1-Minute Quality & Experience Gate (提交前闭环自检)
Before completing any task or creating a commit, run this quick checklist:
- [ ] **Tests pass**: `go test ./...` and `(cd core && go test ./...)` pass.
- [ ] **Frontend builds**: `npm run build --prefix frontend` succeeds without type errors.
- [ ] **Wails bindings updated**: If Go `App` signatures changed, `wails generate module` was executed.
- [ ] **Experience captured**:
  - *Did this task fix a non-obvious bug or surface valuable architectural insight?*
  - If **yes**, update the relevant spec under `docs/specs/` and include the documentation change in the **same commit batch** as the code changes.
