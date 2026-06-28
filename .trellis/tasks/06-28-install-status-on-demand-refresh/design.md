# 设计：InstallStatus 按需触发本地与远程元数据刷新

## 1. 设计目标

在不破坏现有四态语义（new/installed/update/conflict）与不引入远程同步阻塞的前提下，让搜索列表 `InstallStatus` 重建旧实现的"缓存读取 + 异步回填"链路，适配重构后的 versionID 键缓存结构。

## 2. 改动边界

```
app.go                      — GetDownloadStates 增加 emitter 回调，注册新事件 download-states-updated
  └─ downloader/download.go — GetDownloadStates 透传 emitter 回调
       └─ modbridge/modbridge.go
             ├─ InstallStatus      — R1: 读 DB 缓存；R3: 缓存 miss 时记录待回填 version
             ├─ DownloadStates     — R2: 实例未扫描时同步扫描；汇总待回填 version 后异步解析
             ├─ resolveVersionModIDs (新) — R1: version.ModIDs → database.GetVersionModIDs 同步读取
             ├─ ensureInstanceModsScanned (新) — R2: 复用 minecraft.ScanVersionMods
             └─ backfillVersionModIDs (新) — R3: 异步调用 VersionModIDs + 去重守卫 + emitter
frontend/src/stores/downloadSearch.ts — 新增 download-states-updated 事件监听 → refreshDownloadStates()
```

不动 `InstallStatusPrecise`、`VersionModIDs`、`database`、`providers`、前端 `refreshDownloadStates` 现有调用时机。

## 3. 核心数据流

### 3.1 InstallStatus（同步路径，R1 + R3 触发点）

```
InstallStatus(req)
  ├─ ResolveVersions → version
  ├─ SHA1 命中 → BtnStatusInstalled
  ├─ 项目 SHA1 集合命中 → BtnStatusUpdate
  └─ 冲突判定:
       modIDs = resolveVersionModIDs(version)   // R1: memory → DB 同步读
       if len(modIDs) > 0:
           按 modIDs 匹配本地 → BtnStatusConflict / BtnStatusNew
       else:
           markBackfill(version, modLoader)     // R3: 记录待异步回填（去重）
           return BtnStatusNew
```

### 3.2 resolveVersionModIDs（R1 新函数）

```go
// resolveVersionModIDs 同步读取版本 modIDs：先内存字段，再 DB 缓存。
// 不发起远程调用。供搜索列表 InstallStatus 使用。
func resolveVersionModIDs(version models.ModVersion) []string {
    if modIDs := normalizedModIDs(version.ModIDs); len(modIDs) > 0 {
        return modIDs
    }
    if version.ID == "" {
        return nil
    }
    persisted, err := database.GetVersionModIDs(version.ID)
    if err != nil {
        logging.Warn("get version mod IDs from DB failed", "platformVersionID", version.ID, "error", err)
        return nil
    }
    return normalizedModIDs(persisted)
}
```

与 `VersionModIDs` 的区别：**不调用 `parseRemoteModJar`**，仅做同步两级读取。

### 3.3 DownloadStates（R2 同步扫描 + R3 异步回填编排）

```go
func DownloadStates(req appstructs.DownloadStatesRequest, onBackfillComplete func()) []appstructs.ModDownloadButtonState {
    // ... 现有参数规整 ...
    selected := global.GetSelectedVersion()
    instanceID := versionInstanceID(selected)

    // R2: 实例未扫描时同步补扫一次
    if instanceID != "" && !global.HasLocalModPathsInInstance(instanceID) {
        ensureInstanceModsScanned(selected)
    }

    if instanceID == "" || !global.HasLocalModPathsInInstance(instanceID) {
        // 扫描后仍为空 → 全 New
        for i := range req.Results { states[i] = defaultDownloadButtonState(req.Results[i]) }
        return states
    }

    // 现有并发判定流程（每个 goroutine 内调 InstallStatus）
    // InstallStatus 在 R3 缓存 miss 时通过 markBackfill 记录
    var wg sync.WaitGroup
    for i := range req.Results { ... go func() { InstallStatus(...) } ... }
    wg.Wait()

    // R3: 汇总本轮待回填 version，异步解析
    backfill := drainPendingBackfills()
    if len(backfill) > 0 && onBackfillComplete != nil {
        go func() {
            for _, v := range backfill {
                backfillVersionModIDs(v.version, v.modLoader)  // 含去重守卫
            }
            onBackfillComplete()  // 通知 app.go 发事件
        }()
    }
    return states
}
```

### 3.4 ensureInstanceModsScanned（R2 新函数）

```go
// ensureInstanceModsScanned 在实例本地 mod 索引为空时同步扫描一次。
// 与 app.refreshVersionMods 等价但自包含于 modbridge（不依赖 app.go）。
func ensureInstanceModsScanned(selected mcstructs.VersionInfo) {
    instanceID := versionInstanceID(selected)
    if instanceID == "" { return }
    mcDir := global.GetMinecraftDir()
    versionDir := minecraft.VersionDirPath(mcDir, selected)
    if versionDir == "" { return }
    // ScanVersionMods 内部会调 global.UpsertLocalMod 填充索引
    minecraft.ScanVersionMods(versionDir, instanceID, selected.MinecraftVersion, selected.ModLoader, mcDir)
}
```

