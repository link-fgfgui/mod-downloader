# Backend Development Guidelines

> Executable conventions for the Go Wails shell and reusable `core/` module.

---

## Overview

Read the package index first, then only the topic files named by its
pre-development checklist.

---

## Guidelines Index

| Guide | Description | Status |
|-------|-------------|--------|
| [Directory Structure](./directory-structure.md) | Repo split, core submodule layout, `models` single-source-of-truth convention, bridge pattern | Filled |
| [Storage Guidelines](./storage-guidelines.md) | Metadata cache, SQLite user data, versioning, and persistence boundaries | Filled |
| [Error Handling](./error-handling.md) | Go error propagation, adapter translation, logging | Filled |
| [Quality Guidelines](./quality-guidelines.md) | Code standards, forbidden patterns (aliases, re-exports, parallel converters) | Filled |
| [Logging Guidelines](./logging-guidelines.md) | Structured logging, stderr/file policy, config bootstrap, log levels, sensitive data | Filled |

---

All documents describe current code contracts, not generic Go advice. When a
new boundary or bug pattern is discovered, update the smallest relevant topic
file and its index link.
