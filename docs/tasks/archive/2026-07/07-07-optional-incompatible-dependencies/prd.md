# Optional and incompatible dependency handling

## Goal

Platform dependency metadata already distinguishes required, optional, and incompatible dependency relations. Extend the download workflow so users see optional dependency recommendations without forcing them into the download, and so incompatible installed mods are surfaced with the same explicit-confirmation style as existing update/conflict replacement.

The user request, in Chinese:

> 分析可选依赖和不兼容依赖。下载的模组有可选依赖时在下载列表单独tab里显示主mod和这些依赖mod，可以一键安装（仍然走完整兼容性判定），也可以从主mod处删除提醒。不兼容依赖遵照现有modid冲突模式类似显示，类似二次确认机制，类似rename处理。下载队列无mod但有可选mod提醒时仍然显示展开队列按钮，但是此时可以右键按钮清空所有消息，并且此时icon设为铃铛。

## Confirmed Facts

- `models.ModVersion.Dependencies []ModDependency` already carries dependency metadata, including `DependencyType` values such as `required`, `optional`, and `incompatible` ([core/models/models.go](/home/link/Documents/PROJ/mod-downloader-dev/opandimp/core/models/models.go:35)).
- CurseForge dependency relation types are normalized to `optional`, `required`, `incompatible`, etc.; Modrinth dependency types are lowercased and stored as provided by the API ([core/providers/modprovider.go](/home/link/Documents/PROJ/mod-downloader-dev/opandimp/core/providers/modprovider.go:543)).
- Required dependencies are currently handled automatically by `queueMissingRequiredDependencies`, and non-required dependencies are ignored during queueing ([core/downloader/download.go](/home/link/Documents/PROJ/mod-downloader-dev/opandimp/core/downloader/download.go:490)).
- Install-time compatibility decisions use `modbridge.InstallStatusPrecise`, which may parse remote jars and returns existing button states such as new/installed/update/conflict ([core/modbridge/modbridge.go](/home/link/Documents/PROJ/mod-downloader-dev/opandimp/core/modbridge/modbridge.go:208)).
- Existing conflict/update confirmation is a frontend dialog driven by search-result button status. Left-click confirms, right-click skips confirmation, and batch download currently skips confirmation ([frontend/src/stores/downloadSearch.ts](/home/link/Documents/PROJ/mod-downloader-dev/opandimp/frontend/src/stores/downloadSearch.ts:66), [frontend/src/components/SearchResultList.vue](/home/link/Documents/PROJ/mod-downloader-dev/opandimp/frontend/src/components/SearchResultList.vue:45)).
- Existing replacement behavior archives superseded local jars by renaming them with `.old`, `.old.2`, etc. ([core/downloader/download.go](/home/link/Documents/PROJ/mod-downloader-dev/opandimp/core/downloader/download.go:456)).
- The floating download button is shown when `downloadQueueStore.queue.active` is true, opens the queue panel, and currently uses the download icon unless the panel is open ([frontend/src/App.vue](/home/link/Documents/PROJ/mod-downloader-dev/opandimp/frontend/src/App.vue:23)).
- `DownloadQueueState.Active` is true if there is a running job, pending job, or retryable history item; queue items currently model download jobs only ([core/downloader/download.go](/home/link/Documents/PROJ/mod-downloader-dev/opandimp/core/downloader/download.go:748), [core/structs/search.go](/home/link/Documents/PROJ/mod-downloader-dev/opandimp/core/structs/search.go:47)).
- The archived dependency-analysis task established that download dependency logic should consume platform-side `ModVersion.Dependencies` only, not local JAR dependency declarations.

## Confirmed Decisions

- D1: Batch download must not silently skip incompatible confirmations. When the batch download button is pressed, the app must analyze all selected items for incompatible conflicts before queueing.
- D2: Batch incompatible conflicts must use a single confirmation dialog that shows the complete conflict content for all affected selected mods.
- D3: In the batch incompatible dialog, confirming continues with the full install logic for all selected mods, including incompatible archive/rename behavior.
- D4: In the batch incompatible dialog, declining skips only the conflicted mods and continues downloading the non-conflicted selected mods.
- D5: Pressing `Esc` or otherwise canceling/dismissing the batch incompatible dialog cancels this batch operation entirely; no selected mods from that batch are queued.

## Requirements

### Optional dependency reminders

