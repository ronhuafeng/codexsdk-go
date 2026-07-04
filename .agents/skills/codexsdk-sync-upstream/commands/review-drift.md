# Command: review-drift

State handled:
- Compact drift artifacts exist and the caller needs an evidence-based decision before checked-in baseline changes.

Trusted inputs:
- Candidate artifact directory, target provenance, and optional upstream response mapping evidence.

Read:
- Top-level skill contract and invariants in `../SKILL.md`.
- `../references/local-sync.md`, "Generate And Review Drift" and "Decision Rules".
- Candidate `reports/SUMMARY.md`, `reports/drift_summary.json`, and `reports/matrix_update_skeleton.json`.

Fixed tools:
- Structured readers such as `jq`, Python JSON parsing, or repo scripts may be used for compact reports.
- Do not use candidate apply or generation scripts in this command.

Allowed side effects:
- May write a concise local review note if explicitly requested.

Forbidden side effects:
- Do not copy schemas into the checked-in baseline.
- Do not regenerate generated Go.
- Do not edit manifest, coverage, SDK code, reports, branches, commits, PRs, tags, or issues unless explicitly asked.

Shortest safe path:
- Read only the compact evidence needed to classify the candidate.
- Tie manifest, coverage, response mapping, and public SDK conclusions to reviewed drift rather than speculation.
- Classify the next safe command without applying or repairing files here.

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
