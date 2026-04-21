# lathe

> Turn any API spec into a polished CLI.

Feed lathe a Swagger / OpenAPI 2.0 document or a `.proto + google.api.http` file, and it gives you back a single `cobra` binary with per-operation subcommands, hostname-keyed auth, `--file` / `--set` request bodies, and table / JSON / YAML output. No hand-written command code.

> **Status**: v0.x, pre-release. Shape is stable (two backends, one IR, hostname-keyed auth), but API details may still shift before v0.1.

## Why lathe

Every service with an API eventually wants a CLI. Teams burn weeks writing cobra commands that 1:1 mirror the existing Swagger or protobuf spec. If the spec is the source of truth, the CLI should derive from it — not duplicate it.

lathe treats the spec as input and the CLI as output. Your job is to pin a version and describe the branding. lathe handles parsing, command tree synthesis, flag marshaling, HTTP transport, and output formatting.

## What you get

- **Two native backends**: Swagger 2.0 and `.proto + google.api.http`. No cross-transcoding; each spec is consumed in its real form.
- **One IR, one runtime**: both backends converge to `runtime.CommandSpec`. Adding a backend doesn't touch execution.
- **Hostname-keyed auth**, modeled on `gh`: per-host `hosts.yml`, no "current context" ambient state.
- **Body builder**: `--file -` for stdin JSON, `--set spec.replicas=3` for inline patches.
- **Output formats**: `-o table|json|yaml|raw`. Table mode auto-picks columns from the response shape.
- **Overlay layer**: hand-written YAML per module overrides help text / aliases without touching generated code.

## Quick start

Click **"Use this template"** on github.com/samzong/lathe to get your own copy, then fill in the four downstream-specific pieces:

1. `cli.yaml` at repo root — your CLI's identity (name, short, config dir, auth endpoint). See `internal/config/manifest.go` for the full schema.
2. `embed.go` at repo root — `//go:embed cli.yaml` into a `ManifestBytes []byte`.
3. `specs/sources.yaml` — pin the upstream repos that define your API (see "Defining sources" below).
4. `cmd/<your-cli-name>/main.go` — a ~20-line main that loads the manifest, binds it, and wires `internal/generated.MountModules` onto `lathe.NewApp` (from `github.com/samzong/lathe/pkg/lathe`).

Then:

```sh
make sync-specs && make gen     # pull upstream specs at pinned tags + generate code
go build -o bin/<name> ./cmd/<name>
```

Full scaffolding docs + a runnable example live under `examples/` (coming in a follow-up).

## Defining sources

`specs/sources.yaml` is the single source of truth for which modules your CLI exposes. Each entry pins an upstream repo at a specific tag.

```yaml
sources:
  # Swagger 2.0 — one or more files merged into one module
  iam:
    repo_url: https://github.com/acme/iam.git
    pinned_tag: v1.4.0
    backend: swagger
    swagger:
      files:
        - api/openapi/user.swagger.json
        - api/openapi/role.swagger.json

  # proto + google.api.http — staged into a protoc include root
  billing:
    repo_url: https://github.com/acme/billing.git
    pinned_tag: v0.9.2
    backend: proto
    proto:
      staging:
        - from: api/proto
          to: "."
      entries:
        - v1/accounts.proto
        - v1/invoices.proto
```

Rules:

- **Pinned tags are mandatory.** Floating branches break binary reproducibility.
- **Swagger**: multiple files merge on `definitions` + `paths`; duplicates warn with first-seen wins.
- **Proto**: only rpc methods annotated with `google.api.http` become commands. Message-only files come in via `import` transitively.
- **Grouping**: swagger uses `tags[0]`; proto uses the `service` name.

Full architecture docs are TBD; for now the code in `internal/runtime/` + `internal/codegen/` is the source of truth.

## Design principles

1. **Spec is truth. Code is derived.** Before hand-writing a command, ask why the spec doesn't already say it.
2. **Mechanical first, overlay second.** Layer 1 is a verbatim spec mapping; hand-polish only where reality beats the spec.
3. **No hidden state.** Hosts are keyed by hostname. No ambient "current context".
4. **Two backends, one IR.** Adding AsyncAPI or RAML later means a new backend — the runtime stays the same.

## License

[MIT](LICENSE) © samzong
