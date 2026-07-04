# Command: validate-local

State handled:
- The checked-in sync surface has local changes and needs validation before commit, publish, or final report.

Trusted inputs:
- Candidate schema directory when available.
- Target commit SHA and current checked-in baseline.

Read:
- Top-level skill contract and invariants in `../SKILL.md`.
- `../references/local-sync.md`, "Validate".

Fixed tools:
- `scripts/codexsdk_validate_sync.sh` for the canonical local sync validation path.
- Focused tests, `gofmt`, generator reproduction checks, sync-state checks, and path sanitization checks when they fit the changed files.

Allowed side effects:
- May run `scripts/codexsdk_validate_sync.sh`.
- May run targeted tests, `gofmt`, generator reproduction checks, sync-state checks, and path sanitization checks.

Forbidden side effects:
- Do not publish remote state.
- Do not commit, push, tag, edit issues, create PRs, request merges, or change branches.

Shortest safe path:
- Prefer the canonical validation script when candidate and target inputs are available.
- Add focused checks for files changed during repair or for the first actionable failure.
- Inspect validation failures before adding code or changing manifest, coverage, or SDK classification.

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
