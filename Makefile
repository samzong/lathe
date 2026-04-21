GO       ?= go
OUT_DIR  := ./bin

BOLD  := \033[1m
CYAN  := \033[36m
GREEN := \033[32m
RESET := \033[0m

.DEFAULT_GOAL := help

# ── Bootstrap ────────────────────────────────────────────────────────────────

.PHONY: bootstrap sync-specs gen

bootstrap: sync-specs gen ## First-time setup — sync upstream specs + generate command tree
	@printf '\n$(GREEN)  ✓ bootstrap complete$(RESET) — next: $(CYAN)go build -o $(OUT_DIR)/<name> ./cmd/<name>$(RESET)\n\n'

sync-specs: ## Fetch upstream specs pinned in specs/sources.yaml
	$(GO) run ./cmd/specsync

gen: ## Regenerate internal/generated from cached specs
	$(GO) run ./cmd/codegen

# ── Quality ──────────────────────────────────────────────────────────────────

.PHONY: check test vet fmt fmt-check

check: ## Full quality gate — fmt-check, vet, test
	@printf '\n$(BOLD)[1/3] Checking format$(RESET)\n'
	@$(MAKE) --no-print-directory fmt-check
	@printf '\n$(BOLD)[2/3] Running vet$(RESET)\n'
	$(GO) vet ./...
	@printf '\n$(BOLD)[3/3] Running tests$(RESET)\n'
	$(GO) test ./...
	@printf '\n$(GREEN)  ✓ All checks passed$(RESET)\n\n'

test: ## Run tests
	$(GO) test ./...

vet: ## Run go vet
	$(GO) vet ./...

fmt: ## Format code in place
	$(GO) fmt ./...

fmt-check: ## Fail if any file needs gofmt
	@out=$$(gofmt -l cmd internal pkg); \
	if [ -n "$$out" ]; then \
	  printf '$(BOLD)gofmt violations:$(RESET)\n%s\n' "$$out"; \
	  exit 1; \
	fi

# ── Maintenance ──────────────────────────────────────────────────────────────

.PHONY: tidy clean

tidy: ## Tidy go.mod / go.sum
	$(GO) mod tidy

clean: ## Remove build artifacts + generated code
	rm -rf $(OUT_DIR) internal/generated

# ── Help ─────────────────────────────────────────────────────────────────────

.PHONY: help

help: ## Show available targets
	@awk 'BEGIN {FS = ":.*## "; printf "\n$(BOLD)lathe$(RESET) — spec-driven CLI generator\n"} \
		/^# ── / {n = $$0; gsub(/(^# ── | ─+$$)/, "", n); printf "\n$(BOLD)%s$(RESET)\n", n} \
		/^[a-zA-Z_-]+:.*## / {printf "  $(CYAN)make %-12s$(RESET) %s\n", $$1, $$2} \
		END {printf "\n"}' $(MAKEFILE_LIST)
