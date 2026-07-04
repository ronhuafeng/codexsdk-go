# Command: recover-failure

State handled:
- A sync PR check, merge wait, finalize step, drift verification, or sync tag operation failed and the caller needs a narrow recovery path.

Trusted inputs:
- Failure type, PR number or branch when relevant, CI/run ID when relevant, target provenance, and landed commit when finalize has started.

Read:
- Top-level skill contract and invariants in `../SKILL.md`.
- `../references/recovery.md`.

Fixed tools:
- GitHub CLI/API inspection for PRs, checks, runs, tags, and issues.
- Documented recovery recipes in `../references/recovery.md`.
- `scripts/codexsdk_sync_tag.py` for tag conflict recovery when a suffix tag is allowed.

Allowed side effects:
- May inspect GitHub PRs, checks, CI runs, tags, and issues.
- May rerun an expected `action_required` `GITHUB_TOKEN` PR check when the user or caller owns that recovery.
- May create documented follow-up commits or suffix sync tags only through the protected recipes.

Forbidden side effects:
- Do not weaken branch protection.
- Do not bypass required checks, introduce PATs or app tokens, synthesize statuses, force-push `main`, or delete/move tags.
- Do not add new automation before trying the recovery recipes.

Shortest safe path:
- Classify the failed state from available evidence and the recovery reference.
- Preserve useful failure evidence before reruns or cleanup.
- Apply the narrow documented recipe for that failure class.
- Resume with another command only after the state and allowed side effects are clear.

Success means:
- The failed state is recovered or reduced to an explicit blocker with the next safe command identified.

Validation:
- Recovery action stays on the protected PR path.
- Required real checks remain authoritative.
- Tags and branch protection are unchanged except for documented allowed actions.

Final output:
- Failure classification, evidence collected, recovery action taken, remaining blockers, next command, and completion layer.

Stop rules:
- Stop if recovery would require bypassing protection or moving/deleting tags.
- Stop if evidence is insufficient to identify the failure class.
- Stop if the same blocker persists after documented recovery steps.
