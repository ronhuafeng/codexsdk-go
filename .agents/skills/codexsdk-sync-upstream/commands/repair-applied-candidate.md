# Command: repair-applied-candidate

State handled:
- Detect and mechanical apply have already completed, and the caller needs a bounded local repair pass for concrete issues exposed by reviewed drift or validation.

Trusted inputs:
- Target ref, target ref kind, target commit SHA, and landing ref.
- Candidate artifact directory containing `schema/`, `reports/`, and `common.rs`.
- Mechanical apply summary and diff name-status.

Read:
- Top-level skill contract and invariants in `../SKILL.md`.
- Caller-provided side-effect boundaries, if stricter than this command.
- Candidate compact reports, apply result, diff name-status, and focused code paths implicated by reviewed drift.

Fixed tools:
- Focused repo tests, generator checks, manifest/coverage checks, `gofmt`, and JSON readers as needed.
- `scripts/codexsdk_validate_sync.sh` may be left to the caller or workflow unless focused local validation is useful during repair.

Allowed side effects:
- May edit concrete local issues in generated Go, generator behavior, manifest/coverage classification, or small handwritten SDK surfaces justified by reviewed drift.
- May run focused validation for changed behavior when practical.

Forbidden side effects:
- Do not rerun `resolve-target`.
- Do not rerun `detect-drift`, `scripts/codexsdk_track_upstream.sh`, full Rust schema generation, or `apply-candidate`.
- Do not re-copy schemas from upstream.
- Do not commit, push, tag, edit issues, create PRs, change branches, request merges, or publish remote state.

Shortest safe path:
- Treat candidate artifacts and mechanical apply output as authoritative; do not restart earlier commands.
- Read the smallest set of compact evidence and focused files needed to explain or repair the current issue.
- Fix concrete local issues rather than producing passive review when the fix is clear and inside this boundary.
- Keep handwritten SDK changes minimal and tied to reviewed drift.
- Leave experimental or internal upstream surface in generated `protocolv2` unless manifest rules require public SDK exposure.

Success means:
- The applied candidate has either been locally repaired or confirmed to need no extra repair, with blockers called out.

Validation:
- Run the smallest useful checks for changed behavior when practical.
- Prefer focused generator, manifest, coverage, gofmt, or package tests during iteration.
- The caller may still run full `scripts/codexsdk_validate_sync.sh` afterward.

Final output:
- `completed_actions`: concrete inspection and repair actions.
- `files_changed`: files changed by this command, or `none`.
- `validation`: checks run and results.
- `blockers`: remaining blockers, or `none`.
- `highest local completion layer`: usually `local repair complete, validation pending` or `local sync complete`.

Stop rules:
- Stop if required candidate artifacts or apply summaries are missing.
- Stop if evidence points to an unresolved policy/provenance mismatch from earlier commands.
- Stop before remote side effects or broad regeneration.
