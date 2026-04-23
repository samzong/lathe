# Petstore CLI — end-to-end example

Demonstrates the full lathe workflow: OpenAPI 3 spec → codegen → working CLI binary.

## Run it

```sh
bash examples/petstore/run.sh
```

The script creates a throwaway project in `/tmp`, runs codegen, builds a real binary, and prints `--help` output. Everything is cleaned up on exit.

## What the script does

1. Builds the `codegen` tool from this repo
2. Creates a Go module with `replace` directive pointing at local lathe
3. Writes `cli.yaml` (CLI identity) and `specs/sources.yaml` (source config)
4. Pre-stages a Petstore OpenAPI 3 spec (replaces `make sync-specs`)
5. Runs codegen → produces `internal/generated/`
6. Writes `cmd/petstore/main.go` (wiring code)
7. `go build` → runs `petstore --help`

## Expected output

```
Petstore CLI demo

Usage:
  petstore [command]

Authentication:
  auth        Authenticate petstore with a host

Modules:
  pets        pets API

Additional Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  version     Print version information
```

## Adapting for your project

See the [Quick start](../../README.md#quick-start) in the main README. The key files are:

- **`cli.yaml`** — CLI name, description, auth endpoint
- **`specs/sources.yaml`** — upstream specs pinned at immutable tags
- **`cmd/<name>/main.go`** — embed `cli.yaml`, call `lathe.NewApp` + `generated.MountModules`
