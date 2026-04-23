# Architecture

This document describes how lathe turns an API spec into a CLI, the packages involved, and the workflow for adding a new module. For user-facing usage, see [../README.md](../README.md).

## Prime idea

> The spec is input. The CLI is output. Humans curate the edges; code fills the middle.

Everything in lathe is organized around a single invariant: **spec is the source of truth, code is derived**. The consequences ripple through every package — no runtime awareness of the original backend, no hand-written commands for operations already described by the spec, no floating tags.

## Two-phase model

lathe has two disjoint phases. They share types (`runtime.CommandSpec`) but run at different times and in different binaries.

```mermaid
flowchart LR
    subgraph codegen["Codegen-time (in the template repo)"]
        direction TB
        S1[specs/sources.yaml] --> S2[make sync-specs]
        S2 --> S3[make gen]
        S3 --> S4[internal/generated/*]
    end

    subgraph runtime["Runtime (in the end-user's shell)"]
        direction TB
        R1[go build] --> R2[single binary]
        R2 --> R3[cobra command tree]
        R3 --> R4[HTTP request]
        R4 --> R5[formatted output]
    end

    S4 -.compiled into.-> R1

    style codegen fill:#eef2ff,stroke:#4338ca
    style runtime fill:#ecfdf5,stroke:#047857
```

The seam between them is `internal/generated/<module>/<module>_gen.go` — a single file per module containing a `[]runtime.CommandSpec` literal. Everything above the seam is a build concern; everything below it is a user concern. The runtime has no idea whether a command came from Swagger or from proto.

## Codegen pipeline

```mermaid
flowchart TD
    A[specs/sources.yaml] -->|sourceconfig.Load| B[Config]
    B --> C{backend?}

    C -->|swagger| D1[specsync.syncSwagger]
    C -->|openapi3| D3[specsync.syncOpenAPI3]
    C -->|proto| D2[specsync.syncProto]

    D1 --> E1[.cache/specs-sync/mod/*.swagger.json]
    D3 --> E3[.cache/specs-sync/mod/*.yaml / *.json]
    D2 --> E2[.cache/specs-sync/mod/*.proto + deps]

    E1 -->|swagger.Parse| F[rawir.RawModule]
    E3 -->|openapi3.Parse| F
    E2 -->|proto.Parse| F

    F -->|normalize.Normalize| G["[]runtime.CommandSpec"]
    H[internal/overlay/&lt;mod&gt;.yaml] -->|overlay.LoadDir| I[Overrides]
    I -->|merge| G

    G -->|render.RenderModule| J[internal/generated/mod/mod_gen.go]

    style F fill:#fef3c7,stroke:#b45309
    style G fill:#fef3c7,stroke:#b45309
    style J fill:#dcfce7,stroke:#16a34a
```

Three backends fan in to a single raw IR (`rawir.RawModule`). `normalize` is the only place that understands the IR's semantics and projects it onto the runtime's `CommandSpec` shape. `render` is a pure `text/template` emit — if the `CommandSpec` shape is wrong, `render` cannot fix it.

### Raw IR vs runtime spec

Two IRs exist on purpose. `rawir` preserves backend-adjacent detail (schemas, refs, per-response shape) needed for **normalization decisions** (list-path detection, column picking). `runtime.CommandSpec` is the minimal declarative form the runner needs. The boundary between them is enforced by the package graph: nothing under `pkg/runtime` imports `internal/codegen/**`.

### Why three backends, one IR

| Concern | Swagger backend | OpenAPI 3 backend | Proto backend |
|---|---|---|---|
| Grouping | operation's first `tag` | operation's first `tag` | `service` name |
| Operation ID | `operationId` | `operationId` | `rpc` name |
| Path / method | operation object | operation object | `google.api.http` annotation |
| Body schema | `requestBody` | `requestBody` (with `$ref` rewrite) | input message |
| Response schema | first 2xx response | first 2xx response | output message |

All of the above are normalized into the same `RawOperation` fields. By the time a spec reaches `normalize.Normalize`, the origin is irrelevant.

## Package layout

