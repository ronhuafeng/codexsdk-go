# Command: detect-drift

State:
- Caller has a resolved target and needs policy plus local drift artifacts.

Inputs:
- Target ref, ref kind, target SHA, target explicit/default status, policy mode, and downgrade policy.
- Upstream Codex repo path and output directory.

Tools:
- `scripts/codexsdk_target_policy.py`
- `scripts/codexsdk_track_upstream.sh`

Boundaries:
- Run policy before drift generation.
- May write policy output and local drift artifacts.
- May let tracking fetch the selected target narrowly after policy allows.
- Must not apply a candidate, mutate checked-in sync files, commit, push, tag, create PRs, or touch issues unless the caller explicitly owns issue side effects.

Checks:
- Policy JSON parses and has decision plus reason.
- On `allow`, compact reports include `SUMMARY.md`, `drift_summary.json`, and `matrix_update_skeleton.json`.
- Checked-in baseline files remain unchanged.

Output:
- Policy decision, reason, drift status, target provenance, artifact directory, and any caller-owned issue action.

Stop if:
- Policy returns `block` or `skip`.
- Candidate provenance is missing or drift generation fails.
