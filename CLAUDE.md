# CLAUDE.md

## Project intent

Lathe generates agent-friendly Cobra CLIs from Swagger 2.0, OpenAPI 3, and protobuf API specs. The API spec and project configuration are the source of truth; generated code should stay reproducible from pinned inputs.

## Product positioning

Lathe is a spec-to-agent-toolchain generator. It turns pinned API specs plus repo-local configuration into one inspectable Cobra CLI that both humans and AI agents can run safely.

The core product promise is: agents should not guess command names, flags, auth state, request body shape, HTTP path, or output format. Generated CLIs must expose machine-readable contracts (`search`, `commands --json`, `commands show`, `commands schema`) and generated Skill guidance so agents can discover, inspect, verify auth, and then execute.

Lathe is not a generic CLI framework, plugin loader, GUI/TUI, API gateway, or hand-written SDK replacement. Product work should strengthen spec fidelity, reproducibility, generated command correctness, runtime catalog inspectability, auth/body/output behavior, and generated Skill quality. Challenge features that move product weight into manually authored commands, runtime plugins, or non-spec-driven behavior.

## Source of truth

- Follow this file for repository structure, commands, style, tests, commits, PR rules, and Claude Code workflow.
- Follow `CONTRIBUTING.md` for contributor workflow and public contribution scope.
- Trust code and current Makefile targets over stale documentation when they disagree.
- Do not commit generated output under `internal/generated/`, upstream clones under `.cache/`, example build artifacts, or ad-hoc generated `skills/<cli-name>/` directories.

## Project structure

- `cmd/lathe` contains the executable entry point for spec syncing, code generation, and bootstrap.
- `internal/lathecmd`, `internal/sourceconfig`, `internal/specsync`, `internal/codegen`, `internal/overlay`, and `internal/auth` hold implementation-only packages.
- `pkg/runtime`, `pkg/config`, and `pkg/lathe` are downstream-facing runtime/library surfaces for generated CLIs.
- Tests live beside implementation as `*_test.go`; golden fixtures live under package-local `testdata/`.
- `examples/` contains runnable examples, and `docs/` contains architecture material and images.

## Architecture invariants

- Lathe has two phases: codegen-time (`cmd/lathe`, `internal/lathecmd`, `internal/sourceconfig`, `internal/specsync`, `internal/codegen/**`, `internal/overlay`) and runtime (`pkg/config`, `pkg/runtime`, `pkg/lathe`, `internal/auth`, plus generated modules).
- The seam is `internal/generated/<module>/<module>_gen.go`: generated `[]runtime.CommandSpec` literals compiled into the downstream CLI.
- `pkg/runtime` must remain independent of `internal/codegen/**`; runtime behavior cannot depend on raw specs, overlays, or sync cache state.
- Overlays are codegen-time polish only. They are merged into `CommandSpec`; the runtime must not learn overlay concepts.
- `specs/sources.yaml`, `cli.yaml`, pinned upstream refs, and optional overlays are the durable inputs. Generated files are outputs, not hand-edited source.

## Development workflow

- Keep changes small and focused; avoid speculative abstractions.
- Prefer configuration or overlays (`cli.yaml`, `specs/sources.yaml`, overlay config) over hard-coded generated behavior.
- Preserve package boundaries: `internal/**` is implementation-only, `pkg/**` is downstream-facing API.
- Use standard Go formatting through `gofmt` / `go fmt`.
- Wrap errors with context using `fmt.Errorf("...: %w", err)`.
- For codegen, normalization, or runtime behavior changes, add package-local tests and golden fixtures when they make behavior clearer.
- For CLI-visible behavior changes, update docs or examples only when the user-facing output or workflow actually changes.
- If a change alters generated command shape, catalog JSON, auth flow, body building, output formatting, retry/debug behavior, or Skill rendering, treat it as product behavior and prove it with focused tests or an example run.

## Commands

