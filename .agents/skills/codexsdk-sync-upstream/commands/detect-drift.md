# Command: detect-drift

State handled:
- The caller has a resolved upstream target and needs target policy plus local drift artifacts when policy allows.

Trusted inputs:
- Target ref, target ref kind, target commit SHA, explicit/default status, policy mode, and downgrade policy.
- Local upstream Codex repo path and output directory.

Read:
- Top-level skill contract and invariants in `../SKILL.md`.
- `../references/local-sync.md`, "Resolve Target And Policy" and "Generate And Review Drift".

Fixed tools:
- `scripts/codexsdk_target_policy.py` for allow/skip/block decisions.
- `scripts/codexsdk_track_upstream.sh` for drift candidate and compact report generation.

Allowed side effects:
- May write target policy output and local drift artifacts.
- May let `scripts/codexsdk_track_upstream.sh` fetch the selected target narrowly after policy allows.
- May create/update/close drift issues only when the caller explicitly owns issue side effects.

Forbidden side effects:
- Do not run drift generation before `scripts/codexsdk_target_policy.py` allows it.
- Do not apply a candidate into the checked-in baseline.
- Do not mutate generated Go, manifest, coverage, branches, commits, PRs, tags, or issues outside a caller-owned issue step.

Shortest safe path:
- Run policy before any drift generation and accept `block`, `skip`, or `allow` as command state, not advice to override.
- On `allow`, use the tracking script to produce candidate artifacts for the resolved commit and provenance.
- Preserve compact evidence for later review; do not decide SDK repair or apply checked-in changes in this command.

Success means:
- Policy decision is known and, when allowed, drift status and candidate artifacts are available.

Validation:
- Policy JSON parses and includes a decision and reason.
- For `allow`, drift reports include `SUMMARY.md`, `drift_summary.json`, and `matrix_update_skeleton.json`.
- Checked-in baseline files remain unchanged.

Final output:
- Policy decision, policy reason, drift status, target provenance, artifact directory, and any caller-owned issue action.

Stop rules:
- Stop on policy `block` or malformed policy output.
- Stop if candidate provenance cannot be tied to the resolved target.
- Stop if drift generation fails or produces incomplete reports.
