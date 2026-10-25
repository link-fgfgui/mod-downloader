# Mod Metadata Parsing

## Scenario: Reading JSON metadata exactly like Fabric Loader

### 1. Scope / Trigger

Use this contract whenever a JAR-declared JSON file (`fabric.mod.json`,
`META-INF/jarjar/metadata.json`) is decoded into `structs/minecraft.ModInfo`.

### 2. Reader Comparison (verified)

Fabric Loader parses `fabric.mod.json` with the Gson `JsonReader` vendored in
`net.fabricmc.loader.impl.lib.gson`. The released jar exposes no public
`setLenient`, so the loader always runs it in non-lenient mode, and
`ModMetadataParser` stops reading after `endObject()`.

The table below was verified by running the same byte sequences through
fabric-loader 0.20.1's reader, google/gson 2.8.9 and 2.15.0, and
`encoding/json` (see `TestUnmarshalModMetadataJSONGsonBoundaryParity`).

| Construct | Gson / fabric-loader | encoding/json | Implementation |
|---|---|---|---|
| Root is not an object (`null`, array, scalar) | Rejected by `ModMetadataParser` | `null` is a no-op; other types error | `hasJSONObjectRoot` rejects before decoding |
| Leading UTF-8 BOM | Accepted (skipped by `fillBuffer`) | Rejected | Stripped |
| Raw C0 control chars in strings/keys | Accepted | Rejected | Escaped |
| `\'` escape in a double-quoted string | Accepted as `'` | Rejected | Unescaped |
| Backslash + LF inside a string | Accepted as LF | Rejected | Rewritten to `\n` |
| Bytes after the root object | Ignored | Rejected | `json.Decoder.Decode` reads one value |
| Trailing commas, comments, single quotes, unquoted names/values, `=`, `;`, `NaN` | Rejected | Rejected | Unchanged |
| `01`, `+1`, `.5`, `5.`, `1e` | Rejected | Rejected | Unchanged |
| `\q`, `\x41`, `\v`, `\0`, backslash + CR, short `\u` | Rejected | Rejected | Unchanged |
| Invalid UTF-8 / lone surrogate escape | Accepted | Accepted, replaced with U+FFFD | Unchanged |
| Duplicate names | Accepted, last wins | Accepted, last wins | Unchanged |
| `1e400` in an untyped value | Accepted | Rejected when decoding into `any` (float64 overflow) | `Depends` uses `json.RawMessage` |

### 3. Algorithm Invariants

- `hasJSONObjectRoot` skips at most one leading BOM and the four JSON
  whitespace bytes, then requires `{`. This matches the `peek() != BEGIN_OBJECT`
  check in `ModMetadataParser` and also covers `null`, which a Go struct
  decoder would otherwise treat as a successful no-op.
- `normalizeModMetadataJSON` is a single left-to-right byte transducer. Outside
  strings every byte is copied. Inside strings it rewrites only C0 control
  bytes, `\'`, and backslash-LF; the backslash is deferred until the next byte
  is known so replacement never duplicates it. Structural bytes (`{}[],:`) are
  never rewritten because they can only be copied outside strings.
- If the input needs no rewrite, the returned bytes are identical to the input.
  The function never validates structure itself; the decoder does.
- The retry uses `json.Decoder.Decode` against a fresh value. That decodes a
  single JSON value and ignores whatever follows, which is exactly the
  `endObject()`-then-return behavior of `ModMetadataParser`. The result is
  assigned to the caller's value only on success, so a failed retry cannot leak
  a partially decoded struct.
- Time is O(n) and space is O(n) in the worst case (three linear passes: strict
  decode, normalize, single-value decode). The normalizer is iterative and has
  no recursion or unbounded allocation per byte beyond the output buffer.

### 4. Contracts

- `unmarshalModMetadataJSON` tries strict decoding first and normalizes only on
  failure. Normalization rewrites exactly the three string/BOM deviations above;
  trailing bytes are handled by the single-value decoder, not rewritten.
- Never enable general lenient JSON behavior (comments, single quotes,
  unquoted names, trailing commas, `NaN`, non-standard numbers). Both readers
  reject those, so accepting them would diverge from the loader.
- Decode `depends` as `map[string]json.RawMessage`. Only its keys are used and
  `RawMessage` avoids the float64 overflow rejection that `map[string]any`
  would introduce for values such as `1e400`.
- Keep the original strict error in the returned message when the normalized
  retry also fails, so genuinely malformed metadata stays diagnosable.
- Do not relax JSON parsing for unrelated files (launcher profiles, pack
  manifests, provider responses).

### 5. Scope Boundary: Field Extraction

The helper aligns the JSON reader, not Fabric's schema validation. Field
extraction still uses `encoding/json` struct decoding and therefore differs
from Fabric's manual `peek()`-based reader in three schema-level ways:

- Struct keys match case-insensitively, so `"ID"` binds to `ID`; Fabric's
  switch only accepts `"id"`.
- JSON `null` for a string, map, or slice field is a no-op in Go; Fabric's
  typed readers reject it. The root-level `null` case is covered above.
- `schemaVersion` type/range and required-field checks are not replicated.

These only affect malformed metadata that Fabric would reject at the schema
level. They do not change the JSON syntax acceptance boundary. If exact schema
parity is ever needed, decode the top level into `map[string]json.RawMessage`
and look up exact keys instead of using struct tags.

### 6. Tests Required

- Table test covering every row above, including accepted/rejected status and
  the decoded string values for `\'`, backslash-LF, raw control chars, and
  escaped sequences that must be preserved.
- Root boundary: `null`, arrays, scalars, leading garbage, and BOM + control
  chars.
- Fabric metadata with literal newlines/tabs in `description` still yields the
  mod ID, name, version, description, and required dependencies.
- Structurally invalid JSON still returns an error.
- Parse a published mod that failed only because of this difference (for
  example Entity Texture Features 7.2.6) and confirm its mod ID.

### 7. Wrong vs Correct

Wrong:

```go
if err := json.Unmarshal(data, &meta); err != nil {
    return parsedModMetadata{}, fmt.Errorf("parse fabric.mod.json: %w", err)
}
```

Correct:

```go
if err := unmarshalModMetadataJSON(data, &meta); err != nil {
    return parsedModMetadata{}, fmt.Errorf("parse fabric.mod.json: %w", err)
}
```
