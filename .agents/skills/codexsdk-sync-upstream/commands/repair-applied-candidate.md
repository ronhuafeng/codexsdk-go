# Command: repair-applied-candidate

State:
- Detect and mechanical apply already completed; caller needs bounded local repair or confirmation that no repair is needed.

Inputs:
- Target ref, ref kind, target SHA, and landing ref.
- Candidate `schema/`, `reports/`, and `common.rs`.
- Apply result and diff name-status.

Boundaries:
- Treat candidate artifacts and apply output as authoritative.
- Choose the smallest compact evidence and focused files needed for the repair decision.
- May edit concrete local issues in generated Go, generator behavior, manifest/coverage classification, or small handwritten SDK surfaces when justified by reviewed drift.
- May run focused checks for changed behavior.
- Must not rerun target resolution, policy, tracking, full Rust schema generation, or candidate apply.
- Must not re-copy schemas, commit, push, tag, edit issues, create PRs, change branches, request merges, or publish remote state.

Checks:
- Run focused generator, manifest, coverage, gofmt, or package tests when practical and relevant to touched files.
- The caller/workflow may still run full `scripts/codexsdk_validate_sync.sh`.

Output:
- `completed_actions`
- `files_changed`
- `validation`
- `blockers`
- `highest local completion layer`

Stop if:
- Required candidate artifacts or apply summaries are missing.
- Evidence points to unresolved policy/provenance mismatch from earlier commands.
- Repair would require broad regeneration or remote side effects.
