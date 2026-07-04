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
- May create, update, close, or dispatch follow-up workflow state only when the caller explicitly owns those side effects.
- Must not apply a candidate, mutate checked-in sync files, commit, push, tag, create PRs, or touch issues unless the caller explicitly owns issue side effects.
- Must not rely on a `GITHUB_TOKEN` issue event as the control plane for fixes.
- When dispatching a follow-up workflow, use the trusted default-branch workflow ref rather than a ref from issue metadata or the current issue state.

Checks:
- Policy JSON parses and has decision plus reason.
- On `allow`, compact reports include `SUMMARY.md`, `drift_summary.json`, and `matrix_update_skeleton.json`.
- Issue or artifact evidence records upstream ref, upstream SHA, drift fingerprint, and run URL when caller-owned automation updates remote state.
- Checked-in baseline files remain unchanged.

Output:
- Policy decision, reason, drift status, drift fingerprint, target provenance, artifact directory, run URL, and any caller-owned issue or dispatch action.

Stop if:
- Policy returns `block` or `skip`.
- Candidate provenance is missing or drift generation fails.
