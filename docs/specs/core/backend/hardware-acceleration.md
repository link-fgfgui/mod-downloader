# Webview Hardware Acceleration

> Force GPU rendering on Linux; the policy is fixed when the window is created.

## Contract

- Config: `preferences.hardware_acceleration` (`*bool`, default **true**;
  `configs.DefaultHardwareAcceleration`, accessor
  `Preferences.HardwareAccelerationValue()`). The pointer keeps an unset value
  distinct from an explicit `false`.
- Environment: `PREFERS_HARDWARE_ACCELERATION`. It is parsed manually because
  cleanenv cannot bind `*bool` fields from environment variables.
- Bootstrap read: `configs.LoadHardwareAcceleration()` reads only this
  preference, without logging and without starting the core.
- Startup wiring: `main.go` maps `true` to
  `linux.WebviewGpuPolicyAlways` and `false` to
  `linux.WebviewGpuPolicyNever`, then passes a non-nil `Linux` options
  struct to `wails.Run`.
- Settings surface: `SettingsView.HardwareAcceleration` and
  `SaveHardwareAccelerationSettingsRequest` (appcore + app adapter).
  `SaveHardwareAccelerationSettings` only persists the value.

## Why the default must be overridden

Wails v2 sets `WebviewGpuPolicyNever` when `options.Linux` is nil
(https://github.com/wailsapp/wails/issues/2977). Leaving the options nil
therefore disables GPU rendering even though `WebviewGpuPolicyOnDemand` is the
zero value. An explicit non-nil policy is required to enable acceleration.

## The policy is startup-only

`wails.Run` consumes `WebviewGpuPolicy` while creating the GTK webview. It
cannot be changed on a live window. Saving the setting therefore returns the
new value but only takes effect after the app restarts; the settings UI must
keep showing a restart hint. Users on old GPUs, VMs, or remote sessions can turn
the switch off to fall back to software rendering.

### Don't: load the full config in `main.go`

**Problem**: calling `configs.Load()` before `wails.Run` just to read the GPU
preference.

**Why it's bad**: it emits log lines before logging is configured and couples
window creation to unrelated config validity (a bad theme value could silently
flip the policy).

**Instead**: call `configs.LoadHardwareAcceleration()`, which reads a minimal
struct and stays silent.
