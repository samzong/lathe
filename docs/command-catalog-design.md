# Command Catalog Design

## Problem

Generated CLIs need a stable way for humans and agents to discover how to use
them. Plain `--help` output is good for humans, but it is weak as an agent
protocol:

- it is prose, not structured data;
- it loses generated metadata such as operation IDs, HTTP methods, paths,
  parameter locations, auth scopes, and output hints;
- search quality becomes fragile if agents must infer command shape from text;
- generated reference docs can drift from the binary they describe.

lathe already has the right source of truth: every operation is compiled into a
`runtime.CommandSpec`. The catalog should expose that structured command model
at runtime instead of making agents scrape help text or rely on stale docs.

## Goals

1. Give every lathe-generated CLI a machine-readable command catalog.
2. Make command discovery deterministic enough for agents.
3. Keep the protocol generic across all lathe-generated CLIs.
4. Let search and future SKILL/manual generation reuse the same catalog data.
5. Avoid extra generated files, local indexes, network calls, or runtime spec
   parsers.

## Non-goals

- No interactive TUI.
- No fuzzy-search dependency in the first version.
- No persistent local index.
- No attempt to make arbitrary non-lathe CLIs introspectable.
- No replacement for Cobra help output; help remains the human-facing local
  explanation for a command.

## Recommendation

Add a root-level `commands` command to every lathe application:

```sh
acmectl commands --json
acmectl commands show iam users create-user --json
acmectl commands schema
```

Then add search as a consumer of the same data:

```sh
acmectl search users --json
acmectl search "create user"
```

The catalog is the exact source of truth. Search is only candidate retrieval.
An agent should use search to narrow choices, then use `commands show --json` to
retrieve exact flags and execution requirements before running a command.

## Agent Workflow

Agents should follow this order:

1. Run `<cli> commands --json` when entering an unfamiliar CLI.
2. If the desired command is obvious, select by exact command path or operation
   ID from the catalog.
3. If the desired command is ambiguous, run `<cli> search --json "<intent>"`.
4. For the selected candidate, run `<cli> commands show <path...> --json`.
5. Build the final command from structured flags, required fields, auth hints,
   body requirements, and output hints.
6. Do not infer flags or request shape from README prose.

Future SKILL files should teach this workflow instead of embedding a large,
easy-to-stale command list.

## Command Surface

### `commands`

Lists the command catalog. Default output is a compact human table. JSON output
is the stable agent protocol.

```sh
acmectl commands
acmectl commands --json
acmectl commands --include-hidden --json
```

Flags:

| Flag | Default | Meaning |
|---|---:|---|
| `--json` | `false` | Emit the versioned catalog JSON. |
| `--include-hidden` | `false` | Include hidden commands and flags. |

### `commands show`

Returns one command by exact command path.

```sh
acmectl commands show iam users create-user
acmectl commands show iam users create-user --json
```

The path is the same path a user would type after the root command, excluding
flags. The command should return a not-found error if the path does not resolve
to a generated operation command.

The path excludes the root binary name. It uses the actual Cobra command names
the user types, so the group segment is lowercased to match `runtime.Build`.
For example, `["iam", "users", "create-user"]` corresponds to
`acmectl iam users create-user`.

### `commands schema`

Prints the catalog schema version supported by the binary.

```sh
acmectl commands schema
acmectl commands schema --json
```

This command is intentionally narrow. JSON output should be a small object such
as `{"catalog_schema_version":1}`. The field contract lives in this design doc
and in implementation tests; do not pretend to ship a full JSON Schema until the
project actually needs one.

### `search`

Searches over the same catalog entries.

```sh
acmectl search user
acmectl search "create user" --json
acmectl search /api/v1/users --limit 50 --json
```

Flags:

| Flag | Default | Meaning |
|---|---:|---|
| `--json` | `false` | Emit ranked search results as JSON. |
| `--limit` | `20` | Maximum number of results. |

Search must never be the precision layer. It should return candidates with
scores and full command paths; exact usage still comes from `commands show`.
Search excludes hidden commands. Agents that need a complete inventory should
use `commands --include-hidden --json` instead of search.

## Catalog Schema

The catalog JSON is versioned independently from `runtime.SchemaVersion`.
`runtime.SchemaVersion` protects generated Go compatibility. The command
catalog schema protects external tools and agents.

Top-level shape:

```json
{
  "catalog_schema_version": 1,
  "cli": {
    "name": "acmectl",
    "version": "v1.2.3"
  },
  "output": {
    "default_format": "table",
    "formats": ["table", "json", "yaml", "raw"]
  },
  "commands": []
}
```

Command entry shape:

