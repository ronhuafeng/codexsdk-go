# Command: finalize-landed-sync

State handled:
- A sync PR has landed and caller-owned remote completion may need tagging or drift verification.

Trusted inputs:
- Landed ref, landed commit, sync PR, upstream target ref, upstream ref kind, upstream commit, and whether to dispatch drift verification.

Read:
- Top-level skill contract and invariants in `../SKILL.md`.
- `../references/local-sync.md`, "Tag Landed Syncs".

Fixed tools:
- `scripts/codexsdk_sync_tag.py` for stable upstream sync tag creation and suffix selection.
- Caller-owned workflow dispatch or CI tooling for drift verification when explicitly requested.

Allowed side effects:
- May create stable upstream sync tags only through `scripts/codexsdk_sync_tag.py`.
- May run a caller-provided drift verification step when explicitly requested.

Forbidden side effects:
- Do not delete or move existing tags.
- Do not tag unmerged PR heads, failed attempts, drift-check-only work, or uncommitted changes.
- Do not call a drift issue fully resolved before CI, tag, drift verification, and issue closure are complete when applicable.

Shortest safe path:
- Verify the landed commit before any tag or verification side effect.
- For `stable_rust_tag`, create tags only through the sync tag script and documented suffix path.
- For manual refs or commits, skip upstream sync tagging.
- Run drift verification only when the caller owns that dispatch.
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
