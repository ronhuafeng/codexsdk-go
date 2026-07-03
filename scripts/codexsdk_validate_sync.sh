#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/codexsdk_validate_sync.sh [options]

Options:
  --candidate <path>   Candidate schema directory that must match the baseline.
  --target-sha <sha>   Expected baseline source_commit.

Runs the validation gate required after an upstream protocol baseline sync.
EOF
}

candidate=""
target_sha=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --candidate)
      candidate="$2"
      shift 2
      ;;
    --target-sha)
      target_sha="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
baseline="codexsdk/internal/protocolschema/appserver/v2"

cd "${repo_root}"

git ls-files -z -- '*.go' ':!:vendor/**' | xargs -0 gofmt -w
GOWORK=off go vet ./...
GOWORK=off go test ./...

tmp="$(mktemp -d)"
trap 'rm -rf "${tmp}"' EXIT
GOWORK=off go run ./codexsdk/internal/cmd/protocolv2gen -out "${tmp}"
diff -u codexsdk/protocolv2/method_registry.gen.go "${tmp}/method_registry.gen.go"
diff -u codexsdk/protocolv2/protocol_types.gen.go "${tmp}/protocol_types.gen.go"

git diff --check

sync_state_args=(
  --baseline "${baseline}"
)
if [[ -n "${candidate}" ]]; then
  sync_state_args+=(--candidate "${candidate}")
fi
if [[ -n "${target_sha}" ]]; then
  sync_state_args+=(--target-sha "${target_sha}")
fi
python3 scripts/codexsdk_sync_state.py "${sync_state_args[@]}"

if grep -R -n -E '/Users/|/home/|[.]cache/codexsdk-upstream|[.]cache/openai-codex' "${baseline}"; then
  echo "checked-in protocol baseline contains local or cache paths" >&2
  exit 1
fi
