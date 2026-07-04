# Command: review-drift

Use when:
- Reviewing compact drift artifacts before changing the checked-in schema baseline.

Inputs:
- Candidate artifact directory, target provenance, and optional upstream response mapping evidence.

Read:
- Top-level skill contract and invariants in `../SKILL.md`.
- `../references/local-sync.md`, "Generate And Review Drift" and "Decision Rules".
- Candidate `reports/SUMMARY.md`, `reports/drift_summary.json`, and `reports/matrix_update_skeleton.json`.

Allowed side effects:
- May write a concise local review note if explicitly requested.

Forbidden side effects:
- Do not copy schemas into the checked-in baseline.
- Do not regenerate generated Go.
- Do not edit manifest, coverage, SDK code, reports, branches, commits, PRs, tags, or issues unless explicitly asked.

Procedure:
- Inspect compact drift summaries, method deltas, schema file changes, and matrix update skeleton.
- Check manifest, coverage, and response mapping implications.
- Classify SDK impact as metadata-only, generated-only, public-facade-required, ignored-internal, policy/blocker, or ambiguous.
- Decide whether the candidate is safe for mechanical apply or needs repair/recovery first.

Success means:
- A concise review decision is available: `clean`, `mechanical-only`, `repair-needed`, `policy/blocker`, or `ambiguous`.

Validation:
- Review cites the drift evidence that drives the decision.
- Public SDK or coverage recommendations are tied to reviewed drift, not speculation.

Final output:
- Decision, short rationale, files or surfaces implicated, and next command to run.

Stop rules:
- Stop on missing compact reports.
- Stop on candidate provenance mismatch.
- Stop with `ambiguous` when evidence is insufficient to justify checked-in changes.
