# Command: resolve-target

State:
- Caller needs the selected upstream target before drift generation or sync mutation.

Inputs:
- Requested tag, ref, full commit SHA, or no target for latest stable.
- Baseline metadata path, usually `codexsdk/internal/protocolschema/appserver/v2/baseline_metadata.json`.

Tool:
- `scripts/codexsdk_resolve_upstream.py`

Boundaries:
- May write a local target JSON file.
- May perform resolver-owned network reads.
- Must not clone upstream, generate drift, mutate checked-in files, commit, push, tag, create issues, or create PRs.

Checks:
- Resolver JSON parses.
- Peeled SHA is a full commit SHA.
- Ref kind is resolver-supported.

Output:
- Target ref, ref kind, target commit SHA, explicit/default status, and current baseline ref/commit.

Stop if:
- Target is ambiguous or resolver output lacks required provenance.