```mermaid
graph TD
    subgraph cmd["cmd/ — binaries"]
        C1[cmd/specsync]
        C2[cmd/codegen]
    end

    subgraph internal["internal/ — codegen-only"]
        I1[specsync]
        I2[sourceconfig]
        I3[codegen/backends/swagger]
        I3b[codegen/backends/openapi3]
        I4[codegen/backends/proto]
        I5[codegen/normalize]
        I6[codegen/rawir]
        I7[codegen/render]
        I8[overlay]
    end

    subgraph pkg["pkg/ — library (compiled into downstream CLI)"]
        P1[pkg/config]
        P2[pkg/runtime]
        P3[pkg/lathe]
        P4[internal/auth]
    end

    C1 --> I1
    C1 --> I2
    C2 --> I1
    C2 --> I2
    C2 --> I3
    C2 --> I3b
    C2 --> I4
    C2 --> I5
    C2 --> I7
    C2 --> I8

    I3 --> I6
    I3b --> I6
    I4 --> I6
    I5 --> I6
    I5 --> P2
    I7 --> P2
    I7 --> I8

    P3 --> P1
    P3 --> P2
    P3 --> P4
    P4 --> P1

    style pkg fill:#ecfdf5,stroke:#047857
    style internal fill:#eef2ff,stroke:#4338ca
    style cmd fill:#fef3c7,stroke:#b45309
```

### Responsibilities

| Package | Phase | Responsibility |
|---|---|---|
| `cmd/specsync` | codegen | Thin wrapper over `internal/specsync`. Resolves cache root, runs sync. |
| `cmd/codegen` | codegen | Orchestrates: load sources → verify sync state → parse → normalize → render. |
| `internal/sourceconfig` | codegen | Parse `specs/sources.yaml`. Requires `pinned_tag` to be set; treats the value as an immutable ref for reproducibility. |
| `internal/specsync` | codegen | `git clone --filter=blob:none` into `.cache/specs-work/<module>/`, then `git checkout` the pinned ref and stage the relevant files into `.cache/specs-sync/<module>/`. Writes `sync-state.yaml` (including `resolved_sha`) consumed by codegen to detect stale caches. |
| `internal/codegen/backends/swagger` | codegen | Parse `*.swagger.json` → `RawModule`. Merges multiple files; first-seen wins on duplicate operation IDs. |
| `internal/codegen/backends/openapi3` | codegen | Parse OpenAPI 3.x YAML/JSON → `RawModule`. Rewrites `#/components/schemas/` refs to rawir format; inherits path-level parameters. |
| `internal/codegen/backends/proto` | codegen | Parse staged `.proto` tree → `RawModule`. Only RPCs with a `google.api.http` binding become operations. |
| `internal/codegen/rawir` | codegen | Backend-agnostic raw types (`RawModule`, `RawOperation`, `RawSchema`). Includes `$ref` resolution. |
| `internal/codegen/normalize` | codegen | The semantic projection. Groups, picks `Short`, derives list path, picks default columns, enforces method-ordering for determinism. |
| `internal/codegen/render` | codegen | `text/template` → gofmt'd Go. Emits `internal/generated/<mod>/<mod>_gen.go` and the top-level `modules_gen.go` index. |
| `internal/overlay` | codegen | Load `internal/overlay/<module>.yaml`. Results are passed to `render.RenderModule`, baked into the emitted `CommandSpec` literal. Runtime never sees overlays. |
| `internal/auth` | runtime | `auth login/logout/status`. Uses `manifest.AuthInfo.Validate` to call the configured endpoint and display the identified principal. |
| `pkg/config` | runtime | `Manifest` (CLI identity) and `Hosts` (per-hostname credentials). `Bind(m)` seeds package-level helpers with the active manifest. |
| `pkg/runtime` | runtime | `CommandSpec` IR, the `Build` function that materializes cobra commands from specs, body builder (`--set`, `--file`), HTTP client with retry transport, `Authenticator` interface, `Formatter` registry, typed `LatheError` with stable exit codes, schema version contract. |
| `pkg/lathe` | runtime | `NewApp(m)` — returns the root cobra command with auth subtree and module groups attached. The downstream `main.go` is ~15 lines. |

