# Build and Runtime Requirements

NameLens depends on `go-libsql`, which requires CGO and a glibc-based runtime.

## Requirements

- **CGO required**: `CGO_ENABLED=1` for builds.
- **glibc required**: Linux builds must run on glibc-based images/distros.
- **Unsupported**: Alpine/musl (unless you provide a custom libsql build).

## Go Build Tags (Important)

NameLens uses Rust FFI libraries (via CGO). To avoid Rust static library symbol
collisions, builds and tests should use the shared-library mode for sysprims.

- **Use the Make targets**: `make test`, `make check`, `make build` (these set the correct tags).
- **If running Go directly**, include: `-tags sysprims_shared`

Examples:

```bash
# Run full test suite
go test -tags sysprims_shared ./...

# Build the CLI
CGO_ENABLED=1 go build -tags sysprims_shared -o bin/namelens ./cmd/namelens
```

If you run `go test` without `-tags sysprims_shared`, you may see a linker
failure like:

```
duplicate symbol '_rust_eh_personality'
```

This happens when both `go-libsql` and `sysprims` are linked as Rust static
archives in the same test binary.

### IDE/Test Runner Note

Some IDEs run `go test` without project-specific tags. Ensure your editor/test
runner includes `sysprims_shared` (or set `GOFLAGS="-tags=sysprims_shared"` in
your shell when debugging locally).

## Recommended Images

- `debian:bookworm-slim`
- `gcr.io/distroless/base-debian12`

## Notes

- Static linking on Linux can be fragile with CGO dependencies. If CI builds
  fail with static link flags, prefer dynamic glibc linking and document the
  requirement.
- macOS builds must run on native runners for CGO (Intel + Apple Silicon).
