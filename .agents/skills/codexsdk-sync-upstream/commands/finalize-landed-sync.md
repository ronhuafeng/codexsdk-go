# Command: finalize-landed-sync

State:
- Sync PR landed and caller-owned remote completion may need tagging or drift verification.

Inputs:
- Landed ref, landed commit, sync PR, upstream target ref/kind/SHA, and whether caller requested drift verification.

Tools:
- `scripts/codexsdk_sync_tag.py`
- Caller-owned workflow dispatch or CI tooling for drift verification.

Boundaries:
- May create stable upstream sync tags only through the sync tag script.
- May run drift verification only when explicitly requested.
- Must not delete/move tags, tag unmerged PR heads or failed attempts, or call drift fully resolved before CI, tag, drift verification, and issue closure are complete when applicable.

Checks:
- Tag, if created, points at the landed commit.
- Drift verification result is known when dispatched.
- Existing tags were not moved or deleted.

Output:
- Landed ref, landed commit, sync PR, upstream target, upstream commit, sync tag when present, drift verification result, and completion layer.

Stop if:
- PR has not landed or landed commit cannot be verified.
- Target is manual ref/commit and tagging would be attempted.
- Tag conflict cannot use the documented suffix path.
