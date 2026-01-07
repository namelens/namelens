# Build and Runtime Requirements

NameLens depends on `go-libsql`, which requires CGO and a glibc-based runtime.

## Requirements

- **CGO required**: `CGO_ENABLED=1` for builds.
- **glibc required**: Linux builds must run on glibc-based images/distros.
- **Unsupported**: Alpine/musl (unless you provide a custom libsql build).

## Recommended Images

- `debian:bookworm-slim`
- `gcr.io/distroless/base-debian12`

## Notes

- Static linking on Linux can be fragile with CGO dependencies. If CI builds
  fail with static link flags, prefer dynamic glibc linking and document the
  requirement.
- macOS builds must run on native runners for CGO (Intel + Apple Silicon).
