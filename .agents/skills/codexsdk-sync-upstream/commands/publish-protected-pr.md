# Command: publish-protected-pr

Use when:
- Publishing a validated local sync through the protected PR path.

Inputs:
- Landing ref, sync branch prefix, target ref, target commit SHA, candidate schema path, and explicit user or caller request to publish.

Read:
- Top-level skill contract and invariants in `../SKILL.md`.
- `../references/recovery.md` when publishing is recovering or continuing a failed remote state.

Allowed side effects:
- May create or update a sync branch and PR through `scripts/codexsdk_publish_sync_pr.sh`.
- May request protected PR auto-merge when the caller explicitly owns that step.

Forbidden side effects:
- Do not push directly to protected `main`.
- Do not introduce PATs, GitHub App tokens, bot-token bypasses, or synthetic required statuses.
- Do not tag, edit drift issues as fully resolved, delete branches, or merge around failed checks.

Procedure:
- Confirm publishing was explicitly requested and local validation status is known.
- Run `scripts/codexsdk_publish_sync_pr.sh` with the target and candidate inputs.
- Preserve `action_required` as the expected maintainer rerun/approval gate for `GITHUB_TOKEN` PRs.
- Report that remote completion is only `commit pushed` until protected merge, tag, CI, drift verification, and issue closure finish.

Success means:
- A sync branch and PR exist on the protected PR path, or the command stops before unsafe publishing.

Validation:
- Published branch is not protected `main`.
- PR targets the landing ref and uses real required checks.
- No synthetic statuses or bypass tokens were introduced.

Final output:
- Sync branch, PR URL or number, pushed commit when known, `action_required` status when relevant, and completion layer.

Stop rules:
- Stop if publishing was not explicitly requested.
- Stop if local validation has failed or is unknown and the caller cannot justify proceeding.
- Stop if branch protection would be bypassed.
