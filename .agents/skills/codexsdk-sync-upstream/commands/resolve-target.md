# Command: resolve-target

Use when:
- Resolving the upstream Codex target before drift generation or sync work.

Inputs:
- Requested upstream tag, ref, or full commit SHA, or no target for latest stable.
- Baseline metadata path, defaulting to `codexsdk/internal/protocolschema/appserver/v2/baseline_metadata.json`.

Read:
- Top-level skill contract and invariants in `../SKILL.md`.
- `../references/local-sync.md`, "Resolve Target And Policy".

Allowed side effects:
- May write a local target JSON file such as `.cache/codexsdk-upstream-target.json`.
- May perform resolver-owned network reads needed by `scripts/codexsdk_resolve_upstream.py`.

Forbidden side effects:
- Do not clone upstream.
- Do not generate drift.
- Do not mutate checked-in schemas, reports, generated Go, branches, tags, issues, PRs, or commits.

Procedure:
- Inspect current baseline metadata enough to report the existing provenance.
- Run `scripts/codexsdk_resolve_upstream.py` with the explicit target or latest-stable default.
- Capture `upstream_ref`, `upstream_ref_kind`, `upstream_sha`, tag SHA when present, and `target_explicit`.

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
