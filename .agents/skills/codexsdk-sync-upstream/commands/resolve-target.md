# Command: resolve-target

State handled:
- The caller needs the selected upstream Codex target and current baseline provenance before any drift generation or sync mutation.

Trusted inputs:
- Requested upstream tag, ref, or full commit SHA, or no target when latest stable is intended.
- Baseline metadata path, defaulting to `codexsdk/internal/protocolschema/appserver/v2/baseline_metadata.json`.

Read:
- Top-level skill contract and invariants in `../SKILL.md`.
- `../references/local-sync.md`, "Resolve Target And Policy".

Fixed tools:
- `scripts/codexsdk_resolve_upstream.py` for latest-stable selection, ref lookup, tag peeling, and JSON output.

Allowed side effects:
- May write a local target JSON file such as `.cache/codexsdk-upstream-target.json`.
- May perform resolver-owned network reads needed by `scripts/codexsdk_resolve_upstream.py`.

Forbidden side effects:
- Do not clone upstream.
- Do not generate drift.
- Do not mutate checked-in schemas, reports, generated Go, branches, tags, issues, PRs, or commits.

Shortest safe path:
- Inspect only enough baseline metadata to report existing provenance.
- Use the resolver instead of ad hoc shell logic for target selection or tag peeling.
- Capture the resolved target fields needed by later commands; stop before any drift or baseline mutation.

Success means:
- Target ref, ref kind, peeled full SHA, explicit/default status, and baseline metadata are known.

Validation:
- Resolver JSON parses.
- Peeled SHA is a full commit SHA.
- Ref kind is one of the resolver-supported values.

Final output:
- Target ref, target ref kind, target commit SHA, explicit/default status, and current baseline ref/commit.

Stop rules:
- Stop before any mutation if the target cannot be inferred from the request, latest stable tag, or local context.
- Stop if resolver output is missing provenance fields or the target is ambiguous.