- R1: When a queued mod version has `optional` dependencies, the app must create a non-download reminder group tied to the main mod.
- R2: The expanded floating queue panel must include a separate tab for optional dependency reminders. That tab must show the main mod and its optional dependency mods.
- R3: A reminder group must allow one-click installation of its optional dependency mods. Each dependency install must go through the same backend queue path and full compatibility analysis used by normal installs, including update/conflict/incompatible handling.
- R4: Users must be able to dismiss a reminder group from the main mod entry without canceling existing downloads or changing installed mods.
- R5: Optional dependency reminders must survive normal queue activity during the current app session, but do not need to persist across app restarts.
- R6: If there are no running/pending/retryable download items but optional reminder messages remain, the floating queue button must still be visible.
- R7: In the reminder-only state, the floating button icon must be `mdi-bell-outline` or an equivalent bell icon, not the normal download icon.
- R8: In the reminder-only state, right-clicking the floating queue button must clear all optional reminder messages.

### Incompatible dependency handling

- R9: The app must analyze `incompatible` dependency relations for the selected version before queueing/installing the requested mod.
- R10: If an incompatible dependency target is detected as installed in the selected instance, the search result must present a warning state similar to existing mod-ID `conflict` state.
- R11: Left-clicking an incompatible warning install must require a second confirmation before queueing. Right-click single install may keep the existing skip-confirmation convention.
- R12: The confirmation copy must tell the user that incompatible installed jars will be renamed with the `.old` suffix before the requested mod is installed.
- R13: Once confirmed, the backend install path must archive the incompatible local jars through the same `.old` naming policy as replacement/conflict handling before installing the requested mod.
- R14: If incompatible dependency metadata cannot be resolved to local mod IDs, the app must fail soft: log/ignore unresolved incompatible checks rather than blocking downloads.
- R15: Batch download must run a batch incompatible analysis before queueing selected mods.
- R16: If batch analysis finds incompatible conflicts, the app must show one dialog containing the full conflict list grouped by selected mod and incompatible installed mod.
- R17: If the user confirms the batch incompatible dialog, the app must queue all selected mods and each conflicted mod must execute the full incompatible archive/rename path.
- R18: If the user declines the batch incompatible dialog, the app must queue only selected mods without incompatible conflicts and skip conflicted selected mods.
- R19: If the user cancels the batch incompatible dialog with `Esc` or dialog dismissal, the app must cancel the entire batch operation without queueing any selected mods from that click.

### Data and API boundaries

- R20: All dependency analysis must use platform-side `ModVersion.Dependencies`; JAR-embedded dependency parsing remains out of scope.
- R21: Backend queue state must expose enough structured data for the frontend to render download items and optional reminder groups without deriving dependency relationships from display strings.
- R22: Dependency project display metadata should be resolved from existing provider/cache APIs when available and fall back to platform/projectID when metadata is unavailable.
- R23: User-facing labels and confirmation copy must be localized in existing zh/en i18n files.

## Acceptance Criteria

- [ ] Downloading a mod with optional dependencies creates an optional-reminder tab entry showing the main mod and its optional dependency mods.
- [ ] The optional-reminder tab can install all missing optional dependency mods with one action, and those installs use the normal queue path.
- [ ] Dismissing an optional reminder from its main mod removes that reminder while preserving active/retryable download items.
- [ ] When only optional reminders remain, the floating queue button remains visible, uses a bell icon, opens the queue panel, and right-click clears all optional reminders.
- [ ] A mod with installed incompatible dependencies shows a warning/confirm state analogous to existing conflict handling.
- [ ] Confirmed incompatible installs archive matching incompatible local jars using the existing `.old` suffix policy before installing the requested mod.
- [ ] Right-click single install preserves the current skip-confirmation convention for warning states.
- [ ] Pressing batch download runs one batch incompatible analysis before any selected mod is queued.
- [ ] A batch with incompatible conflicts shows one dialog with the complete conflict list.
- [ ] Confirming the batch incompatible dialog queues all selected mods and preserves full archive/rename behavior for conflicted mods.
- [ ] Declining the batch incompatible dialog skips conflicted selected mods and queues non-conflicted selected mods.
- [ ] Canceling the batch incompatible dialog with `Esc` or dismissal queues none of the selected mods from that batch click.
- [ ] Required dependency auto-queue behavior remains unchanged.
- [ ] Existing update/conflict button behavior and retryable download queue behavior remain unchanged.
- [ ] Go tests cover optional reminder queue state, clearing/dismissing reminders, incompatible installed detection, incompatible archive handling, and batch incompatible analysis results.
- [ ] Frontend tests or focused manual verification cover batch confirm, decline, and Esc cancel paths.
- [ ] Frontend build/type checks pass with regenerated or synchronized Wails bindings.

## Notes

- This is a complex cross-layer task touching core downloader state, modbridge compatibility analysis, appcore/Wails bindings, Pinia stores, App queue UI, Download confirmation UI, generated Wails types, and i18n.

## Out of Scope

- Parsing dependency declarations embedded inside local JAR metadata.
- Persisting optional reminders across app restarts.
- Parallel downloads or byte-level download progress.
- Changing how required dependencies are auto-installed.
- Changing the existing right-click skip-confirmation convention unless the user explicitly requests a stricter incompatible flow.
