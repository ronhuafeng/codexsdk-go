You are maintaining codexsdk-go.

Task: Use codexsdk-sync-upstream command: repair-applied-candidate.

Read and follow:

- .agents/skills/codexsdk-sync-upstream/SKILL.md
- .agents/skills/codexsdk-sync-upstream/commands/repair-applied-candidate.md

Detect and apply have already completed. The workflow has resolved the upstream target, generated the drift candidate, and applied the mechanical sync with scripts/codexsdk_apply_sync_candidate.py:

- upstream ref: ${UPSTREAM_REF}
- upstream ref kind: ${UPSTREAM_REF_KIND}
- upstream commit: ${UPSTREAM_SHA}
- landing ref: ${LAND_REF}

Treat candidate artifacts and mechanical apply output as authoritative inputs:

- ${CANDIDATE_DIR}/schema
- ${CANDIDATE_DIR}/reports
- ${CANDIDATE_DIR}/common.rs
- ${AUTO_SYNC_DIR}/apply-result.json
- ${AUTO_SYNC_DIR}/diff-name-status.txt

Success criteria:

- Inspect drift evidence, apply summary, diff name-status, and focused code paths implicated by the drift.
- Fix concrete local issues in generated Go, generator behavior, manifest/coverage classification, or small handwritten SDK surfaces when justified by reviewed drift.
- If no additional repair is needed, say that clearly.
- Keep handwritten SDK changes minimal and tied to reviewed drift.

Validation expectations:

- Run the smallest useful checks for changed behavior when practical.
- The workflow runs scripts/codexsdk_validate_sync.sh after you finish.
- Keep command output compact and do not print full schema, report, manifest, coverage, or generated Go files.
- The workflow owns validate, commit, PR publication, CI, merge, tags, issues, and remote verification after you finish.

Do not run resolve-target, detect-drift, track-upstream, full Rust schema generation, or apply-candidate. Do not re-copy schemas from upstream. Do not commit, push, tag, edit issues, create PRs, change branches, request merges, or publish remote state.

Final output must include:

- completed_actions
- files_changed
- validation
- blockers
- highest local completion layer
