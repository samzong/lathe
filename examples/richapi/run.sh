#!/bin/bash
set -euo pipefail

LATHE_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
DIR="$(mktemp -d)"
trap 'rm -rf "$DIR"' EXIT

echo "==> Building codegen tool..."
CODEGEN="$DIR/codegen"
(cd "$LATHE_ROOT" && go build -o "$CODEGEN" ./cmd/codegen)

cd "$DIR"

echo "==> Setting up richapi project..."

go mod init example/richapi
go mod edit -require "github.com/samzong/lathe@v0.0.0"
go mod edit -replace "github.com/samzong/lathe=$LATHE_ROOT"

cat > cli.yaml << 'EOF'
cli:
  name: richapi
  short: "Rich API CLI — dogfooding all v0.1 features"
EOF

mkdir -p specs
cat > specs/sources.yaml << 'EOF'
sources:
  acme:
    repo_url: https://example.com/acme.git
    pinned_tag: v1.0.0
    backend: openapi3
    openapi3:
      files: [openapi.yaml]
EOF

SYNC=.cache/specs-sync/acme
mkdir -p "$SYNC"
cat > "$SYNC/openapi.yaml" << 'SPECEOF'
openapi: "3.0.3"
info:
  title: Acme API
  version: "1.0.0"
security:
  - bearerAuth: []
paths:
  /healthz:
    get:
      operationId: System_Healthz
      tags: [System]
      summary: Health check (public, no auth required)
      security: []
      responses:
        "200":
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string

  /users:
    get:
      operationId: Users_List
      tags: [Users]
      summary: List users with pagination
      parameters:
        - name: page_token
          in: query
          schema:
            type: string
          description: Cursor for next page
        - name: limit
          in: query
          schema:
            type: integer
            default: 20
          description: Max results per page
        - name: status
          in: query
          schema:
            type: string
            enum: [active, inactive, suspended]
          description: Filter by status
        - name: sort
          in: query
          schema:
            type: string
            default: created_at
          description: Sort field
      responses:
        "200":
          content:
            application/json:
              schema:
                type: object
                properties:
                  items:
                    type: array
                    items:
                      $ref: "#/components/schemas/User"
                  next_page_token:
                    type: string
    post:
      operationId: Users_Create
      tags: [Users]
      summary: Create a new user
      security:
        - bearerAuth: [admin:write]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/UserInput"
      responses:
        "201":
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"

  /users/{user_id}:
    parameters:
      - name: user_id
        in: path
        required: true
        schema:
          type: string
          format: uuid
    get:
      operationId: Users_Get
      tags: [Users]
      summary: Get user by ID
      responses:
        "200":
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"
    patch:
      operationId: Users_Update
      tags: [Users]
      summary: Update a user
      security:
        - bearerAuth: [admin:write]
      requestBody:
        required: false
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/UserInput"
      responses:
        "200":
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"
    delete:
      operationId: Users_Delete
      tags: [Users]
      summary: Delete a user (deprecated)
      deprecated: true
      security:
        - bearerAuth: [admin:write, admin:delete]
      responses:
        "204": {}

  /users/{user_id}/avatar:
    post:
      operationId: Users_UploadAvatar
      tags: [Users]
      summary: Upload user avatar
      parameters:
        - name: user_id
          in: path
          required: true
          schema:
            type: string
        - name: X-Request-Id
          in: header
          schema:
            type: string
          description: Trace ID for request tracking
      requestBody:
        required: true
        content:
          application/octet-stream:
            schema:
              type: string
              format: binary
      responses:
        "200":
          content:
            application/json:
              schema:
                type: object
                properties:
                  url:
                    type: string

  /reports/{report_id}/download:
    get:
      operationId: Reports_Download
      tags: [Reports]
      summary: Download report as PDF
      parameters:
        - name: report_id
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          content:
            application/pdf:
              schema:
                type: string
                format: binary

  /events:
    get:
      operationId: Events_Stream
      tags: [Events]
      summary: Stream events via SSE
      responses:
        "200":
          content:
            text/event-stream:
              schema:
                type: string

  /jobs:
    post:
      operationId: Jobs_Create
      tags: [Jobs]
      summary: Create a long-running job
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                type:
                  type: string
                params:
                  type: object
      responses:
        "202":
          content:
            application/json:
              schema:
                type: object
                properties:
                  job_id:
                    type: string
                  status:
                    type: string

components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
  schemas:
    User:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
        email:
          type: string
        status:
          type: string
        created_at:
          type: string
          format: date-time
    UserInput:
      type: object
      properties:
        name:
          type: string
        email:
          type: string
SPECEOF

cat > "$SYNC/sync-state.yaml" << 'EOF'
source: acme
backend: openapi3
synced_from: v1.0.0
resolved_sha: "0000000000000000000000000000000000000000"
EOF

echo "==> Running codegen..."
"$CODEGEN" -sources specs/sources.yaml -cache .cache

echo "==> Inspecting generated code..."
echo "--- generated specs file ---"
cat internal/generated/acme/acme_gen.go

mkdir -p cmd/richapi
cp cli.yaml cmd/richapi/cli.yaml
cat > cmd/richapi/main.go << 'GOEOF'
package main

import (
	_ "embed"
	"os"

	"github.com/samzong/lathe/pkg/config"
	"github.com/samzong/lathe/pkg/lathe"
	"github.com/samzong/lathe/pkg/runtime"

	"example/richapi/internal/generated"
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

echo "==> Building richapi CLI..."
go mod tidy
go build -o bin/richapi ./cmd/richapi

echo ""
echo "=========================================="
echo "  Dogfooding: richapi CLI help output"
echo "=========================================="
echo ""

echo "--- richapi --help ---"
./bin/richapi --help
echo ""

echo "--- richapi acme --help ---"
./bin/richapi acme --help
echo ""

echo "--- richapi acme users --help ---"
./bin/richapi acme users --help
echo ""

echo "--- richapi acme users list --help ---"
./bin/richapi acme users list --help
echo ""

echo "--- richapi acme users create --help ---"
./bin/richapi acme users create --help
echo ""

echo "--- richapi acme users delete --help (deprecated + scopes) ---"
./bin/richapi acme users delete --help
echo ""

echo "--- richapi acme system --help ---"
./bin/richapi acme system --help
echo ""

echo "--- richapi acme system healthz --help (public endpoint) ---"
./bin/richapi acme system healthz --help
echo ""

echo "--- richapi acme events --help ---"
./bin/richapi acme events --help
echo ""

echo "--- richapi acme reports --help ---"
./bin/richapi acme reports --help
echo ""

echo "--- richapi acme jobs --help ---"
./bin/richapi acme jobs --help
echo ""

DEST="$LATHE_ROOT/examples/richapi/bin"
mkdir -p "$DEST"
cp bin/richapi "$DEST/richapi"

echo "=========================================="
echo "  Done. Binary saved to examples/richapi/bin/richapi"
echo "=========================================="
