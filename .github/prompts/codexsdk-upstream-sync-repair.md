You are maintaining codexsdk-go.

Task: Use codexsdk-sync-upstream command: repair-applied-candidate.

Read and follow:

- .agents/skills/codexsdk-sync-upstream/SKILL.md
- .agents/skills/codexsdk-sync-upstream/commands/repair-applied-candidate.md

Current state: Detect and apply have already completed. The workflow resolved `${UPSTREAM_REF}` (`${UPSTREAM_REF_KIND}`) to `${UPSTREAM_SHA}`, generated the candidate, and applied it to `${LAND_REF}` with `scripts/codexsdk_apply_sync_candidate.py`.

Authoritative inputs:

- ${CANDIDATE_DIR}/schema
- ${CANDIDATE_DIR}/reports
- ${CANDIDATE_DIR}/common.rs
- ${AUTO_SYNC_DIR}/apply-result.json
- ${AUTO_SYNC_DIR}/diff-name-status.txt

Do not follow a global sync workflow; stay inside the repair command boundary and use the shortest safe path for the evidence in this checkout.

Do not run `resolve-target`, `detect-drift`, `scripts/codexsdk_track_upstream.sh`, full Rust schema generation, or `apply-candidate`. Do not re-copy schemas from upstream. Do not commit, push, tag, edit issues, create PRs, change branches, request merges, or publish remote state.

The workflow runs full validation, commit, PR publication, CI, merge, tags, issues, and remote verification after you finish. Run only focused checks that are useful for files you touch.

Final output must include: completed_actions, files_changed, validation, blockers, highest local completion layer.
