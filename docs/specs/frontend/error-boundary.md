# Render Error Boundary

## Symptom

An exception thrown while React renders (bad data, a failed lazy chunk, a
component lifecycle bug) unmounts the entire tree. The WebView then shows a
blank white window with no navigation or recovery action.

## Contract

- `frontend/src/main.tsx` wraps `<App />` in `AppErrorBoundary`; the boundary is
  the outermost React node so it also catches failures in `App` itself.
- The fallback is a full-screen localized recovery card: it shows the error
  message plus a reload button. It must not depend on router, stores, or other
  app state that may be the source of the failure.
- The reload action defaults to `window.location.reload()`; tests inject
  `onReload` instead of mocking `window.location`.
- `componentDidCatch` logs the error and component stack to the WebView console.
  There is no frontend remote logger.

## Limits

Error boundaries catch render and lifecycle errors only. Errors thrown inside
event handlers, timers, or async callbacks still need local handling (`catch` /
`try`); the boundary is the last-resort white-screen guard, not a replacement
for error handling at the source.

## Verification

- `frontend/src/components/AppErrorBoundary.test.tsx` throws from a child,
  asserts the recovery card replaces the blank tree, and asserts the reload
  action fires.
- Add future page-level boundaries only if the shell must survive a page crash;
  do not duplicate the root fallback copy.
