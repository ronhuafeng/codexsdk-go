#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/codexsdk_land_sync.sh --land-ref <branch> --work-branch <branch> --target-ref <ref> --target-sha <sha> [options]

Options:
  --candidate <path>        Candidate schema directory validated against the checked-in baseline.
  --remote <name>           Git remote to fetch and push. Defaults to origin.
  --open-pr-on-failure      Create or reuse a draft PR when direct fast-forward landing fails.
  --auto-merge-pr-on-failure
                            Create or reuse a ready PR, wait for required checks,
                            and merge it when direct fast-forward landing fails.
  --require-bot-token-for-auto-merge
                            Refuse auto-merge fallback unless CODEXSDK_SYNC_BOT_TOKEN
                            is set. PRs created by GITHUB_TOKEN do not trigger
                            pull_request CI, so protected branches cannot observe
                            required checks from that token alone.
  --merge-method <method>   PR merge method for auto-merge fallback: rebase, merge,
                            or squash. Defaults to rebase.

The script assumes HEAD is the committed sync change on the temporary work branch.
It validates, rebases onto the current remote landing ref, pushes the temporary
branch, then attempts a non-force fast-forward push to the landing ref. If the
landing ref moves during the attempt, it rebases and validates once more.
EOF
}

remote="origin"
land_ref=""
work_branch=""
target_ref=""
target_sha=""
candidate=""
open_pr_on_failure=0
auto_merge_pr_on_failure=0
require_bot_token_for_auto_merge=0
merge_method="rebase"

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
    --open-pr-on-failure)
      open_pr_on_failure=1
      shift
      ;;
    --auto-merge-pr-on-failure)
      open_pr_on_failure=1
      auto_merge_pr_on_failure=1
      shift
      ;;
    --require-bot-token-for-auto-merge)
      require_bot_token_for_auto_merge=1
      shift
      ;;
    --merge-method)
      merge_method="$2"
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
    --target-sha)
      target_sha="$2"
      shift 2
      ;;
    --work-branch)
      work_branch="$2"
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

if [[ -z "${land_ref}" || -z "${work_branch}" || -z "${target_ref}" || -z "${target_sha}" ]]; then
  usage >&2
  exit 2
fi
land_ref="${land_ref#refs/heads/}"
work_branch="${work_branch#refs/heads/}"
if [[ "${land_ref}" == "${work_branch}" ]]; then
  echo "land ref and work branch must differ" >&2
  exit 2
fi
case "${merge_method}" in
  rebase|merge|squash)
    ;;
  *)
    echo "unsupported merge method: ${merge_method}" >&2
    exit 2
    ;;
esac

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

push_work_branch() {
  git push --force-with-lease "${remote}" "HEAD:refs/heads/${work_branch}"
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

open_or_reuse_pr() {
  local mode=${1:-draft}
  local body title url
  title="Sync Codex protocol baseline to ${target_ref}"
  body="Automated upstream protocol sync could not land directly."
  if [[ "${mode}" == "ready" ]]; then
    body+=" Required CI will be checked before the workflow merges this PR."
  else
    body+=" Review the branch and merge after CI passes."
  fi
  url="$(
    gh pr list \
      --head "${work_branch}" \
      --base "${land_ref}" \
      --state open \
      --json url \
      --jq '.[0].url // empty'
  )"
  if [[ -z "${url}" ]]; then
    local create_args=(
      --base "${land_ref}"
      --head "${work_branch}"
      --title "${title}"
      --body "${body}"
    )
    if [[ "${mode}" != "ready" ]]; then
      create_args=(--draft "${create_args[@]}")
    fi
    url="$(gh pr create "${create_args[@]}")"
  elif [[ "${mode}" == "ready" ]]; then
    gh pr ready "${url}" >/dev/null
  fi
  write_output "fallback_pr_url" "${url}"
  echo "Opened fallback PR: ${url}" >&2
  printf '%s\n' "${url}"
}

