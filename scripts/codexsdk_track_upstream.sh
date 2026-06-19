#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/codexsdk_track_upstream.sh --commit <codex-tag-ref-or-commit> [options]

Options:
  --codex-repo <path>   Codex source repo. Defaults to CODEXSDK_CODEX_REPO.
  --codex-bin <path>    Codex binary used to generate schema. Defaults to codex.
  --baseline <path>     Checked-in SDK schema baseline.
  --generator <mode>    cargo or binary. Defaults to cargo.
  --out <path>          Output workdir for generated schema and reports.
  --source-ref <name>   Original upstream tag/ref name used for provenance.
  --source-ref-kind <k> Upstream target kind, such as stable_rust_tag.
  --no-fetch            Do not fetch the target commit before resolving it.

The workflow is read-only for the checked-in baseline. It fetches the requested
Codex tag, ref, or commit when needed, generates an app-server schema snapshot,
and writes review artifacts under --out. In binary mode, --codex-bin must point
to a Codex binary built from the target commit. In cargo mode, the script
creates a temporary worktree at the target commit and runs cargo from that
worktree.
EOF
}

codex_repo="${CODEXSDK_CODEX_REPO:-}"
codex_bin="codex"
baseline="codexsdk/internal/protocolschema/appserver/v2"
generator="cargo"
out=""
commit=""
source_ref=""
source_ref_kind=""
fetch=1
worktree=""

cleanup() {
  if [[ -n "${worktree}" && -d "${worktree}" ]]; then
    git -C "${codex_repo}" worktree remove --force "${worktree}" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

while [[ $# -gt 0 ]]; do
  case "$1" in
    --codex-repo)
      codex_repo="$2"
      shift 2
      ;;
    --codex-bin)
      codex_bin="$2"
      shift 2
      ;;
    --baseline)
      baseline="$2"
      shift 2
      ;;
    --generator)
      generator="$2"
      shift 2
      ;;
    --out)
      out="$2"
      shift 2
      ;;
    --commit)
      commit="$2"
      shift 2
      ;;
    --source-ref)
      source_ref="$2"
      shift 2
      ;;
    --source-ref-kind)
      source_ref_kind="$2"
      shift 2
      ;;
    --no-fetch)
      fetch=0
      shift
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

if [[ -z "${commit}" ]]; then
  echo "--commit is required" >&2
  usage >&2
  exit 2
fi
if [[ -z "${codex_repo}" ]]; then
  echo "--codex-repo is required unless CODEXSDK_CODEX_REPO is set" >&2
  usage >&2
  exit 2
fi
if [[ ! -d "${codex_repo}/.git" ]]; then
  echo "codex repo is not a git repository: ${codex_repo}" >&2
  exit 1
fi
if [[ ! -d "${baseline}" ]]; then
  echo "baseline directory does not exist: ${baseline}" >&2
  exit 1
fi
if [[ "${generator}" != "binary" && "${generator}" != "cargo" ]]; then
  echo "--generator must be binary or cargo" >&2
  exit 2
fi
if [[ -z "${out}" ]]; then
  out="$(mktemp -d "${TMPDIR:-/tmp}/codexsdk-upstream.XXXXXX")"
fi

