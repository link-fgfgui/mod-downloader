# Download Completion Sound

## Scenario: Notify After An Unfocused Successful Queue Cycle

### 1. Scope / Trigger

Use this contract for audible download completion notifications. The sound is
an attention cue only when the user is outside the app; it must not fire for
ordinary foreground work, failures, cancellations, skipped installs, or every
file in a batch.

### 2. Signatures

Backend event:

```go
const EventDownloadCompleted EventKind = "downloadCompleted"
```

Wails runtime event:

```text
download-completed
```

Frontend helpers:

```ts
appIsUnfocused(): boolean
prepareDownloadCompletionSound(): Promise<void>
playDownloadCompletionSound(): Promise<void>
```

### 3. Contracts

- Downloader invokes `OnDownloadCompleted` only after a new file reaches its
  final path. Existing targets, already-installed versions, failures, and
  cancellations do not emit completion.
- Appcore increments the historical SQLite download counter and emits
  `EventDownloadCompleted` from the same callback.
- The Wails adapter maps it to `download-completed` with no payload.
- The queue store records that at least one file succeeded in the current
  active cycle. It plays once only when a later queue snapshot transitions
  from `active=true` to `active=false`.
- Playback additionally requires `document.visibilityState === "hidden"` or
  `document.hasFocus() === false` at the final transition.
- The success flag is cleared at every active-to-inactive transition, whether
  playback occurs or the app is focused.
- First pointer or keyboard interaction prepares/resumes Web Audio so the
  later unfocused playback is less likely to be blocked by autoplay policy.
- The sound is a short generated sine chime; it adds no media dependency and
  failures are silent because notification audio must never break queue state.

### 4. Validation & Error Matrix

- Success event + active-to-inactive + unfocused -> play once.
- Multiple success events in one active cycle -> play once at final drain.
- Success event + final transition while focused -> no sound; clear flag.
- Active-to-inactive without success -> no sound.
- Failure/cancel leaves retryable queue active -> no completion transition.
- AudioContext missing, suspended resume rejected, or playback setup throws ->
  swallow the audio error; queue state remains correct.
- Store stop -> unregister queue/completion listeners and interaction handlers,
  then clear the cycle flag.

### 5. Good/Base/Bad Cases

- Good: a three-mod batch finishes while the user is in another window; one
  chime plays after the third file and queue drain.
- Base: a foreground single download finishes silently while normal UI state
  updates continue.
- Bad: play immediately on every `download-completed` event; batches produce
  several overlapping sounds.
- Bad: infer success only from `active=false`; cancellation can look like a
  completed queue.

### 6. Tests Required

- Appcore test invokes the completion callback and asserts one SQLite increment
  plus one `EventDownloadCompleted` with nil payload.
- Downloader completion regression continues to assert new-file only behavior.
- Frontend lint/type-check/build verifies event names, DOM focus APIs, Pinia
  listener state, and Web Audio types.
- Manual Wails verification should cover focused silence, unfocused sound,
  batch single sound, and failed/canceled silence when the desktop runtime is
  available.

### 7. Wrong vs Correct

Wrong:

```ts
EventsOn("download-queue-updated", state => {
  if (!state.active) playDownloadCompletionSound();
});
```

Correct:

```ts
const shouldPlay = previous.active && !next.active
  && completedInActiveCycle
  && appIsUnfocused();
completedInActiveCycle = false;
if (shouldPlay) void playDownloadCompletionSound();
```
