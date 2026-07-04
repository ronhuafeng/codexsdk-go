You are maintaining codexsdk-go.

Read and follow .agents/skills/codexsdk-sync-upstream/SKILL.md completely before editing.

This workflow has already applied the large mechanical sync with scripts/codexsdk_apply_sync_candidate.py. Perform a bounded maintainer review of the resulting worktree for this already-resolved upstream target:

- upstream ref: ${UPSTREAM_REF}
- upstream ref kind: ${UPSTREAM_REF_KIND}
- upstream commit: ${UPSTREAM_SHA}
- landing ref: ${LAND_REF}

Candidate schema and drift artifacts from this workflow's detect job are already available under:

- ${CANDIDATE_DIR}/schema
- ${CANDIDATE_DIR}/reports
- ${CANDIDATE_DIR}/common.rs
- ${CANDIDATE_DIR}/common.rs.source_sha

Mechanical apply summary:

- ${AUTO_SYNC_DIR}/apply-result.json
- ${AUTO_SYNC_DIR}/diff-name-status.txt

Inspect the apply summary, compact drift reports, and git diff name-status. Confirm there is no obvious provenance, sanitization, manifest, coverage, or generated-code issue. Do not re-copy schemas, re-run the full Rust schema generator, or print large files. Do not try to manually review every generated JSON or Go line.

Leave the current worktree changes in place. Make only small, clearly necessary fixes if the review finds a concrete issue. Do not perform repository side effects. The workflow will validate, commit, publish a PR, dispatch the required CI on the PR head, and merge through the protected-branch PR path after you finish.

Keep command output compact. Do not print full schema, report, manifest, coverage, or generated Go files. Avoid chained shell commands because the action sandbox may reject them; run separate focused commands instead.

Keep handwritten SDK changes minimal and justify them through reviewed drift. Leave newly discovered experimental or internal upstream surface in generated protocolv2 unless the manifest rules require public SDK exposure.
