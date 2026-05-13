# Repository Guidelines

## Project Structure & Module Organization

Lathe is a Go framework for generating Cobra CLIs from pinned API specs.

- `cmd/specsync` and `cmd/codegen` contain the executable entry points for spec syncing and code generation.
- `internal/sourceconfig`, `internal/specsync`, `internal/codegen`, `internal/overlay`, and `internal/auth` hold implementation-only packages.
- `pkg/runtime`, `pkg/config`, and `pkg/lathe` are reusable runtime/library surfaces for generated CLIs.
- Tests live beside implementation as `*_test.go`; golden fixtures live under package-local `testdata/`.
- `examples/` contains runnable examples, and `docs/` contains architecture material and images.
- Do not commit generated output under `internal/generated/` or upstream clones under `.cache/`.

## Build, Test, and Development Commands

- `make help` lists available targets.
- `make bootstrap` runs `sync-specs` and `gen` for first-time generated CLI setup.
- `make sync-specs` fetches specs declared in `specs/sources.yaml`.
- `make gen` regenerates `internal/generated` from cached specs.
- `make check` is the local quality gate: format check, `go vet`, `golangci-lint`, and tests.
- `make test` runs `go test ./...`; CI runs `go test -race ./...`.
- `make fmt`, `make fmt-check`, `make vet`, `make lint`, and `make tidy` run focused maintenance tasks.
- `make example-petstore` or `bash examples/richapi/run.sh` exercise end-to-end examples.

## Coding Style & Naming Conventions

Use standard Go formatting (`gofmt`) and keep package boundaries strict: `internal/**` is private implementation, while `pkg/**` is downstream-facing API. Prefer mechanical names that match existing code, such as `Build`, `Parse`, `Normalize`, `RenderModule`, and `NewApp`. Wrap errors with context using `fmt.Errorf("...: %w", err)`. For data-driven behavior, extend `cli.yaml`, `specs/sources.yaml`, or overlay config instead of hard-coding values.

## Testing Guidelines

Add tests next to the package being changed. Use table-driven tests where they keep cases readable, and use `testdata/*.golden.json` for stable codegen or normalization expectations. Run `make check` before opening a PR; for runtime-sensitive changes, also run the relevant example script or `go test -race ./...`.

## Commit & Pull Request Guidelines

Commit messages follow Conventional Commits, for example `feat: add openapi3 backend` or `fix(runtime): preserve debug output`. Sign off commits with `git commit -s`. Keep PRs focused, describe the problem and fix, link related issues, and include the exact verification commands you ran. New features should include tests; CLI behavior changes should include updated docs or examples when user-facing output changes.

## Security & Configuration Tips

Report vulnerabilities through `SECURITY.md`. Treat specs, hosts, and generated code paths as untrusted inputs: validate paths, avoid committing credentials, and prefer explicit host resolution over ambient global state.
