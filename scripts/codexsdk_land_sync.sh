#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/codexsdk_land_sync.sh --land-ref <branch> --target-ref <ref> --target-sha <sha> [options]

Options:
  --candidate <path>        Candidate schema directory validated against the checked-in baseline.
  --remote <name>           Git remote to fetch and push. Defaults to origin.
  --target-kind <kind>      Upstream target kind, such as stable_rust_tag.

The script assumes HEAD is the committed sync change. It validates, rebases onto
the current remote landing ref, then fast-forward pushes the landing ref.
EOF
}

remote="origin"
land_ref=""
target_ref=""
target_kind=""
target_sha=""
candidate=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --candidate)
      candidate="$2"
      shift 2
      ;;
    --land-ref)
      land_ref="$2"
      shift 2
      ;;
    --remote)
      remote="$2"
      shift 2
      ;;
    --target-ref)
      target_ref="$2"
      shift 2
      ;;
    --target-kind)
      target_kind="$2"
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

if [[ -z "${land_ref}" || -z "${target_ref}" || -z "${target_sha}" ]]; then
  usage >&2
  exit 2
fi
if [[ -z "${target_kind}" ]]; then
  if [[ "${target_ref}" =~ ^rust-v[0-9]+[.][0-9]+[.][0-9]+$ ]]; then
    target_kind="stable_rust_tag"
  elif [[ "${target_ref}" =~ ^[0-9a-f]{40}$ ]]; then
    target_kind="manual_commit"
  else
    target_kind="manual_ref"
  fi
fi
land_ref="${land_ref#refs/heads/}"

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "worktree must be clean before landing a sync commit" >&2
  git status --short >&2
  exit 1
fi

validate_sync() {
  local args=(
    --target-sha "${target_sha}"
  )
  if [[ -n "${candidate}" ]]; then
    args+=(--candidate "${candidate}")
  fi
  scripts/codexsdk_validate_sync.sh "${args[@]}" || return 1
  if ! git diff --quiet || ! git diff --cached --quiet; then
    echo "validation changed the committed sync tree; commit validation changes before landing" >&2
    git status --short >&2
    return 1
  fi
}

confirm_target_still_points_at_sha() {
  local resolved_sha
  resolved_sha="$(
    python3 scripts/codexsdk_resolve_upstream.py \
      --upstream-ref "${target_ref}" \
      --json |
      jq -r '.upstream_sha'
  )"
  if [[ "${resolved_sha}" != "${target_sha}" ]]; then
    echo "upstream target moved: ${target_ref} resolved to ${resolved_sha}, expected ${target_sha}" >&2
    return 1
  fi
}

fetch_landing_ref() {
  git fetch "${remote}" "refs/heads/${land_ref}:refs/remotes/${remote}/${land_ref}"
}

rebase_and_validate() {
  fetch_landing_ref
  git rebase "${remote}/${land_ref}" || return 1
  confirm_target_still_points_at_sha || return 1
  validate_sync || return 1
}

try_land_fast_forward() {
  fetch_landing_ref
  local remote_head
  remote_head="$(git rev-parse "${remote}/${land_ref}")"
  if ! git merge-base --is-ancestor "${remote_head}" HEAD; then
    echo "HEAD is not based on ${remote}/${land_ref}; refusing to land" >&2
    return 1
  fi
  git push "${remote}" "HEAD:refs/heads/${land_ref}"
}

write_output() {
  local name=$1
  local value=$2
  if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
    printf '%s=%s\n' "${name}" "${value}" >> "${GITHUB_OUTPUT}"
  fi
}

fail_publish() {
  local reason=$1
  echo "${reason}" >&2
  git rebase --abort >/dev/null 2>&1 || true
  echo "Unable to publish sync commit for ${land_ref}." >&2
  exit 1
}

validate_sync
if ! rebase_and_validate; then
  fail_publish "Pre-publish gate failed during rebase, target movement check, or validation."
fi

if try_land_fast_forward; then
  landed_commit="$(git rev-parse HEAD)"
  write_output "landed_commit" "${landed_commit}"
  write_output "landed_ref" "${land_ref}"
  exit 0
fi

echo "Landing ref changed or rejected the fast-forward push; retrying once after rebase." >&2
if ! rebase_and_validate; then
  fail_publish "Retry gate failed during rebase, target movement check, or validation."
fi

if try_land_fast_forward; then
  landed_commit="$(git rev-parse HEAD)"
  write_output "landed_commit" "${landed_commit}"
  write_output "landed_ref" "${land_ref}"
  exit 0
fi

echo "Unable to fast-forward ${land_ref} after one retry." >&2
exit 1