resolve_target_ref() {
  local ref="$1"
  local candidate=""
  local candidates=("${ref}")

  case "${ref}" in
    refs/tags/*)
      candidates+=("${ref}")
      ;;
    refs/heads/*)
      candidates+=("refs/remotes/origin/${ref#refs/heads/}")
      ;;
    *)
      candidates+=("refs/tags/${ref}" "refs/remotes/origin/${ref}")
      ;;
  esac

  for candidate in "${candidates[@]}"; do
    if git -C "${codex_repo}" rev-parse --verify -q "${candidate}^{commit}"; then
      return 0
    fi
  done
  return 1
}

fetch_target_ref() {
  local ref="$1"

  if [[ "${ref}" =~ ^[0-9a-f]{40}$ ]]; then
    git -C "${codex_repo}" fetch origin "${ref}"
    return
  fi

  case "${ref}" in
    refs/tags/*)
      git -C "${codex_repo}" fetch origin "${ref}:${ref}"
      ;;
    refs/heads/*)
      git -C "${codex_repo}" fetch origin "${ref}:refs/remotes/origin/${ref#refs/heads/}"
      ;;
    *)
      git -C "${codex_repo}" fetch origin "refs/tags/${ref}:refs/tags/${ref}" 2>/dev/null \
        || git -C "${codex_repo}" fetch origin "refs/heads/${ref}:refs/remotes/origin/${ref}" 2>/dev/null \
        || git -C "${codex_repo}" fetch origin "${ref}"
      ;;
  esac
}

if [[ "${fetch}" -eq 1 ]] && ! resolve_target_ref "${commit}" >/dev/null; then
  fetch_target_ref "${commit}"
fi
if ! resolved_commit="$(resolve_target_ref "${commit}")"; then
  echo "unable to resolve Codex target as a commit: ${commit}" >&2
  exit 1
fi
if [[ -z "${source_ref}" ]]; then
  source_ref="${commit}"
fi
if [[ -z "${source_ref_kind}" ]]; then
  if [[ "${source_ref}" =~ ^rust-v[0-9]+[.][0-9]+[.][0-9]+$ ]]; then
    source_ref_kind="stable_rust_tag"
  elif [[ "${source_ref}" =~ ^[0-9a-f]{40}$ ]]; then
    source_ref_kind="manual_commit"
  else
    source_ref_kind="manual_ref"
  fi
fi
generated="${out}/schema"
reports="${out}/reports"
if [[ "${generated}" == "/" || "${reports}" == "/" ]]; then
  echo "refusing to use unsafe output paths under --out=${out}" >&2
  exit 1
fi
rm -rf "${generated}" "${reports}"
mkdir -p "${generated}" "${reports}"

case "${generator}" in
  binary)
    "${codex_bin}" app-server generate-json-schema --experimental --out "${generated}"
    codex_version="$("${codex_bin}" --version 2>/dev/null || true)"
    generator_detail="${codex_bin}"
    ;;
  cargo)
    worktree="${out}/codex-worktree"
    git -C "${codex_repo}" worktree add --detach "${worktree}" "${resolved_commit}" >/dev/null
    (
      cd "${worktree}/codex-rs"
      cargo run -p codex-cli -- app-server generate-json-schema --experimental --out "${generated}"
    )
    codex_version="$(
      cd "${worktree}/codex-rs"
      cargo run -p codex-cli -- --version 2>/dev/null || true
    )"
    generator_detail="${worktree}/codex-rs cargo run -p codex-cli"
    ;;
esac

python3 - "${baseline}" "${generated}" "${reports}" "${resolved_commit}" "${codex_repo}" "${codex_version}" "${generator}" "${generator_detail}" "${source_ref}" "${source_ref_kind}" <<'PY'
import hashlib
import json
import pathlib
import sys

baseline = pathlib.Path(sys.argv[1])
generated = pathlib.Path(sys.argv[2])
reports = pathlib.Path(sys.argv[3])
commit = sys.argv[4]
repo = sys.argv[5]
codex_version = sys.argv[6]
generator = sys.argv[7]
generator_detail = sys.argv[8]
source_ref = sys.argv[9]
source_ref_kind = sys.argv[10]
metadata = {
    "baseline_metadata.json",
    "coverage_matrix.json",
    "drift_report.json",
    "manifest.json",
    "manifest_generation.json",
    "matrix_update_skeleton.json",
}

def hashes(root):
    out = {}
    for path in root.rglob("*.json"):
        if path.name in metadata:
            continue
        rel = path.relative_to(root).as_posix()
        value = json.loads(path.read_text(encoding="utf-8"))
        canonical = json.dumps(value, sort_keys=True, separators=(",", ":")).encode()
        out[rel] = hashlib.sha256(canonical).hexdigest()
    return out

def method_names(root, rel):
    path = root / rel
    if not path.exists():
        return set()
    value = json.loads(path.read_text(encoding="utf-8"))
    names = set()
    for item in value.get("oneOf", []):
        enum = item.get("properties", {}).get("method", {}).get("enum", [])
        if enum:
            names.add(enum[0])
    return names

base = hashes(baseline)
candidate = hashes(generated)
added = sorted(set(candidate) - set(base))
removed = sorted(set(base) - set(candidate))
changed = sorted(k for k in set(base) & set(candidate) if base[k] != candidate[k])
method_diff = {}
for rel in ("ClientRequest.json", "ServerRequest.json", "ServerNotification.json", "ClientNotification.json"):
    before = method_names(baseline, rel)
    after = method_names(generated, rel)
    method_diff[rel] = {
        "added": sorted(after - before),
        "removed": sorted(before - after),
    }

status = "clean"
if added or removed or changed or any(v["added"] or v["removed"] for v in method_diff.values()):
    status = "review-required"

drift = {
    "status": status,
    "comparison_mode": "canonical-json",
    "target": {
        "source_repo": repo,
        "source_ref_name": source_ref,
        "source_ref_kind": source_ref_kind,
        "source_commit": commit,
        "codex_version": codex_version,
        "generator": generator,
        "generator_detail": generator_detail,
    },
    "file_diff": {
        "added": added,
        "changed": changed,
        "removed": removed,
    },
    "method_diff": method_diff,
    "matrix_update_skeleton": "matrix_update_skeleton.json",
}
matrix = {
    "status": "empty" if status == "clean" else "review-required",
    "source": "drift_summary.json",
    "valid_statuses": [
        "supported",
        "supported-generated",
        "deferred",
        "intentionally-unsupported",
    ],
    "method_updates": [
        {"method": method, "source_schema": schema, "change": change, "status": "review-required"}
        for schema, diff in method_diff.items()
        for change, methods in (("added", diff["added"]), ("removed", diff["removed"]))
        for method in methods
    ],
    "type_updates": (
        [{"schema": path, "change": "added", "status": "review-required"} for path in added]
        + [{"schema": path, "change": "changed", "status": "review-required"} for path in changed]
        + [{"schema": path, "change": "removed", "status": "review-required"} for path in removed]
    ),
    "field_updates": [],
}
reports.mkdir(parents=True, exist_ok=True)
(reports / "drift_summary.json").write_text(json.dumps(drift, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
(reports / "matrix_update_skeleton.json").write_text(json.dumps(matrix, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
(reports / "SUMMARY.md").write_text(
    "\n".join([
        "# Codex SDK Upstream Tracking",
        "",
        f"- status: `{status}`",
        f"- source repo: `{repo}`",
        f"- source ref: `{source_ref}`",
        f"- source ref kind: `{source_ref_kind}`",
        f"- source commit: `{commit}`",
        f"- codex version: `{codex_version}`",
        f"- generated schema: `{generated}`",
        f"- drift summary: `{reports / 'drift_summary.json'}`",
        f"- matrix update skeleton: `{reports / 'matrix_update_skeleton.json'}`",
        "",
        "Review the generated reports before updating the checked-in baseline.",
        "",
    ]),
    encoding="utf-8",
)
print(reports / "SUMMARY.md")
PY

echo "generated schema: ${generated}"
echo "reports: ${reports}"