```json
{
  "path": ["iam", "users", "create-user"],
  "service": "iam",
  "group": "Users",
  "use": "create-user",
  "aliases": ["add-user"],
  "summary": "Create a user",
  "description": "Create a user in the IAM service.",
  "example": "acmectl iam users create-user --email alice@example.com",
  "operation_id": "createUser",
  "http": {
    "method": "POST",
    "path_template": "/api/v1/users"
  },
  "auth": {
    "required": true,
    "scopes": ["users:write"]
  },
  "body": {
    "required": true,
    "media_type": "application/json"
  },
  "flags": [
    {
      "name": "workspace",
      "flag": "workspace",
      "location": "query",
      "type": "string",
      "required": true,
      "default": "",
      "enum": [],
      "format": "",
      "deprecated": false,
      "help": "Target workspace."
    }
  ],
  "output": {
    "list_path": "data.items",
    "default_columns": ["id", "email", "role"],
    "response_media_type": "application/json",
    "pagination": {
      "strategy": "cursor",
      "token_param": "page_token",
      "token_field": "next_page_token",
      "limit_param": "limit"
    },
    "streaming": {
      "strategy": "sse"
    }
  },
  "hidden": false,
  "deprecated": false
}
```

Rules:

- Fields are additive. Once published, existing JSON field names and meanings do
  not change within a schema version.
- Empty optional values may be omitted when doing so keeps output smaller and
  unambiguous.
- Hidden commands and flags are excluded by default.
- Deprecated commands and flags remain visible by default.
- Command paths are canonical and stable for the generated binary.
- Command paths exclude the root binary name. The example path
  `["iam", "users", "create-user"]` corresponds to
  `acmectl iam users create-user`.
- Command path segments use the Cobra command names users type. Service and
  operation segments come from generated command names. The group path segment
  is lowercased because `runtime.Build` mounts groups as
  `strings.ToLower(CommandSpec.Group)`.
- The JSON `group` field is the original `CommandSpec.Group` value, not the
  lowercased parent Cobra command name.
- `auth.required` describes runtime behavior. It is `false` only when
  `CommandSpec.Security.Public` is true; otherwise generated operation commands
  require a configured host/auth entry. `auth.scopes` mirrors
  `CommandSpec.Security.Scopes` when present.
- Top-level `output.default_format` and `output.formats` describe the root
  formatter flags for the binary. Per-command `output` only contains operation
  output hints from `CommandSpec.Output`.

## Data Source

The source of truth is the mounted Cobra tree enriched with catalog payloads
derived from `runtime.CommandSpec`.

`runtime.Build(root, service, specs)` already sees every generated operation.
It should attach structured metadata to each operation command when building the
tree. The catalog builder should then walk the Cobra tree and collect generated
operation commands only.

The catalog must read operation metadata from the command annotation, not from
parent Cobra commands. This matters because group container commands are mounted
with lowercased `Use` values for CLI ergonomics, while the annotated payload
preserves the original `CommandSpec.Group` value for structured output.

This avoids:

- parsing help text;
- implicit package-level registries;
- importing `internal/codegen` from runtime packages;
- writing extra generated catalog files.

## Implementation Shape

### Runtime package

Add catalog types and builders in `pkg/runtime`:

```go
const CatalogSchemaVersion = 1
const DefaultSearchLimit = 20

type Catalog struct {
    CatalogSchemaVersion int
    CLI                  CatalogCLI
    Output               CatalogOutputFormats
    Commands             []CatalogCommand
}

type CatalogCommand struct {
    Path        []string
    Service     string
    Group       string
    Use         string
    Aliases     []string
    Summary     string
    Description string
    Example     string
    OperationID string
    HTTP        CatalogHTTP
    Auth        CatalogAuth
    Body        *CatalogBody
    Flags       []CatalogFlag
    Output      CatalogOutput
    Hidden      bool
    Deprecated  bool
}
```

Add helpers:

```go
func AttachCatalogCommand(cmd *cobra.Command, service string, spec CommandSpec)
func BuildCatalog(root *cobra.Command, opts CatalogOptions) Catalog
func FindCatalogCommand(root *cobra.Command, path []string, opts CatalogOptions) (CatalogCommand, bool)
func SearchCatalog(root *cobra.Command, query string, opts SearchOptions) []SearchResult
```

`AttachCatalogCommand` builds a `CatalogCommand` payload from the operation
`CommandSpec`, serializes it as JSON, and stores it in
`cmd.Annotations["lathe.catalog.command"]`. The stored payload does not include
`Path`; path is injected later while walking the mounted Cobra tree.

- `runtime.Build` calls `AttachCatalogCommand` only for generated operation
  commands, never service or group containers.
