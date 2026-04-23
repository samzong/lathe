[English](README.md) | [中文](README_zh.md)

# lathe

> Turn any API spec into a polished CLI.

[![CI](https://github.com/samzong/lathe/actions/workflows/ci.yml/badge.svg)](https://github.com/samzong/lathe/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Feed lathe a Swagger 2.0, OpenAPI 3, or `.proto` file with `google.api.http` annotations, and it gives you back a single `cobra` binary with per-operation subcommands, hostname-keyed auth, flag-driven request bodies, and table / JSON / YAML output — no hand-written command code.

![lathe architecture](docs/images/architecture.png)

---

## Why

Every service with a published API ends up wanting a CLI. Teams routinely burn weeks writing commands that mirror an existing Swagger or protobuf spec 1:1 — a duplication that silently rots the moment the spec evolves.

If the spec is the source of truth, the CLI should be derived from it, not transcribed.

lathe treats the spec as input and the CLI as output. Your job shrinks to:

- pin the upstream spec at a tag,
- declare a few identity fields (CLI name, auth endpoint),
- optionally overlay help text where the spec's wording is weak.

lathe handles everything else.

---

## Features

- **Three native backends** — Swagger 2.0, OpenAPI 3, and `.proto` (with `google.api.http`). Each consumed in its real form; no cross-transcoding.
- **Reproducible** — every spec pinned at an immutable tag; floating branches rejected. Commit SHA recorded and verified.
- **Hostname-keyed auth** — per-host credentials modeled on `gh`. Public endpoints skip auth; scoped endpoints show required OAuth scopes.
- **Rich CLI** — body builder (`--file`, `--set`), `-o table|json|yaml|raw`, enum validation, cursor-based pagination (`--all`), SSE streaming, parameter defaults and deprecation warnings.
- **Overlay layer** — polish help text, aliases, and examples per-module without editing generated code.
- **Extensible** — `Authenticator` and `Formatter` interfaces for custom auth schemes and output formats.
- **Production-ready** — typed error model with stable exit codes (0–4), `--debug` HTTP tracing, schema version contract between generated code and runtime.

---

## Quick start

Click **"Use this template"** on [github.com/samzong/lathe](https://github.com/samzong/lathe), then populate two files and run `make`.

### `cli.yaml` — CLI identity

```yaml
cli:
  name: acmectl
  short: "Command-line tool for Acme services"

auth:
  validate:
    method: GET
    path: /api/v1/whoami
    display:
      username_field: data.username
      fallback_field: data.email
```

### `specs/sources.yaml` — pin upstream specs

```yaml
sources:
  iam:
    repo_url: https://github.com/acme/iam.git
    pinned_tag: v1.4.0
    backend: swagger
    swagger:
      files:
        - api/openapi/user.swagger.json

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

  payments:
    repo_url: https://github.com/acme/payments.git
    pinned_tag: v2.1.0
    backend: openapi3
    openapi3:
      files:
        - api/openapi.yaml
```

### `internal/overlay/<module>.yaml` — polish help text (optional)

```yaml
# internal/overlay/iam.yaml
commands:
  create-user:
    short: "Create a user in the IAM service"
    aliases: [adduser]
    example: |
      acmectl iam create-user \
        --email alice@example.com \
        --role viewer
    params:
      role:
        help: "User role (viewer, editor, admin)"
        default: viewer
```

Overlays are baked into generated code at codegen time — the runtime knows nothing about them. Pass `-overlay internal/overlay` to codegen.

### Build

```sh
make bootstrap          # sync-specs + gen
go build -o bin/acmectl ./cmd/acmectl

./bin/acmectl auth login --hostname acme.example.com
./bin/acmectl iam create-user --email alice@example.com --role viewer
```

Re-run `make bootstrap` whenever you bump a `pinned_tag`.

---

## Sources reference

`specs/sources.yaml` declares which upstream specs become modules in your CLI.

| Field | Required | Notes |
|---|---|---|
| `repo_url` | ✓ | Any URL `git clone` accepts |
| `pinned_tag` | ✓ | Floating branches rejected — reproducibility is mandatory |
| `backend` | ✓ | `swagger`, `openapi3`, or `proto` (exclusive) |
| `swagger.files` | swagger only | Multiple files merge; duplicates warn, first-seen wins |
| `openapi3.files` | openapi3 only | JSON or YAML; multiple files merge; `$ref` resolved within each file |
| `proto.staging` | proto only | Stage files into a protoc include root |
| `proto.entries` | proto only | Only RPCs with `google.api.http` become commands |

Grouping into subcommand trees:

- **Swagger / OpenAPI 3** — uses the operation's first `tag`.
- **Proto** — uses the `service` name.

---

## Configuration

| Env var | Effect |
|---|---|
| `$<NAME>_HOST` | Select host without editing `hosts.yml` |
| `$<NAME>_CONFIG_DIR` | Override config dir (default `~/.config/<name>`) |
| `LATHE_SPECS_CACHE` | Where `make sync-specs` stages specs (default `.cache`) |

`<NAME>` is the uppercased `cli.name`.

### Global flags

| Flag | Effect |
|---|---|
| `--hostname` | Select host for this invocation |
| `-o, --output` | Output format: `table\|json\|yaml\|raw` |
| `--insecure` | Skip TLS certificate verification |
| `--debug` | Print HTTP request/response to stderr |

---

## Design principles

1. **Spec is truth. Code is derived.** Before hand-writing a command, ask why the spec doesn't already say it.
2. **Mechanical first, overlay second.** Layer 1 is a verbatim mapping; polish only where reality beats the spec.
3. **No hidden state.** Hosts keyed by hostname. No ambient "current context", no implicit default.
4. **Multi-backend, one IR.** The runtime does not know whether a command came from Swagger, OpenAPI 3, or proto.

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). All commits must be signed off (`git commit -s`) per [DCO](https://developercertificate.org/).

## Security

See [SECURITY.md](SECURITY.md) for private vulnerability disclosure.

## License

[MIT](LICENSE) © samzong
