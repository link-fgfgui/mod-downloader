# Project Structure Refactor For AI Navigability

## Problem

The application works, but its architecture is difficult to reconstruct from
the filesystem alone. The root Wails adapter (`app.go`) contains a large mixed
surface of lifecycle code, event translation, dialogs, and API forwarding;
`core/appcore/service.go` contains many unrelated domains; and the frontend
has no single map from user workflows to backend boundaries. This causes a
single-pass code-reading agent to spend its context budget discovering
ownership instead of reasoning about behavior.

## Requirements

- Establish a concise, repository-local architecture map covering the root
  Wails shell, `core` packages, frontend views/stores/components, generated
  bindings, persistence, and outbound providers.
- Make ownership and dependency direction explicit at package boundaries.
- Split the most misleading mixed files into responsibility-oriented files
  without changing public Wails or core APIs.
- Add package-level documentation where a directory's purpose is otherwise
  ambiguous, and update existing README guidance to match the actual tree.
- Preserve the current runtime behavior and generated binding compatibility.
- While touching code, fix directly observed correctness issues and add a
  regression test for each such fix.
- Record the work in this Trellis task and leave enough validation evidence for
  another agent to reproduce the result.

## Acceptance Criteria

- A new contributor can start at the architecture map and identify the files
  involved in at least the download, local-mod scan, and settings workflows.
- Root adapter responsibilities are separated into named files with no API
  signature changes.
- `core/appcore` has an index/package guide that points to domain files rather
  than requiring a scan of one large service file.
- `go test ./...` passes at the repository root and `cd core && go test ./...`
  passes; frontend build/lint passes when dependencies are available.
- A repeatable `mimo run` prompt asking for one random workflow returns file
  and function references without inventing package ownership; capture the
  prompt and a concise result in the task notes.
- No generated, runtime, database, or dependency artifacts are committed.
