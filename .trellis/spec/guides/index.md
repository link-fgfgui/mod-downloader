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

After a bug, add a concise project-specific prevention rule to the smallest
relevant guide. Do not copy generic Trellis implementation guidance into this
directory.

## Trellis Runtime Boundary

In an initialized project, Trellis runtime inputs include `.trellis/workflow.md`,
`.trellis/scripts/`, `.trellis/tasks/`, `.trellis/spec/`, `.trellis/.runtime/`,
`.trellis/.template-hashes.json`, and `.trellis/config.yaml`. Changes to these
files can affect hooks, task context, or `trellis update`; verify the relevant
script and parser before editing them.

The Trellis source repository has additional concerns such as registering
Python template files and maintaining versioned docs-site trees. Those rules
belong to Trellis source maintenance, not to this application's business-code
specs. Do not delete initialized runtime files merely because their source
template machinery is not present here.