## Spec lifecycle

From authoring a `sources.yaml` entry to a command the user can run:

```mermaid
stateDiagram-v2
    [*] --> Declared: edit specs/sources.yaml
    Declared --> Cloned: make sync-specs\n(git clone + checkout pinned_tag)
    Cloned --> Staged: backend-specific staging\n(.cache/specs-sync/&lt;mod&gt;/)
    Staged --> Parsed: backend.Parse\n→ RawModule
    Parsed --> Normalized: normalize.Normalize\n→ []CommandSpec
    Normalized --> Polished: overlay merge\n(optional)
    Polished --> Emitted: render.RenderModule\n→ internal/generated/&lt;mod&gt;/&lt;mod&gt;_gen.go
    Emitted --> Compiled: go build
    Compiled --> Runnable: ./bin/&lt;name&gt; &lt;mod&gt; &lt;cmd&gt;
    Runnable --> [*]

    Declared --> Declared: bump pinned_tag\n→ restart cycle
```

Each transition is idempotent and cache-checked. `specsync.VerifyState` rejects a stale cache where the on-disk `sync-state.yaml` does not match the requested `pinned_tag`, so `make gen` alone never silently uses a drifted spec.

## Adding a new module

The canonical flow for adding an upstream API to your CLI:

```mermaid
flowchart TD
    A[Pick upstream repo + immutable tag] --> B[Identify spec form:\nSwagger 2.0 or .proto]
    B --> C[Edit specs/sources.yaml]
    C --> D{Swagger?}

    D -->|yes| E1[Add backend: swagger\nlist spec files under swagger.files]
    D -->|no| E2[Add backend: proto\ndeclare proto.staging\nlist proto.entries]

    E1 --> F[make sync-specs]
    E2 --> F

    F --> G[make gen]
    G --> H{generated code looks right?}

    H -->|missing cmds| I1[Check operationId / google.api.http\npresent upstream?]
    H -->|weak help text| I2[Add internal/overlay/&lt;mod&gt;.yaml]
    H -->|yes| J[go build]

    I1 --> F
    I2 --> G

    J --> K[./bin/&lt;name&gt; &lt;mod&gt; --help]
    K --> L[commit:\nspecs/sources.yaml,\ninternal/overlay/&lt;mod&gt;.yaml,\ninternal/generated/&lt;mod&gt;/&lt;mod&gt;_gen.go]

    style C fill:#fef3c7,stroke:#b45309
    style I2 fill:#fef3c7,stroke:#b45309
    style L fill:#dcfce7,stroke:#16a34a
```

The two manual surfaces are `specs/sources.yaml` (mandatory) and `internal/overlay/<mod>.yaml` (optional). Everything else is mechanical.

### Swagger vs proto at sync time

```mermaid
sequenceDiagram
    participant User
    participant Make as make sync-specs
    participant Git
    participant Swagger as swagger backend
    participant Proto as proto backend
    participant Cache as .cache/specs-sync/&lt;mod&gt;/

    User->>Make: invoke
    Make->>Git: clone --filter=blob:none + checkout pinned_tag
    Git-->>Make: .cache/specs-work/&lt;mod&gt;/

    alt backend: swagger
        Make->>Swagger: syncSwagger(src, workDir, syncDir)
        Swagger->>Cache: copy declared swagger.files verbatim
    else backend: proto
        Make->>Proto: syncProto(src, workDir, syncDir)
        Proto->>Cache: stage proto.staging layers
        Proto->>Cache: ensure proto.entries are resolvable
    end

    Make->>Cache: write sync-state.yaml\n(source, backend, synced_from)
```

Sync is a pure file-staging step. It never parses semantics. Parsing happens in `make gen`, which means a broken spec fails codegen, not sync — and the cache on disk is always a faithful copy of the upstream tag.

### Overlay bake

Overlays are a codegen-time concern only. They never reach runtime.

