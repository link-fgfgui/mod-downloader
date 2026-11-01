# 全局单实例（端口锁）

## 结论

mod-downloader 的 GUI 与 TUI 共享同一个进程级单实例守卫：启动时先绑定
`127.0.0.1:18801`（`httpserver.DefaultAddr`，也是浏览器扩展的接收端口）。
**端口即锁**，绑定成功才允许继续启动；绑定失败返回 `instance.ErrAlreadyRunning`，
适配器必须在创建窗口、打开共享 SQLite/缓存之前中止。

- 实现：`core/instance`（`Acquire` / `Guard`）
- 接入点：`core/host` 的 `Start`（GUI 与 TUI 都走这里，无第二份实现）
- 失败类型：`instance.ErrAlreadyRunning`（`errors.Is` 可判定）
- 测试覆盖：`core/instance/instance_test.go`、`core/host/host_lifecycle_test.go`

## 工作方式

1. `host.Start` 在启动 core 之前调用 `instance.Acquire(instanceAddr)`。
2. `Acquire` 对 `127.0.0.1:18801` 执行 `net.Listen`：
   - 成功：返回 `Guard`，listener 由 host 持有到 `Shutdown`；
   - 失败：返回包装后的 `ErrAlreadyRunning`，`Host.Start` 直接失败，不初始化 core。
3. 扩展 HTTP 服务器不再自行 bind 18801：当 `HTTPAddr` 解析结果与守卫地址相同时，
   host 把守卫的 listener 交给 `httpserver.Options.Listener`，同一端口只 bind 一次。
4. `Shutdown` 先停 HTTP 服务器与 core，再关闭守卫；下一个进程即可重新绑定。
5. GUI 的 `App.ServiceStartup` 对 `ErrAlreadyRunning` 返回错误，Wails 尚未显示窗口就会
   退出（不会出现第二个窗口或第二个 SQLite 连接）；`main` 识别该错误后以退出码 0
   安静退出。TUI 的 `host.Start` 错误沿 `runWith` 返回，同样在进入 tview 前退出。

## 约束

**Don't**

- 不要用锁文件、PID 文件或 Wails `SingleInstance`（D-Bus）另起一套判断：单实例规则
  只能有一份，端口已经是扩展服务器的天然锁点。
- 不要在 `app.go`、`main.go`、TUI `internal/` 或前端复制 `18801` 常量；需要地址时用
  `instance.DefaultAddr` / `httpserver.DefaultAddr`。
- 不要在宿主测试或 TUI 测试里绑定真实 18801：测试必须把
  `host.Options.InstanceAddr` 与 `HTTPAddr` 设为 `127.0.0.1:0`，否则桌面端一开就会
  让整套测试失败。
- 不要在 `Acquire` 失败后继续初始化 core 或显示主窗口；共享存储一旦被第二个进程
  打开，单实例就失去意义。

**Instead**

- 适配器只消费 `ErrAlreadyRunning`：GUI 在建窗前中止并安静退出，TUI 直接返回错误。
- 需要“唤起第一个窗口/传递启动参数”时，再在守卫端口上增加显式的 handoff 端点，
  并把协议写进本文件；当前实现只保证检测与拒绝启动。
- 新增前端时，复用 `host.Start` 即可自动获得守卫，不要另写启动生命周期。

## 取舍

- 端口被**其他程序**占用时同样拒绝启动：这是“端口即锁”的必然结果，错误信息会带上
  底层 `bind: address already in use`，便于排查。若要改成“端口被外部程序占用时仍然
  允许启动”，必须先设计第二把锁，不能简单忽略该错误。
- 第二个实例目前不会把第一个窗口提到前台；这是有意的范围控制，避免在守卫 listener
  上提前启动 HTTP 服务而引入启动竞态。