wait_for_required_pr_checks() {
  local pr_url=$1
  local timeout_seconds=${CODEXSDK_PR_CHECK_TIMEOUT_SECONDS:-3600}
  local interval_seconds=${CODEXSDK_PR_CHECK_INTERVAL_SECONDS:-15}
  local deadline=$((SECONDS + timeout_seconds))
  local checks_json rc check_count failed_count pending_count

  while (( SECONDS < deadline )); do
    set +e
    checks_json="$(
      gh pr checks "${pr_url}" \
        --required \
        --json name,bucket,state,link
    )"
    rc=$?
    set -e

    if [[ "${rc}" -eq 0 || "${rc}" -eq 8 ]]; then
      check_count="$(jq 'length' <<<"${checks_json}")"
      failed_count="$(jq '[.[] | select(.bucket == "fail" or .bucket == "cancel")] | length' <<<"${checks_json}")"
      pending_count="$(jq '[.[] | select(.bucket == "pending")] | length' <<<"${checks_json}")"

      if [[ "${failed_count}" != "0" ]]; then
        echo "Required PR checks failed for ${pr_url}:" >&2
        jq -r '.[] | select(.bucket == "fail" or .bucket == "cancel") | "- \(.name): \(.state)"' <<<"${checks_json}" >&2
        return 1
      fi
      if [[ "${check_count}" != "0" && "${pending_count}" == "0" ]]; then
        echo "Required PR checks passed for ${pr_url}." >&2
        return 0
      fi
    elif [[ "${rc}" -ne 1 ]]; then
      echo "Unable to read PR checks for ${pr_url}." >&2
      return "${rc}"
    fi

    sleep "${interval_seconds}"
  done

  echo "Timed out waiting for required PR checks for ${pr_url}." >&2
  return 1
}

merge_flag_for_method() {
  case "${merge_method}" in
    rebase)
      printf '%s\n' "--rebase"
      ;;
    merge)
      printf '%s\n' "--merge"
      ;;
    squash)
      printf '%s\n' "--squash"
      ;;
  esac
}

auto_merge_fallback_pr() {
  local pr_url merge_flag landed_commit
  if [[ "${require_bot_token_for_auto_merge}" -eq 1 && -z "${CODEXSDK_SYNC_BOT_TOKEN:-}" ]]; then
    echo "auto-merge fallback requires CODEXSDK_SYNC_BOT_TOKEN." >&2
    echo "GitHub does not trigger pull_request CI for PRs created by the workflow GITHUB_TOKEN, so required checks would never appear." >&2
    return 1
  fi
  pr_url="$(open_or_reuse_pr ready)"
  wait_for_required_pr_checks "${pr_url}"
  merge_flag="$(merge_flag_for_method)"
  gh pr merge "${pr_url}" "${merge_flag}" --delete-branch --match-head-commit "$(git rev-parse HEAD)"
  fetch_landing_ref
  landed_commit="$(git rev-parse "${remote}/${land_ref}")"
  write_output "fallback_pr_merged" "true"
  write_output "landed_commit" "${landed_commit}"
  write_output "landed_ref" "${land_ref}"
  write_output "work_branch" "${work_branch}"
}

fail_with_pr() {
  local reason=$1
  echo "${reason}" >&2
  git rebase --abort >/dev/null 2>&1 || true
  if [[ "${open_pr_on_failure}" -eq 1 ]]; then
    push_work_branch || true
    open_or_reuse_pr draft || true
  fi
  echo "Unable to land ${work_branch} into ${land_ref}." >&2
  exit 1
}

validate_sync
if ! rebase_and_validate; then
  fail_with_pr "Pre-main gate failed during rebase, target movement check, or validation."
fi
push_work_branch

if try_land_fast_forward; then
  landed_commit="$(git rev-parse HEAD)"
  write_output "landed_commit" "${landed_commit}"
  write_output "landed_ref" "${land_ref}"
  write_output "work_branch" "${work_branch}"
  exit 0
fi

echo "Landing ref changed or rejected the fast-forward push; retrying once after rebase." >&2
if ! rebase_and_validate; then
  fail_with_pr "Retry gate failed during rebase, target movement check, or validation."
fi
push_work_branch

if try_land_fast_forward; then
  landed_commit="$(git rev-parse HEAD)"
  write_output "landed_commit" "${landed_commit}"
  write_output "landed_ref" "${land_ref}"
  write_output "work_branch" "${work_branch}"
  exit 0
fi

if [[ "${auto_merge_pr_on_failure}" -eq 1 ]]; then
  auto_merge_fallback_pr
  exit 0
fi

if [[ "${open_pr_on_failure}" -eq 1 ]]; then
  open_or_reuse_pr draft >/dev/null
fi

echo "Unable to land ${work_branch} into ${land_ref} after one retry." >&2
exit 1