- `BuildCatalog`, `FindCatalogCommand`, and `SearchCatalog` only emit commands
  that have the catalog annotation. Core commands such as `auth`, `version`,
  `commands`, and `search` are excluded by construction.
- annotation state belongs to the command tree. Multiple roots in the same
  process are isolated, and discarded roots can be garbage-collected.
- `FindCatalogCommand` walks the command tree by path, including Cobra aliases,
  and reads the annotation only after all path segments resolve.

Do not introduce package-level mutable registries for catalog state.

### Lathe package

Add root commands in `pkg/lathe`:

```go
func commandsCmd(m *config.Manifest) *cobra.Command
func searchCmd(m *config.Manifest) *cobra.Command
```

`NewApp` mounts both commands. They evaluate the root command at execution
time, so downstream generated modules may still be mounted after `NewApp`
returns.

### Sorting

Catalog output should be deterministic:

1. service path segment;
2. group path segment;
3. command use.

Search output should be deterministic:

1. score descending;
2. command path ascending.

## Search Scoring

Start with simple token matching. Lowercase the query and candidate fields.
Split query on whitespace. A command matches only if every query token appears
in at least one searchable field.

Searchable fields:

- command path;
- service;
- group;
- use;
- aliases;
- summary;
- description;
- operation ID;
- HTTP method;
- HTTP path template;
- flag names;
- flag help.

Suggested scoring:

| Match field | Score |
|---|---:|
| exact full path | 100 |
| exact operation ID | 90 |
| exact command use or alias | 80 |
| path segment prefix | 60 |
| HTTP path substring | 45 |
| summary or description substring | 30 |
| flag name exact match | 25 |
| flag help substring | 10 |

For each query token, take the highest score produced by any matching field.
The command's final score is the sum of token scores plus any exact full-query
bonus. This lets `users get` match across path and command fields without
double-counting every repeated substring.

This is intentionally boring. If search quality is not good enough later, add a
better ranking function without changing the catalog protocol.

## Text Output

Human output should stay compact:

```text
iam users create-user
  Create a user
  POST /api/v1/users

iam users get-user
  Get a user
  GET /api/v1/users/{id}
```

`commands show` text output can include flags:

```text
iam users create-user
  Create a user
  POST /api/v1/users

Flags:
  --workspace string  required  Target workspace.
  --role string                 User role.
```

JSON is the protocol. Text is convenience.

## Interaction With SKILL Generation

Future SKILL generation has two separate outputs.

The portable SKILL should teach the runtime workflow, not duplicate the full
catalog as prose:

```md
Use `<cli> commands --json` as the source of truth.
Use `<cli> search --json "<intent>"` for discovery.
Use `<cli> commands show <path...> --json` before executing unfamiliar commands.
Do not infer flags from README prose.
```

If a static reference document is needed, generate it from `commands --json`.
That document is only a snapshot of a specific binary, not a separate source of
truth.

## Compatibility

The first implementation is additive:

- existing generated commands continue to work;
- existing help output remains valid;
- downstream CLIs get the new commands through `pkg/lathe.NewApp`;
- no generated file format changes are required unless the implementation
  chooses to embed extra catalog-only metadata later.

If the catalog output needs new fields, add them under the same schema version
only when old consumers can ignore them. If field meanings must change, bump
`catalog_schema_version`.

## Verification Plan

Unit tests:

- `runtime.BuildCatalog` includes generated commands and excludes core commands.
- hidden commands are excluded by default and included with `--include-hidden`.
- required flags, enum values, defaults, body requirements, auth scopes, and
  output hints survive into catalog JSON.
- catalog paths exclude the root command, use lowercased group path segments,
  and keep the original `CommandSpec.Group` in the `group` JSON field.
- `FindCatalogCommand` resolves exact command paths and operation aliases.
- `SearchCatalog` matches by path, operation ID, HTTP path, summary, alias, and
  flag name.
- search ordering is stable.
- catalog JSON round-trips through `encoding/json` without losing fields.

Command tests:

- `commands --json` emits valid JSON with `catalog_schema_version`.
- `commands --json` emits `commands: []`, not `null`, when no generated
  operations are mounted.
- `commands show <path...> --json` emits one command.
- `commands show` fails for an unknown path.
- `search --json` emits ranked results with full paths.

Project verification:

```sh
make check
```

## Rollout

1. Add catalog annotation attachment in `runtime.Build`; only generated
   operation commands get catalog annotations.
2. Add catalog builder and JSON/text renderers.
3. Add `commands`, `commands show`, and `commands schema`.
4. Add catalog-backed `search`.
5. Add tests.
6. Update the lathe README with the agent workflow.
7. Use the catalog as the input for the future SKILL/manual generator.
