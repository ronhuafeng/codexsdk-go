<!-- codexsdk-upstream-sync
baseline_commit: ${BASELINE_SHA}
upstream_ref: ${UPSTREAM_REF}
upstream_ref_kind: ${UPSTREAM_REF_KIND}
upstream_commit: ${UPSTREAM_SHA}
drift_sha256: ${DRIFT_SHA}
-->

## Protocol Drift Detected

The checked-in Codex app-server schema baseline no longer matches upstream `openai/codex`.

- Baseline: `openai/codex@${BASELINE_SHA}`
- Upstream target: ${UPSTREAM_DISPLAY}
- Upstream target kind: `${UPSTREAM_REF_KIND}`
- Drift status: `${STATUS}`
- Drift fingerprint: `${DRIFT_SHA}`
- Workflow run: ${RUN_URL}

## File Diff

- Added: ${ADDED_COUNT}
- Changed: ${CHANGED_COUNT}
- Removed: ${REMOVED_COUNT}

### Added Schemas

${ADDED_SCHEMAS}

### Changed Schemas

${CHANGED_SCHEMAS}

### Removed Schemas

${REMOVED_SCHEMAS}

## Method Diff

${METHOD_DIFF}

## Sync Control Plane

This issue is state and audit evidence. Routine scheduled sync does not wait for a maintainer to act on this issue.

When the upstream protocol sync workflow finds `review-required` drift outside `force_compare` verification, it uses this analysis to publish a protected sync PR automatically. The PR body includes the drift analysis above plus the fix description.

The sync workflow regenerates the candidate, applies it locally, runs the bounded Codex `repair-applied-candidate` command, validates, commits the sync, and publishes a protected PR. It does not push `main`, tag, close this issue as resolved, or decide whether the PR should merge.

Issue metadata records the upstream target and drift fingerprint for audit. Workflow code, PR base, and finalize refs are constrained to the repository default branch.

After the sync PR lands through branch protection, the post-merge finalize trigger should pick it up automatically. Scheduled finalize runs are the fallback. To recover manually, dispatch finalize with:

```sh
gh workflow run upstream-protocol-finalize.yml \
  -f pr_number=<merged sync PR number> \
  -f issue_number=<this issue number>
```

Finalize verifies the landed commit, creates the stable sync tag when applicable, dispatches drift verification, and closes or updates this issue based on that verification result.

Sync PRs created by `GITHUB_TOKEN` may require one maintainer approval or rerun before GitHub schedules the required `Go` pull request check. If the first PR CI run is `action_required` with no jobs, rerun that run once; protected branch rules and repository auto-merge rules should continue after the real `Go` check passes.
