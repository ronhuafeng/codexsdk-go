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

## Automated Sync

This workflow will attempt a guarded Codex sync, rebase it onto the landing ref, validate it, publish a sync PR, run the required `Go` check on the PR head, and merge through the protected-branch PR path.

Sync PRs created by `GITHUB_TOKEN` may require one maintainer approval or rerun before GitHub schedules the required `Go` pull request check. If the first PR CI run is `action_required` with no jobs, rerun that run once; auto-merge should continue after the real `Go` check passes.