```mermaid
flowchart LR
    O[internal/overlay/&lt;mod&gt;.yaml] -->|overlay.LoadDir| M[Overrides map\nkeyed by command Use]
    S["[]CommandSpec\n(from normalize)"] --> R[render.RenderModule]
    M --> R
    R --> G["internal/generated/&lt;mod&gt;/&lt;mod&gt;_gen.go\n(Short, Long, Example, Aliases baked in)"]

    G -.compiled.-> B[CLI binary]

    style O fill:#fef3c7,stroke:#b45309
    style G fill:#dcfce7,stroke:#16a34a
```

`Short`, `Long`, `Example` are replaced if non-empty. `Aliases` append (not replace), so overlay-added aliases sit alongside any the spec already implied. Runtime never reads overlay files; an empty or missing overlay dir is a pass-through.

## Runtime request lifecycle

What happens when a user runs `./bin/acmectl iam create-user --email alice@example.com`:

```mermaid
sequenceDiagram
    autonumber
    participant Shell
    participant Root as cobra root
    participant Runner as runtime.Build
    participant Host as hosts.yml / $NAME_HOST
    participant Body as body builder
    participant HTTP
    participant Out as output formatter

    Shell->>Root: acmectl iam create-user --email alice@example.com
    Root->>Runner: dispatched to generated module cmd
    Runner->>Host: ResolveHost(--hostname, $NAME_HOST, hosts.yml)
    alt exactly one host && no flag
        Host-->>Runner: auto-selected host
    else multiple hosts, no flag, no env
        Host-->>Runner: ErrAmbiguousHost
    else --hostname or env
        Host-->>Runner: selected host + credentials
    end

    Runner->>Body: assemble body (--file, --set dotted.paths)
    Body-->>Runner: JSON payload

    Runner->>HTTP: method + PathTpl(params) + body + auth header
    HTTP-->>Runner: response

    Runner->>Out: format(-o table|json|yaml|raw, OutputHints)
    Out-->>Shell: stdout
```

Three pieces of state cross the boundary:

1. **Manifest** (immutable, from `cli.yaml` embedded at build time) — defines the CLI's identity and auth shape.
2. **Hosts** (mutable, `~/.config/<name>/hosts.yml`) — per-hostname credentials. No "current host" is stored; the host is always resolved at invocation.
3. **Flags** (transient, per-invocation) — `--hostname`, `--output`, `--insecure`, plus operation-specific flags.

`config.Bind(m)` is called once in `main.go` so that `pkg/runtime` helpers can reach the active manifest without a parameter-passing chain.

## Design invariants

These are structural, not stylistic. Violating any of them means the architecture breaks.

1. **`pkg/runtime` does not import `internal/codegen/**`.** The runtime cannot know how a `CommandSpec` was produced. This is what makes "two backends, one IR" real rather than aspirational.
2. **`pinned_tag` is required and validated.** `sourceconfig.Load` rejects empty values and floating refs (`HEAD`, `main`, `refs/heads/*`). Only immutable tags and 40-char SHAs are accepted. `specsync` records the resolved commit SHA and codegen verifies it.
3. **Codegen is never invoked at `go build` time.** Downstream consumers of a lathe-generated CLI do not need Go toolchain tags, build flags, or network access to install it.
4. **Overlays bake at codegen-time.** The runtime has no overlay concept. This keeps `pkg/runtime` small and keeps overlay bugs from being runtime bugs.
5. **No ambient "current host".** The host is a per-invocation input. This mirrors `gh` and avoids the classic "oops, wrong cluster" class of bug.
6. **`sync-state.yaml` guards the cache.** `make gen` refuses a cache that doesn't match the requested `pinned_tag`. Stale generation fails loud, not silent.

## Where to look next

- **Using the CLI** — [../README.md](../README.md)
- **Contributing** — [../CONTRIBUTING.md](../CONTRIBUTING.md)
- **CLI identity shape** — [../pkg/config/manifest.go](../pkg/config/manifest.go)
- **Runtime IR** — [../pkg/runtime/spec.go](../pkg/runtime/spec.go)
- **Raw IR** — [../internal/codegen/rawir/types.go](../internal/codegen/rawir/types.go)
- **Example overlay** — [../examples/overlay/example.yaml](../examples/overlay/example.yaml)
