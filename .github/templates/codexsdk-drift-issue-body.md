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

This issue is state and audit evidence, not the only control plane.

To create a sync PR, explicitly dispatch the fix workflow with this issue number or with the upstream target plus drift fingerprint:

```sh
gh workflow run upstream-protocol-auto-sync.yml \
  -f issue_number=<this issue number> \
  -f upstream_ref=${UPSTREAM_REF} \
  -f upstream_sha=${UPSTREAM_SHA} \
  -f drift_sha=${DRIFT_SHA}
```

The fix workflow regenerates or verifies the candidate, applies it locally, runs the bounded Codex `repair-applied-candidate` command, validates, and publishes a protected PR. It does not push `main`, tag, close this issue as resolved, or decide whether the PR should merge.

Issue metadata controls the upstream target and drift fingerprint only. Workflow code, PR base, and finalize refs are constrained to the repository default branch.

After the sync PR lands through branch protection, dispatch the finalize workflow:

```sh
gh workflow run upstream-protocol-finalize.yml \
  -f pr_number=<merged sync PR number> \
  -f issue_number=<this issue number>
```

Finalize verifies the landed commit, creates the stable sync tag when applicable, dispatches drift verification, and closes or updates this issue based on that verification result.

Sync PRs created by `GITHUB_TOKEN` may require one maintainer approval or rerun before GitHub schedules the required `Go` pull request check. If the first PR CI run is `action_required` with no jobs, rerun that run once; protected branch rules and repository auto-merge rules should continue after the real `Go` check passes.
