# Command: review-drift

State:
- Compact drift artifacts exist and caller needs a decision before checked-in baseline changes.

Inputs:
- Candidate artifact directory, target provenance, optional upstream response mapping evidence.

Evidence:
- `reports/SUMMARY.md`
- `reports/drift_summary.json`
- `reports/matrix_update_skeleton.json`

Boundaries:
- Read only enough compact evidence to classify the candidate.
- Tie manifest, coverage, response mapping, and public SDK conclusions to reviewed drift.
- Must not copy schemas, regenerate Go, edit manifest/coverage/SDK files, commit, push, tag, create PRs, or touch issues unless explicitly asked.

Checks:
- Decision cites the drift evidence that drives it.
- Public SDK or coverage recommendations are evidence-backed, not speculative.

Output:
- Decision: `clean`, `mechanical-only`, `repair-needed`, `policy/blocker`, or `ambiguous`.
- Short rationale, implicated surfaces, and next command.

Stop if:
- Compact reports are missing.
- Candidate provenance mismatches target.
- Evidence is insufficient; return `ambiguous`.
