---
name: codexsdk-sync-upstream
description: Sync codexsdk-go's checked-in Codex app-server protocol baseline to a selected upstream openai/codex tag, ref, or commit. Use for protocol drift issues, baseline metadata/report refresh, protocolv2 regeneration, validation, upstream sync tagging, and drift issue closure.
---

# Codex SDK Upstream Sync

## Contract

Update the SDK by treating the checked-in app-server schema baseline as the source for generated Go code. Do not make the SDK follow the local `codex` binary implicitly during normal builds.

Use the repository's tracking script first, then review protocol drift before copying anything into the tree. Helper scripts are report-only or mechanical unless their usage says otherwise; they must not decide the sync strategy.

Tool output contract:

- successful commands are quiet by default
- use `--verbose` for human-readable paths, progress, and counts on stderr
- use `--json` when another command or workflow needs machine-readable stdout
- treat exit code as the success/failure signal

## Completion Layers

A baseline sync is complete only when checked-in schemas match the selected upstream target, baseline metadata and checked-in drift reports are sanitized and clean, manifest and coverage no longer reference removed surface, `protocolv2` generated files reproduce exactly, and validation passes.

Every final response must state the highest completed layer:

- `local sync complete`: files validate locally, but nothing was pushed
- `commit pushed`: the sync commit was pushed, but tag/CI/drift workflow/issue closure are still pending
- `drift issue fully resolved`: tag, pushed CI, drift workflow, and issue closure are complete when applicable

Do not call a drift issue solved at push time.

## Non-Negotiable Invariants

- Do not push directly to protected `main`.
- Do not introduce a PAT, GitHub App token, bot-token bypass, or synthetic required status.
- Do not delete or move upstream sync tags.
- Do not weaken branch protection or merge around failed required checks.
- Keep `action_required` documented as an expected maintainer rerun gate for sync PRs created by `GITHUB_TOKEN`.
- Keep auto-merge on the real protected-branch PR path after the required `Go` check passes.
- Keep generated reports free of local absolute paths, `.cache` output paths, private repo paths, account data, and raw smoke-test transcripts.
- Preserve unrelated user changes.

## Reference Routing

Read only the references needed for the task:

- For a local baseline sync, target resolution, drift review, checked-in baseline updates, regeneration, validation, or tagging, read [references/local-sync.md](references/local-sync.md).
- For the GitHub Actions auto-sync workflow, protected PR path, drift issue behavior, auto-merge behavior, or remote completion layers, read [references/automation.md](references/automation.md).
- For failed sync PR checks, auto-merge timeouts, finalize failures, drift verification failures, or existing sync tags, read [references/recovery.md](references/recovery.md).

If a failure occurs, use the recovery reference before adding automation.

## Default Workflow

1. Inspect `git status --short`, the current branch, and current baseline metadata.
2. Resolve the upstream target with `scripts/codexsdk_resolve_upstream.py`; default to the latest stable `rust-vX.Y.Z` tag only when no target is specified.
3. Run `scripts/codexsdk_target_policy.py` before generating drift. Stop on `block`; do not convert a policy block into a drift issue.
4. Generate drift artifacts with `scripts/codexsdk_track_upstream.sh` after policy allows the target.
5. Review compact drift reports before updating checked-in files.
6. Apply reviewed baseline changes, regenerate generated Go, and reconcile handwritten SDK only when reviewed drift requires it.
7. Run validation from the local sync reference before staging or publishing.
8. If publishing, use the protected PR path and report the correct completion layer.

## Required Inputs

Collect or infer:

- upstream Codex target tag, ref, or commit
- resolved upstream commit as the peeled full SHA
- upstream provenance as `source_ref_name` and `source_ref_kind`
- local OpenAI Codex repo path, from the prompt, `CODEXSDK_CODEX_REPO`, or `.cache/openai-codex`
- generator mode, defaulting to `cargo`; use `binary` only when the provided `codex` binary is known to be built from the resolved target commit
- output workdir, defaulting to `.cache/codexsdk-upstream-<short-sha>`

If the target cannot be inferred from the user request, latest stable tag, or local context, ask before changing files.
