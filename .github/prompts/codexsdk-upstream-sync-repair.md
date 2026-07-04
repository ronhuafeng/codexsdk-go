You are maintaining codexsdk-go.

Task: Use codexsdk-sync-upstream command: repair-applied-candidate.

Read and follow:

- .agents/skills/codexsdk-sync-upstream/SKILL.md
- .agents/skills/codexsdk-sync-upstream/commands/repair-applied-candidate.md

Current state:

- The active command is `repair-applied-candidate`.
- Detect and apply have already completed.
- The workflow has resolved the upstream target, generated the drift candidate, and applied the mechanical sync with `scripts/codexsdk_apply_sync_candidate.py`.
- Do not follow a global sync workflow; stay inside the repair command boundary and use the shortest safe path for the evidence in this checkout.

- upstream ref: ${UPSTREAM_REF}
- upstream ref kind: ${UPSTREAM_REF_KIND}
- upstream commit: ${UPSTREAM_SHA}
- landing ref: ${LAND_REF}

Authoritative inputs:

- ${CANDIDATE_DIR}/schema
- ${CANDIDATE_DIR}/reports
- ${CANDIDATE_DIR}/common.rs
- ${AUTO_SYNC_DIR}/apply-result.json
- ${AUTO_SYNC_DIR}/diff-name-status.txt

Success criteria:

- Choose the compact drift evidence, apply summary details, diff entries, and focused code paths needed to decide whether local repair is required.
- Fix concrete local issues in generated Go, generator behavior, manifest/coverage classification, or small handwritten SDK surfaces when justified by reviewed drift and inside this command boundary.
- If no additional repair is needed, say that clearly.
- Keep handwritten SDK changes minimal and tied to reviewed drift.

Validation expectations:

- Run the smallest useful checks for changed behavior when practical; prefer focused generator, manifest, coverage, gofmt, or package tests that match the files you touched.
- The workflow runs `scripts/codexsdk_validate_sync.sh` after you finish.
- Keep command output compact and do not print full schema, report, manifest, coverage, or generated Go files.
- The workflow owns validate, commit, PR publication, CI, merge, tags, issues, and remote verification after you finish.

Do not run `resolve-target`, `detect-drift`, `scripts/codexsdk_track_upstream.sh`, full Rust schema generation, or `apply-candidate`. Do not re-copy schemas from upstream. Do not commit, push, tag, edit issues, create PRs, change branches, request merges, or publish remote state.

Final output must include:

- completed_actions
- files_changed
- validation
- blockers
- highest local completion layer
