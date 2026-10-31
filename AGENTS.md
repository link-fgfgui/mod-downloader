# AGENTS

## UI verification

Use the Playwright MCP browser tools for all UI checks. Do not use `spectacle`,
`xwininfo`, `uinput`, or other desktop screenshot/input automation: the app is
checked through a browser against the headless Wails server, which is faster,
scriptable, and gives us DOM-level assertions.

### Headless server workflow

```bash
wails3 task build:server
./bin/mod-downloader-server
```

The server listens on `http://127.0.0.1:9245` by default. Override with
`MOD_DOWNLOADER_SERVER_PORT` if needed. Then use Playwright MCP to navigate,
inspect the accessibility tree, click, type, and take screenshots. The server
serves the embedded production frontend and the real Go bindings, so API calls
and events run against the actual backend.

For source-level React diagnostics, run the headless server first, then start
Vite with the dev-server proxy enabled:

```bash
cd frontend
WAILS_SERVER_URL=http://127.0.0.1:9245 npm run dev -- --port 9246
```

Open `http://127.0.0.1:9246/` with Playwright. This keeps the real bindings
while letting Vite serve unminified modules and React error stacks.

Plain `cd frontend && npm run dev` does not serve the Wails bindings; use the
headless server for end-to-end UI verification.
