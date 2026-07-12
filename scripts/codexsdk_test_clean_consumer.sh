#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
consumer="$(mktemp -d)"
trap 'rm -rf "${consumer}"' EXIT

cd "${consumer}"
go mod init example.com/codexsdk-clean-consumer >/dev/null
go mod edit -replace "github.com/ronhuafeng/codexsdk-go=${repo_root}"

cat > client_test.go <<'EOF'
package consumer

import (
	"context"
	"testing"

	"github.com/ronhuafeng/codexsdk-go/codexsdk"
	"github.com/ronhuafeng/codexsdk-go/codexsdk/protocolv2"
)

type runStarter interface {
	Start(context.Context, codexsdk.StartThreadRunRequest) (codexsdk.StartedThreadRun, error)
}

type modelLister interface {
	List(context.Context, protocolv2.ModelListParams) (protocolv2.ModelListResponse, error)
}

func useRoot(root *codexsdk.Client) error {
	var _ runStarter = root.ThreadRunner()
	var _ modelLister = root.Models()
	return root.Close()
}

func TestZeroValueLifecycle(t *testing.T) {
	var root codexsdk.Client
	if err := useRoot(&root); err != nil {
		t.Fatal(err)
	}
}
EOF

GOWORK=off go mod tidy
GOWORK=off go test ./...
