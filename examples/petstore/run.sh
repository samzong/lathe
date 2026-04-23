#!/bin/bash
set -euo pipefail

LATHE_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
DIR="$(mktemp -d)"
trap 'rm -rf "$DIR"' EXIT

echo "==> Building codegen tool..."
CODEGEN="$DIR/codegen"
(cd "$LATHE_ROOT" && go build -o "$CODEGEN" ./cmd/codegen)

cd "$DIR"

echo "==> Setting up example project..."

# Go module with replace directive pointing at local lathe
go mod init example/petstore
go mod edit -require "github.com/samzong/lathe@v0.0.0"
go mod edit -replace "github.com/samzong/lathe=$LATHE_ROOT"

# CLI identity
cat > cli.yaml << 'EOF'
cli:
  name: petstore
  short: "Petstore CLI demo"
EOF

# Source config
mkdir -p specs
cat > specs/sources.yaml << 'EOF'
sources:
  pets:
    repo_url: https://example.com/petstore.git
    pinned_tag: v1.0.0
    backend: openapi3
    openapi3:
      files: [openapi.yaml]
EOF

# Pre-stage spec (replaces make sync-specs)
SYNC=.cache/specs-sync/pets
mkdir -p "$SYNC"
cat > "$SYNC/openapi.yaml" << 'EOF'
openapi: "3.0.3"
paths:
  /pets:
    get:
      operationId: Pet_List
      tags: [Pets]
      summary: List all pets
      responses:
        "200":
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: "#/components/schemas/Pet"
  /pets/{id}:
    get:
      operationId: Pet_Get
      tags: [Pets]
      summary: Get a pet by ID
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Pet"
    delete:
      operationId: Pet_Delete
      tags: [Pets]
      summary: Delete a pet
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        "204": {}
components:
  schemas:
    Pet:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
        status:
          type: string
EOF

cat > "$SYNC/sync-state.yaml" << 'EOF'
source: pets
backend: openapi3
synced_from: v1.0.0
resolved_sha: "0000000000000000000000000000000000000000"
EOF

# Run codegen
echo "==> Running codegen..."
"$CODEGEN" -sources specs/sources.yaml -cache .cache

# main.go
mkdir -p cmd/petstore
cp cli.yaml cmd/petstore/cli.yaml
cat > cmd/petstore/main.go << 'GOEOF'
package main

import (
	_ "embed"
	"os"

	"github.com/samzong/lathe/pkg/config"
	"github.com/samzong/lathe/pkg/lathe"
	"github.com/samzong/lathe/pkg/runtime"

	"example/petstore/internal/generated"
)

//go:embed cli.yaml
var manifestBytes []byte

func main() {
	m, err := config.Load(manifestBytes)
	if err != nil {
		panic(err)
	}
	config.Bind(m)
	root := lathe.NewApp(m)
	generated.MountModules(root)
	os.Exit(runtime.Execute(root))
}
GOEOF

# Build
echo "==> Building petstore CLI..."
go mod tidy
go build -o bin/petstore ./cmd/petstore

# Demo
echo ""
echo "========== petstore --help =========="
./bin/petstore --help
echo ""
echo "========== petstore pets --help =========="
./bin/petstore pets --help
echo ""
echo "Done. All temp files cleaned up on exit."
