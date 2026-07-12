# Implementation Plan

1. Inventory current package boundaries and run baseline Go/frontend checks.
2. Add the root architecture map and frontend/core navigation guides.
3. Split root Wails adapter declarations and methods by responsibility while
   preserving the `main` package and exported signatures.
4. Add focused appcore package documentation and extract only cohesive
   declarations/helpers that reduce file ambiguity.
5. Run formatting, Go tests, frontend build/lint, and a targeted static scan
   for stale paths or duplicate API definitions.
6. Run `mimo run` with a random-workflow prompt, inspect the output for correct
   file/function references, and record the evidence in this task.
7. Update relevant Trellis specs with any durable structure convention, then
   run the final quality checklist.
