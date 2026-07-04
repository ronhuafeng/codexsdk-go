# Command: validate-local

Use when:
- Validating local checked-in sync state before commit, publish, or final report.

Inputs:
- Candidate schema directory when available.
- Target commit SHA and current checked-in baseline.

Read:
- Top-level skill contract and invariants in `../SKILL.md`.
- `../references/local-sync.md`, "Validate".

Allowed side effects:
- May run `scripts/codexsdk_validate_sync.sh`.
- May run targeted tests, `gofmt`, generator reproduction checks, sync-state checks, and path sanitization checks.

Forbidden side effects:
- Do not publish remote state.
- Do not commit, push, tag, edit issues, create PRs, request merges, or change branches.

Procedure:
- Run `scripts/codexsdk_validate_sync.sh` with the candidate and target SHA when available.
- Run any focused checks needed for files changed during repair.
- Inspect validation failures before adding code or changing classification.

Success means:
- Local validation passes, or blockers are explicit and actionable.

Validation:
- Report exact checks run and their pass/fail result.
- Confirm no local absolute paths or cache output paths leaked into checked-in reports.

Final output:
- Validation result, commands run, blockers if any, and highest completed local layer.

Stop rules:
- Stop on failing validation and report the first actionable failure.
- Stop if validation inputs do not match the current baseline provenance.
