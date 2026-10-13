# Favorite version migration implementation

## Steps

1. Start after SQL storage/reference APIs are available.
2. Add migration request/preview/result structs in appcore or shared structs if Wails needs them.
3. Implement preview using resolved list contents and provider lookups.
4. Implement apply by reusing preview and favorite upsert helpers.
5. Add Wails adapter methods and regenerate bindings.
6. Add tests with provider seams or cached project/version data where possible.

## Validation

- `cd core && go test ./appcore ./database`
- `cd core && go test ./...`
- `go test ./...`
- `wails generate module`
- `npm run build --prefix frontend`
