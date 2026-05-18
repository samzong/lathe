# Showcase

This page collects runnable Lathe examples and public adoption notes.

## 60-Second Demo

Run the Petstore demo from a source checkout:

```sh
bash examples/petstore/run.sh
```

The script builds a generated CLI from a small OpenAPI 3 spec, then demonstrates the agent loop:

1. `petstore search "list pets" --json`
2. `petstore commands show pets pets list --json`
3. `petstore commands schema --json`

This is the core Lathe promise: agents discover commands, inspect exact contracts, check schema compatibility, and only then execute.

## Examples

- `examples/petstore`: minimal OpenAPI 3 workflow from spec to generated CLI.
- `examples/richapi`: broader dogfood example covering pagination, enums, headers, request bodies, public endpoints, streaming hints, and long-running operation hints.

## Add a Showcase

Open a pull request adding a short entry when you use Lathe in a real project. Good entries include:

- What API spec type you use.
- What generated CLI workflow Lathe replaced.
- Which catalog or agent workflow made the project safer or faster.
- Public links when available.
