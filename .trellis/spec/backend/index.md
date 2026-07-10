# Backend Development Guidelines

> Best practices for backend development in this project.

---

## Overview

This directory contains guidelines for backend development. Fill in each file with your project's specific conventions.

---

## Guidelines Index

| Guide | Description | Status |
|-------|-------------|--------|
| [Directory Structure](./directory-structure.md) | Repo split, core submodule layout, `models` single-source-of-truth convention, bridge pattern | Filled |
| [Storage Guidelines](./storage-guidelines.md) | Metadata cache, SQLite user data, versioning, and persistence boundaries | Filled |
| [Error Handling](./error-handling.md) | Error types, handling strategies | To fill |
| [Quality Guidelines](./quality-guidelines.md) | Code standards, forbidden patterns (aliases, re-exports, parallel converters) | Filled |
| [Logging Guidelines](./logging-guidelines.md) | Structured logging, stderr/file policy, config bootstrap, log levels, sensitive data | Filled |

---

## How to Fill These Guidelines

For each guideline file:

1. Document your project's **actual conventions** (not ideals)
2. Include **code examples** from your codebase
3. List **forbidden patterns** and why
4. Add **common mistakes** your team has made

The goal is to help AI assistants and new team members understand how YOUR project works.

---

**Language**: All documentation should be written in **English**.