不调 `global.ClearLocalModsByInstance`（本就是空的，无需清）；不更新 `global.SetVersionsForDir`（那是 app.go 的职责，本函数只补本地 mod 索引）。

### 3.5 backfillVersionModIDs（R3 异步回填 + 去重守卫）

```go
var (
    backfillMu     sync.Mutex
    backfillInflight = make(map[string]struct{})  // key: version.ID
)

type pendingBackfill struct {
    version   models.ModVersion
    modLoader string
}

var pendingBackfills []pendingBackfill  // InstallStatus 写入，DownloadStates 排空

func markBackfill(version models.ModVersion, modLoader string) {
    if version.ID == "" { return }
    backfillMu.Lock()
    defer backfillMu.Unlock()
    if _, inflight := backfillInflight[version.ID]; inflight { return }
    pendingBackfills = append(pendingBackfills, pendingBackfill{version, modLoader})
}

func drainPendingBackfills() []pendingBackfill {
    backfillMu.Lock()
    defer backfillMu.Unlock()
    out := pendingBackfills
    pendingBackfills = nil
    return out
}

func backfillVersionModIDs(version models.ModVersion, modLoader string) {
    if version.ID == "" { return }
    backfillMu.Lock()
    if _, inflight := backfillInflight[version.ID]; inflight {
        backfillMu.Unlock()
        return
    }
    backfillInflight[version.ID] = struct{}{}
    backfillMu.Unlock()
    defer func() {
        backfillMu.Lock()
        delete(backfillInflight, version.ID)
        backfillMu.Unlock()
    }()

    // VersionModIDs 会读 memory → DB → remote parse，并写回 DB
    _ = VersionModIDs(version, modLoader)
}
```

去重两层：
1. `markBackfill` 检查 `backfillInflight` —— 同一渲染批次内不重复入队
2. `backfillVersionModIDs` 再次检查 `backfillInflight` —— 跨批次/并发调用不重复解析

## 4. 事件通道

### 4.1 新事件：download-states-updated

`app.GetDownloadStates` 提供 emitter 回调：

```go
func (a *App) GetDownloadStates(req appstructs.DownloadStatesRequest) []appstructs.ModDownloadButtonState {
    return downloader.GetDownloadStates(req, func() {
        runtime.EventsEmit(a.ctx, downloadStatesUpdatedEvent)
    })
}
```

`downloader.GetDownloadStates` 透传：
```go
func GetDownloadStates(req appstructs.DownloadStatesRequest, onBackfillComplete func()) []appstructs.ModDownloadButtonState {
    return modbridge.DownloadStates(req, onBackfillComplete)
}
```

### 4.2 前端监听

`downloadSearch.ts` 的 `start()` 新增：
```ts
const downloadStatesUpdatedEvent = "download-states-updated";
this.stopListeningDownloadStatesUpdated = EventsOn(downloadStatesUpdatedEvent, () => {
    void this.refreshDownloadStates();
});
```

`stop()` 对应清理。事件常量与现有事件常量并列声明。

## 5. 并发与性能

- `DownloadStates` 现有 10 路并发 `InstallStatus`，R1 的 `database.GetVersionModIDs` 为内存读，无性能影响。
- R2 的 `ensureInstanceModsScanned` 在并发判定前**同步**执行一次，开销与实例 mod 数量成正比（与实例切换时扫描同等开销），可接受。
- R3 异步回填在 `DownloadStates` 返回后发起，不阻塞首次渲染。10 条结果最多 10 个远程 JAR 解析 goroutine，受 `backfillInflight` 去重，下次渲染命中 DB 缓存即不再发起。
- 远程解析失败不重试（与 `VersionModIDs` 现有行为一致）。

## 6. 兼容性

- 不改 `models.ModVersion` 结构，不改 `cacheVersion`。
- `downloader.GetDownloadStates` 签名变化（+1 参数），但仅 `app.GetDownloadStates` 一处调用，同步更新。
- `modbridge.DownloadStates` 签名变化（+1 参数），仅 `downloader.GetDownloadStates` 一处调用。
- 新增事件 `download-states-updated` 与现有事件不冲突。
- 前端新增监听不影响现有事件处理。

## 7. 风险

- **R2 同步扫描延迟**：首次搜索且实例未扫描时，`DownloadStates` 会同步扫描整个 mods 目录。若 mods 数量大（100+），首次搜索会延迟 1–2 秒。可接受（与实例切换同等开销，且只发生一次）。
- **R3 远程解析风暴**：10 条结果全部缓存 miss → 10 个 HTTP Range 请求并发。`backfillInflight` 仅去重同 version，不去重并发数。若需限制并发，可在 `backfillVersionModIDs` 外加信号量。MVP 不限制，后续按需加。
- **事件循环**：`onBackfillComplete` 触发前端 `refreshDownloadStates` → 再次调 `GetDownloadStates` → 若仍有 miss 再次回填。但因 DB 已写入，下次 `resolveVersionModIDs` 命中缓存，不再 `markBackfill`，循环终止。
