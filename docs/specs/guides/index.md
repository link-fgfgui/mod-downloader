# Thinking Guides

These are short decision checklists, not implementation specifications. Read
only the guide whose trigger matches the change, then follow the linked package
contract for signatures and tests.

## Available Guides

| Guide | Use when |
| --- | --- |
| [Code Reuse](./code-reuse-thinking-guide.md) | Adding helpers, constants, converters, components, or payload readers |
| [Cross-Layer](./cross-layer-thinking-guide.md) | Data crosses frontend, Wails, service, provider, downloader, storage, or Minecraft layers |

## Review Triggers

- Cross-layer change, changed payload, or new event: read the Cross-Layer guide.
- New utility, constant, converter, or duplicated state update: read Code Reuse.
- Before changing a config key, event, or shared field, use `rg` to find every
  reference and update its owner plus consumers.
- Verify review warnings against the actual data source and call path; do not
  accept a finding based only on a generic trust-boundary assumption.

After resolving a non-trivial bug, add a concise project-specific prevention rule to the smallest
relevant guide or topic spec under `docs/specs/`.
