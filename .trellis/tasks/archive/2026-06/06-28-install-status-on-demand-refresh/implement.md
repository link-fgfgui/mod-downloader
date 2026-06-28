# 实施计划：InstallStatus 按需触发本地与远程元数据刷新

## 执行清单

### Step 1: modbridge — 新增 R1 同步读取函数

- [ ] 在 `modbridge/modbridge.go` 新增 `resolveVersionModIDs(version models.ModVersion) []string`
  - 先 `normalizedModIDs(version.ModIDs)`，非空则返回
  - `version.ID != ""` 时读 `database.GetVersionModIDs(version.ID)`，错误仅 Warn
  - 不调用 `parseRemoteModJar`
- [ ] `InstallStatus` 冲突判定段（[modbridge.go:149-156](file:///home/link/Documents/go_proj/worktrees/mod-downloader/refresh/mod-downloader/modbridge/modbridge.go#L149-L156)）：
  将 `normalizedModIDs(version.ModIDs)` 改为 `resolveVersionModIDs(version)`
- [ ] 更新该段注释，去掉"only use already persisted mod IDs"，改为说明读 memory + DB 缓存

**验证**：`go build ./modbridge/...`

### Step 2: modbridge — 新增 R3 去重与异步回填骨架

- [ ] 新增包级状态：
  ```go
  var (
      backfillMu       sync.Mutex
      backfillInflight = make(map[string]struct{})
      pendingBackfills []pendingBackfill
  )
  type pendingBackfill struct {
      version   models.ModVersion
      modLoader string
  }
  ```
- [ ] 新增 `markBackfill(version, modLoader)`：`version.ID == ""` 跳过；加锁检查 `backfillInflight`，已存在则跳过；否则追加到 `pendingBackfills`
- [ ] 新增 `drainPendingBackfills() []pendingBackfill`：加锁取走并清空 `pendingBackfills`
- [ ] 新增 `backfillVersionModIDs(version, modLoader)`：双重检查 `backfillInflight`（进入时标记，defer 释放）；调用 `VersionModIDs(version, modLoader)`（忽略返回值，仅触发 DB 写回）
- [ ] `InstallStatus` 的 R3 触发点：当 `resolveVersionModIDs` 返回空时，调 `markBackfill(version, req.ModLoader)` 后返回 `BtnStatusNew`

**验证**：`go build ./modbridge/...`

### Step 3: modbridge — 改造 DownloadStates 签名与 R2/R3 编排

- [ ] `DownloadStates` 签名改为 `func DownloadStates(req appstructs.DownloadStatesRequest, onBackfillComplete func()) []appstructs.ModDownloadButtonState`
- [ ] 在 `instanceID` 计算后、现有早返回前，插入 R2：
  ```go
  if instanceID != "" && !global.HasLocalModPathsInInstance(instanceID) {
      ensureInstanceModsScanned(selected)
  }
  ```
- [ ] 新增 `ensureInstanceModsScanned(selected mcstructs.VersionInfo)`：
  - 计算 `instanceID`、`mcDir := global.GetMinecraftDir()`、`versionDir := minecraft.VersionDirPath(mcDir, selected)`
  - `versionDir == ""` 则返回
  - 调 `minecraft.ScanVersionMods(versionDir, instanceID, selected.MinecraftVersion, selected.ModLoader, mcDir)`
- [ ] 在 `wg.Wait()` 后、`return states` 前，插入 R3 编排：
  ```go
  backfill := drainPendingBackfills()
  if len(backfill) > 0 && onBackfillComplete != nil {
      go func() {
          for _, b := range backfill {
              backfillVersionModIDs(b.version, b.modLoader)
          }
          onBackfillComplete()
      }()
  }
  ```

**验证**：`go build ./modbridge/...`（此时 downloader 会编译失败，下一步修复）

### Step 4: downloader — 透传 emitter 回调

- [ ] `downloader/download.go` 的 `GetDownloadStates` 签名改为 `func GetDownloadStates(req appstructs.DownloadStatesRequest, onBackfillComplete func()) []appstructs.ModDownloadButtonState`
- [ ] 函数体改为 `return modbridge.DownloadStates(req, onBackfillComplete)`

**验证**：`go build ./downloader/...`

### Step 5: app.go — 注册 emitter + 新事件常量

- [ ] 在 `app.go` 顶部事件常量区新增 `const downloadStatesUpdatedEvent = "download-states-updated"`（与 `searchModsUpdatedEvent` 等并列）
- [ ] `GetDownloadStates` 改为：
  ```go
  func (a *App) GetDownloadStates(req appstructs.DownloadStatesRequest) []appstructs.ModDownloadButtonState {
      return downloader.GetDownloadStates(req, func() {
          runtime.EventsEmit(a.ctx, downloadStatesUpdatedEvent)
      })
  }
  ```

**验证**：`go build ./...`

### Step 6: 前端 — 新增事件监听

- [ ] `frontend/src/stores/downloadSearch.ts` 顶部事件常量区新增 `const downloadStatesUpdatedEvent = "download-states-updated";`
- [ ] `state()` 新增 `stopListeningDownloadStatesUpdated: null as (() => void) | null,`
- [ ] `start()` 开头守卫条件加入 `|| this.stopListeningDownloadStatesUpdated`
- [ ] `start()` 内新增：
  ```ts
  this.stopListeningDownloadStatesUpdated = EventsOn(downloadStatesUpdatedEvent, () => {
      void this.refreshDownloadStates();
  });
  ```
- [ ] `stop()` 内新增 `this.stopListeningDownloadStatesUpdated?.()` 与置 null

**验证**：`cd frontend && npm run build`（或 type-check）

### Step 7: 单元测试

- [ ] `modbridge/modbridge_test.go` 新增：
  - `TestResolveVersionModIDsReadsMemoryField` — `version.ModIDs` 非空时直接返回
  - `TestResolveVersionModIDsReadsDBCache` — `version.ModIDs` 为空、`version.ID` 非空时读 DB（需 stub `database.GetVersionModIDs` 或用真实 DB 临时文件）
  - `TestMarkBackfillDeduplicatesByVersionID` — 同一 version.ID 多次 `markBackfill` 只入队一次
  - `TestBackfillVersionModIDsGuardsInflight` — `backfillInflight` 标记期间再次调用直接返回
- [ ] 若 `database.GetVersionModIDs` 难以 stub，用 `database` 包的临时 DB 文件构造测试数据（参考 `database/mods_test.go` 既有模式）

**验证**：`go test ./modbridge/... -run -v`

### Step 8: 全量验证

- [ ] `go vet ./...`
- [ ] `go test ./...`
- [ ] 前端 `npm run build` 或类型检查通过

## 验证命令汇总

```bash
go build ./...
go vet ./...
go test ./...
cd frontend && npm run build
```

## 风险点与回滚

- **风险点 1**：Step 3 改 `DownloadStates` 签名是破坏性变更，若遗漏调用点会编译失败。已知调用链仅 `app → downloader → modbridge`，编译驱动可捕获。
- **风险点 2**：Step 6 前端若漏改 `stop()` 清理，会导致组件 deactivate 后事件泄漏。务必配对新增。
- **回滚点**：每一步独立可回滚；若 R3 异步回填引发事件风暴，可临时把 `markBackfill` 改为 no-op（退化为仅 R1+R2，等价旧实现缓存读取但不触发远程）。
