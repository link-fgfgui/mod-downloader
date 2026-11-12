# Error Handling and Empty Results

Core methods report failures through `...WithError` variants; the App bindings
use those variants. The frontend must keep "empty" and "failed" distinct:

- A missing pin is an empty result, so opening the pin dialog must not show a
  failure toast.
- Delete and remove actions treat `false` as "nothing changed", not as an error.
- Provider lookup failures appear as operation errors; a successful empty search
  appears as an empty list.
- `SearchModsUpdate.Error` is populated only on the final page and may accompany
  partial results from healthy providers. Show the message while keeping the
  partial results usable.
- A canceled cache-directory selection is not an error; a failed cache switch
  must not look like a cancel.
- A failed settings save must not show a success toast or auto-save success
  state. Reject when a `SettingsView` carries `SaveError`, and treat rejected
  `SaveTheme` / `SaveLanguage` calls as save failures.
- Display copy comes from `locales/en.ts` and `locales/zh.ts`; do not surface a
  raw Go error type.

See `go doc github.com/link-fgfgui/mod-downloader/core/appcore` for the core
side of the contract.
