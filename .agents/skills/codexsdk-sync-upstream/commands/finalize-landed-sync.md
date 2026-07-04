# Command: finalize-landed-sync

Use when:
- A sync PR has landed and remote completion needs tagging or drift verification.

Inputs:
- Landed ref, landed commit, sync PR, upstream target ref, upstream ref kind, upstream commit, and whether to dispatch drift verification.

Read:
- Top-level skill contract and invariants in `../SKILL.md`.
- `../references/local-sync.md`, "Tag Landed Syncs".

Allowed side effects:
- May create stable upstream sync tags only through `scripts/codexsdk_sync_tag.py`.
- May run a caller-provided drift verification step when explicitly requested.

Forbidden side effects:
- Do not delete or move existing tags.
- Do not tag unmerged PR heads, failed attempts, drift-check-only work, or uncommitted changes.
- Do not call a drift issue fully resolved before CI, tag, drift verification, and issue closure are complete when applicable.

Procedure:
- Check out or inspect the exact landed commit.
- For `stable_rust_tag`, create the sync tag through `scripts/codexsdk_sync_tag.py`; use `--next-suffix` for existing base-tag conflicts.
- For manual refs or commits, skip upstream sync tagging.
- Run caller-provided drift verification when requested.
- Report the precise remote completion layer.

Success means:
- Remote completion layer is reported precisely, and stable sync tags or drift verification have been handled when applicable.

Validation:
- Tag, if created, points at the landed commit.
- Drift verification result is known when dispatched.
- Existing tags were not moved or deleted.

Final output:
- Landed ref, landed commit, sync PR, upstream target, upstream commit, sync tag when present, drift verification result, and completion layer.

Stop rules:
- Stop if the PR has not landed or the landed commit cannot be verified.
- Stop before creating a tag for manual refs or manual commits.
- Stop on tag conflict unless using the documented suffix path.
