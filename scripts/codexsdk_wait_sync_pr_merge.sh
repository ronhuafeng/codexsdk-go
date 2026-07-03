#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/codexsdk_wait_sync_pr_merge.sh --pr-number <number> --sync-branch <branch> --land-ref <branch> [options]

Options:
  --attempts <n>          Poll attempts. Defaults to 120.
  --poll-interval <sec>   Seconds between polls. Defaults to 15.
  --remote <name>         Git remote to fetch. Defaults to origin.

Requests GitHub auto-merge for a sync PR, waits for protected-branch merge,
and writes landed_commit and landed_ref to GITHUB_OUTPUT when available.
EOF
}

attempts=120
land_ref=""
poll_interval=15
pr_number=""
remote="origin"
sync_branch=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --attempts)
      attempts="$2"
      shift 2
      ;;
    --land-ref)
      land_ref="$2"
      shift 2
      ;;
    --poll-interval)
      poll_interval="$2"
      shift 2
      ;;
    --pr-number)
      pr_number="$2"
      shift 2
      ;;
    --remote)
      remote="$2"
      shift 2
      ;;
    --sync-branch)
      sync_branch="$2"
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

if [[ -z "${pr_number}" || -z "${sync_branch}" || -z "${land_ref}" ]]; then
  usage >&2
  exit 2
fi

append_summary() {
  if [[ -z "${GITHUB_STEP_SUMMARY:-}" ]]; then
    return 0
  fi
  printf '%s\n' "$@" >> "${GITHUB_STEP_SUMMARY}"
}

write_output() {
  local name=$1
  local value=$2
  if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
    printf '%s=%s\n' "${name}" "${value}" >> "${GITHUB_OUTPUT}"
  fi
}

gh pr merge "${pr_number}" --auto --rebase
append_summary \
  "Auto-merge requested for sync PR #${pr_number}." \
  "" \
  "If the required \`Go\` pull request check is \`action_required\` with no jobs, a maintainer should approve or rerun that CI run once. This is an expected GitHub Actions safety gate for PRs created by \`GITHUB_TOKEN\`; auto-merge should continue after the real \`Go\` check passes."

for ((attempt = 0; attempt < attempts; attempt++)); do
  state="$(
    gh pr view "${pr_number}" \
      --json state \
      --jq '.state'
  )"
  if [[ "${state}" == "MERGED" ]]; then
    break
  fi
  sleep "${poll_interval}"
done

state="$(
  gh pr view "${pr_number}" \
    --json state \
    --jq '.state'
)"
if [[ "${state}" != "MERGED" ]]; then
  echo "Sync PR #${pr_number} was not merged before the workflow timeout window." >&2
  echo "If the required Go check is action_required with no jobs, rerun that CI run once with maintainer permissions; auto-merge should continue after the real Go check passes." >&2
  gh pr view "${pr_number}" \
    --json state,mergeStateStatus,reviewDecision,statusCheckRollup \
    --jq '.'
  gh pr checks "${pr_number}" || true
  gh run list \
    --workflow ci.yml \
    --branch "${sync_branch}" \
    --limit 10 \
    --json databaseId,event,status,conclusion,headSha,url \
    --jq '.' || true
  exit 1
fi

git fetch "${remote}" "refs/heads/${land_ref}:refs/remotes/${remote}/${land_ref}"
landed_commit="$(git rev-parse "refs/remotes/${remote}/${land_ref}")"
write_output "landed_commit" "${landed_commit}"
write_output "landed_ref" "${land_ref}"
append_summary \
  "" \
  "## Sync PR merged" \
  "" \
  "- PR: #${pr_number}" \
  "- Landing ref: \`${land_ref}\`" \
  "- Landed commit: \`${landed_commit}\`"
