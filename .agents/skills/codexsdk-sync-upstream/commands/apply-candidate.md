# Command: apply-candidate

State:
- Reviewed candidate is ready for mechanical application into the checked-in baseline.

Inputs:
- Candidate schema directory, reports directory, upstream `common.rs`, target ref, ref kind, and target SHA.

Tool:
- `scripts/codexsdk_apply_sync_candidate.py`

Boundaries:
- May update checked-in schemas, metadata, clean drift reports, manifest, coverage, generated `protocolv2` Go, and SDK surface through the apply script.
- May write a local apply summary and diff name-status evidence.
- Must not hand-copy schemas or reports.
- Must not make judgment calls beyond provenance/input checks.
- Must not commit, push, tag, create/edit issues, create PRs, request merges, or change branches.

Checks:
- Apply result JSON parses.
- Changed files stay inside the allowed sync surface.
- Candidate provenance still matches target.

Output:
- Apply summary path or key findings, diff name-status path or summary, and whether follow-up repair is needed.

Stop if:
- Candidate artifacts or provenance are missing/mismatched.
- Apply script fails or wants to touch files outside the allowed sync surface.
