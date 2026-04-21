OUT_DIR  := ./bin
GO       ?= go

.PHONY: bootstrap tidy test vet fmt clean sync-specs gen

# First-time setup in a downstream fork: pull specs from upstream + generate code.
# Requires specs/sources.yaml to be populated first; this template ships empty.
# Re-run bootstrap whenever you bump a pinned_tag in specs/sources.yaml.
bootstrap: sync-specs gen
	@echo "bootstrap complete — now: go build -o $(OUT_DIR)/<name> ./cmd/<name>"

sync-specs:
	$(GO) run ./cmd/specsync

gen:
	$(GO) run ./cmd/codegen

test:
	$(GO) test ./...

vet:
	$(GO) vet ./cmd/... ./internal/...

fmt:
	$(GO) fmt ./cmd/... ./internal/...

tidy:
	$(GO) mod tidy

clean:
	rm -rf $(OUT_DIR) internal/generated