- `make help` lists available targets.
- `make bootstrap` runs `lathe bootstrap` for first-time generated CLI setup in a repo that has `cli.yaml` and `specs/sources.yaml`.
- `make sync-specs` fetches specs declared in `specs/sources.yaml`; cache root is `-cache`, `$LATHE_SPECS_CACHE`, or `.cache`.
- `make gen` regenerates `internal/generated` and, by default, `skills/<cli-name>/` from cached specs. It requires valid generated-CLI inputs; do not assume it is the right gate for unrelated core-package edits.
- `make check` is the full local quality gate: format check, `go vet`, `golangci-lint`, and tests.
- `make test` runs `go test ./...`.
- `make fmt`, `make fmt-check`, `make vet`, `make lint`, and `make tidy` run focused maintenance tasks.
- `make example-petstore` or `bash examples/richapi/run.sh` exercise end-to-end examples.
- Prefer the narrowest Make target that proves the changed surface.
- Use `make check` before commit or PR unless the change is documentation-only and the user agrees to skip it.

## Verification rules

- Before claiming completion, run the narrowest command that proves the changed surface.
- For runtime-sensitive changes, also run the relevant example script or `go test -race ./...`.
- For codegen changes, verify regenerated output behavior, but do not commit generated output under `internal/generated/`.
- CI runs `go build ./...`, `go vet ./...`, `golangci-lint`, and `go test -race ./...`; local proof should explain any narrower substitute.
- If a command cannot be run because a dependency is missing, report that directly instead of claiming success.

## Agent-facing contract

- The runtime catalog is the source of truth for generated CLI operation details. Generated Skill files are guidance and indexes, not execution authority.
- Preserve the agent loop: `search "<intent>" --json` for candidates, `commands show <path...> --json` for exact command detail, `auth status --hostname <host>` when `auth.required=true`, then execute with known flags/body/output.
- Search results are discovery only. Agents must not execute directly from search output.
- Prefer `-o json` for machine-readable command output unless the user asked for table, yaml, or raw output.
- Changes touching `pkg/runtime/catalog.go`, `pkg/lathe/catalog.go`, `pkg/runtime/build.go`, or `internal/codegen/render/skill.go` must consider catalog schema compatibility, generated Skill instructions, and agent inspectability.

## Security-sensitive surfaces

- HTTP runtime changes must account for SSRF, unsafe TLS, retry/debug logging, header handling, token leakage, and response error leakage.
- Host config and auth changes must protect persisted credentials and avoid printing secrets in normal, debug, or error output.
- Spec sync and codegen path handling must reject traversal, unsafe output roots, and accidental deletion of non-owned directories.
- Code templates must not emit shell-sensitive or injection-prone behavior from untrusted spec text.
- Treat upstream specs as trusted product inputs but still escape or validate anything that becomes Go code, CLI flags, help text, file paths, or generated Skill markdown.

## Git and release hygiene

- Check `git status --short` before edits that may overlap user work and before any commit workflow.
- Never overwrite user changes or run destructive git commands unless explicitly requested.
- Stage only intended files, not broad workspace state.
- Commit messages use Conventional Commits.
- Use `git commit -s` for every commit.
- Inspect staged changes before committing; commit only what is intentionally staged.
- Do not push, create PRs, or modify remote/shared state unless explicitly requested.
- Keep PRs focused and include the exact verification commands run.

## Helpful Claude Code skills

These apply only when running under Claude Code; they are not repository requirements.

- Use `/systematic-debugging` for bugs, test failures, or unexpected behavior before guessing at fixes.
- Use `/pre-ship` before pushing when local reviewer-style scrutiny is useful.
- Use `/commit` for commit-only workflows after the intended changes are ready.
- Use `/ship` only when the user explicitly wants commit, push, and PR creation.
- Use `/sync-docs` after changes that may make README, examples, or project docs stale.
- Use `/critical-bug-finder` for explicit crash, data loss, security, race, or production reliability audits.

## Local instructions

- If `CLAUDE.local.md` exists, read it before starting non-trivial work; some non-Claude-Code agents will not load it automatically, but its instructions are still required.
