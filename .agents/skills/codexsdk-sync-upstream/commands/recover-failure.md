# Command: recover-failure

Use when:
- Recovering failed sync PR checks, auto-merge timeouts, finalize failures, drift verification failures, or existing sync tag conflicts.

Inputs:
- Failure type, PR number or branch when relevant, CI/run ID when relevant, target provenance, and landed commit when finalize has started.

Read:
- Top-level skill contract and invariants in `../SKILL.md`.
- `../references/recovery.md`.

Allowed side effects:
- May inspect GitHub PRs, checks, CI runs, tags, and issues.
- May rerun an expected `action_required` `GITHUB_TOKEN` PR check when the user or caller owns that recovery.
- May create documented follow-up commits or suffix sync tags only through the protected recipes.

Forbidden side effects:
- Do not weaken branch protection.
- Do not bypass required checks, introduce PATs or app tokens, synthesize statuses, force-push `main`, or delete/move tags.
- Do not add new automation before trying the recovery recipes.

Procedure:
- Classify the failure using `../references/recovery.md`.
- Preserve useful failure evidence before reruns or cleanup.
- Apply the narrow recovery recipe for the classified failure.
- Resume with `validate-local`, `publish-protected-pr`, or `finalize-landed-sync` only after the state is clear.

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
