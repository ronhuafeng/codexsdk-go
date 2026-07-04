# Recovery Reference

Use these recipes before adding more automation. Keep recovery actions on the protected PR path, and do not delete tags, delete branches, write synthetic required statuses, force-push `main`, or bypass branch protection.

## Sync PR `Go` Check Is `action_required`

This is expected for sync PRs created by `GITHUB_TOKEN`. Confirm the first run has no jobs, then have a maintainer with write access approve or rerun it once:

```sh
gh run view <run-id> --attempt 1 --json conclusion,jobs,actor,triggeringActor
gh run rerun <run-id>
```

After the real `Go` check passes, auto-merge should continue. Do not manually merge just to bypass this gate.

## Sync PR Does Not Merge Before Timeout

Inspect the PR state, required checks, and recent CI runs on the sync branch:

```sh
gh pr view <pr-number> --json state,mergeStateStatus,reviewDecision,statusCheckRollup
gh pr checks <pr-number>
gh run list --branch <sync-branch> --limit 10 \
  --json databaseId,event,status,conclusion,headSha,url
```

If the required `Go` run is `action_required`, rerun it once as above. If a real check failed, inspect logs and fix the sync branch with a normal follow-up commit or rerun the caller-owned automation; do not force a merge around the failed required check.

## Finalize Failed After PR Merged

If the sync PR merged but tag creation or drift dispatch failed, recover from the landed `main` commit:

```sh
git fetch origin main --tags
git checkout --detach origin/main
python3 scripts/codexsdk_sync_tag.py --json
python3 scripts/codexsdk_sync_tag.py --create --push origin
```

If the base upstream sync tag already exists at another commit, use the suffix path instead:

```sh
python3 scripts/codexsdk_sync_tag.py --next-suffix --create --push origin
```

Then run the caller-owned drift verification path for the landed baseline target and watch it to completion.

## Drift Verification Fails After Main CI Passes

Treat this as incomplete until the cause is understood. Check whether landed `baseline_metadata.json` still matches the intended upstream target, then inspect drift run logs.

If the failure is transient, rerun the drift verification path. If drift is real, keep or update the protocol-drift issue and run another sync. Do not report `drift issue fully resolved`.

## Sync Tag Already Exists

The base tag `upstream-codex-rust-vX.Y.Z` is used for the first landed sync to a stable upstream tag. Follow-up syncs to the same upstream tag must use `-sync.N` through:

```sh
python3 scripts/codexsdk_sync_tag.py --next-suffix --create --push origin
```

Never move or delete an existing upstream sync tag.
