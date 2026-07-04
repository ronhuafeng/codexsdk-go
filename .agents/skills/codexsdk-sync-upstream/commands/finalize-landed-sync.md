# Command: finalize-landed-sync

State:
- Sync PR landed and caller-owned remote completion may need landed-commit verification, tagging, drift verification, or issue closure/update.

Inputs:
- Landed ref, landed commit, sync PR, upstream target ref/kind/SHA, optional drift issue/fingerprint metadata, and whether caller requested drift verification.

Tools:
- `scripts/codexsdk_sync_tag.py`
- Caller-owned workflow dispatch or CI tooling for drift verification.

Boundaries:
- Verify the landed commit is the commit being finalized before tagging or dispatching verification.
- Accept only the repository default branch as the landing/finalize ref unless an explicit future allowlist exists.
- May create stable upstream sync tags only through the sync tag script, which must choose suffixes from remote tag state when pushing.
- May run drift verification only when explicitly requested.
- May close or update the protocol-drift issue only after landed commit verification, tag handling when applicable, and drift verification have completed.
- Must not delete/move tags, tag unmerged PR heads or failed attempts, merge PRs, or call drift fully resolved before CI, tag, drift verification, and issue closure are complete when applicable.

Checks:
- Tag, if created, points at the landed commit.
- Drift verification result is known when dispatched.
- Issue state matches the verification result when issue closure/update is requested.
- Existing tags were not moved or deleted.

Output:
- Landed ref, landed commit, sync PR, upstream target, upstream commit, sync tag when present, drift verification result, issue action when present, and completion layer.

Stop if:
- PR has not landed or landed commit cannot be verified.
- Target is manual ref/commit and tagging would be attempted.
- Tag conflict cannot use the documented suffix path.
