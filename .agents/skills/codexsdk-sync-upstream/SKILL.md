---
name: codexsdk-sync-upstream
description: Sync codexsdk-go's checked-in Codex app-server protocol baseline to a selected upstream openai/codex tag, ref, or commit. Use for protocol drift issues, baseline metadata/report refresh, protocolv2 regeneration, validation, upstream sync tagging, and drift issue closure.
---

# Codex SDK Upstream Sync

## Contract

Update the SDK by treating the checked-in app-server schema baseline as the source for generated Go code. Do not make the SDK follow the local `codex` binary implicitly during normal builds.

This skill is a contract and router, not a monolithic workflow. It supplies domain context, source-of-truth rules, completion layers, command boundaries, and dangerous-operation guardrails. Codex may choose the shortest safe path inside the selected command boundary, using focused evidence and checks that fit the current state.

Fixed, reproducible, error-prone operations belong in canonical scripts. Helper scripts are report-only or mechanical unless their usage says otherwise; they must not decide the sync strategy, bypass review, or publish remote state on their own.

Tool output contract:

- successful commands are quiet by default
- use `--verbose` for human-readable paths, progress, and counts on stderr
- use `--json` when another command or caller needs machine-readable stdout
- treat exit code as the success/failure signal

## Source Of Truth

- Checked-in baseline: `codexsdk/internal/protocolschema/appserver/v2`
- Baseline metadata: `codexsdk/internal/protocolschema/appserver/v2/baseline_metadata.json`
- Generated public protocol types: `codexsdk/protocolv2/*.gen.go`
- Sync candidate artifacts: the `schema/`, `reports/`, and `common.rs` outputs produced for the resolved target
- Canonical scripts under `scripts/` for target resolution, target policy, drift tracking, candidate apply, local validation, PR publication, merge waiting, and sync tagging

## Completion Layers

A baseline sync is complete only when checked-in schemas match the selected upstream target, baseline metadata and checked-in drift reports are sanitized and clean, manifest and coverage no longer reference removed surface, `protocolv2` generated files reproduce exactly, and validation passes.

Every final response must state the highest completed layer:

- `local sync complete`: files validate locally, but nothing was pushed
- `commit pushed`: the sync commit was pushed, but tag, CI, drift verification, or issue closure are still pending
- `drift issue fully resolved`: tag, pushed CI, drift verification, and issue closure are complete when applicable

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

Read only the references needed for the selected command and current failure state:

- For a local baseline sync, target resolution, drift review, checked-in baseline updates, regeneration, validation, or tagging, read [references/local-sync.md](references/local-sync.md).
- For failed sync PR checks, auto-merge timeouts, finalize failures, drift verification failures, or existing sync tags, read [references/recovery.md](references/recovery.md).

If a failure occurs, use the recovery reference before adding automation.

## Deterministic Tools

Use the existing scripts for stable, testable operations with clear inputs, outputs, and exit-code results:

- `scripts/codexsdk_resolve_upstream.py`: resolve tags, refs, commits, and peeled SHAs.
- `scripts/codexsdk_target_policy.py`: decide whether a target is allowed, skipped, or blocked.
- `scripts/codexsdk_track_upstream.sh`: generate drift artifacts or compare a candidate against the baseline.
- `scripts/codexsdk_apply_sync_candidate.py`: mechanically apply reviewed candidate artifacts.
- `scripts/codexsdk_validate_sync.sh`: validate local baseline, reports, manifests, coverage, generated Go, and path hygiene.
- `scripts/codexsdk_publish_sync_pr.sh`: publish validated local sync work through the protected PR path.
- `scripts/codexsdk_wait_sync_pr_merge.sh`: wait for the protected PR path to merge after real required checks.
- `scripts/codexsdk_sync_tag.py`: create stable upstream sync tags without moving existing tags.

Treat these scripts as capabilities for bounded commands. Do not replace them with ad hoc shell logic for target sorting, tag peeling, candidate copying, report sanitization, PR publication, merge waiting, or tag creation.

## Command Routing

Commands live under [commands/](commands/). A caller may invoke one command directly; that command's inputs, side effects, forbidden side effects, validation, and stop rules override any broader sequence.

- [resolve-target](commands/resolve-target.md): resolve the selected upstream Codex target and baseline provenance.
- [detect-drift](commands/detect-drift.md): run target policy, then generate local drift artifacts only when policy allows.
- [review-drift](commands/review-drift.md): review compact drift evidence before checked-in baseline changes.
- [apply-candidate](commands/apply-candidate.md): mechanically apply an already-reviewed candidate through canonical scripts.
- [repair-applied-candidate](commands/repair-applied-candidate.md): perform a bounded local repair pass after detect and apply have completed.
- [validate-local](commands/validate-local.md): validate local sync state without publishing remote state.
- [publish-protected-pr](commands/publish-protected-pr.md): publish through the protected PR path only when explicitly requested.
- [finalize-landed-sync](commands/finalize-landed-sync.md): tag and optionally verify a sync after the PR has landed.
- [recover-failure](commands/recover-failure.md): recover failed checks, merge waits, finalize failures, drift verification failures, or tag conflicts.

## Agent Judgment

Codex owns judgment inside the selected command boundary:

- identify the highest already-completed layer
- decide which compact evidence is needed
- decide whether drift needs local repair
- classify validation failures as generator, schema, manifest, coverage, SDK surface, provenance, or environment issues
- choose focused checks for changed behavior
- pick the narrow recovery command for a failed state

Codex does not own dangerous state transitions unless the selected command and caller explicitly grant them:

- push to protected branches
- bypass required checks or synthesize statuses
- create PATs, GitHub App tokens, or bot-token bypasses
- move or delete tags
- merge directly outside the protected PR path
- close drift issues while claiming full resolution before tag, CI, drift verification, and issue closure are complete

## Common Compositions

These are examples, not required global flows. A caller may start with any single command whose inputs and state match, and that command's boundary takes priority.

Typical local baseline sync:

1. `resolve-target`
2. `detect-drift`
3. `review-drift`
4. `apply-candidate`
5. `repair-applied-candidate` when reviewed drift or validation shows local repair is needed
6. `validate-local`
7. `publish-protected-pr` only when the user or workflow explicitly owns publishing

Typical recovery:

1. `recover-failure`
2. resume with the narrow command matching the recovered state, often `validate-local`, `publish-protected-pr`, or `finalize-landed-sync`

## Required Inputs

Collect or infer:

- upstream Codex target tag, ref, or commit
- resolved upstream commit as the peeled full SHA
- upstream provenance as `source_ref_name` and `source_ref_kind`
- local OpenAI Codex repo path, from the prompt, `CODEXSDK_CODEX_REPO`, or `.cache/openai-codex`
- generator mode, defaulting to `cargo`; use `binary` only when the provided `codex` binary is known to be built from the resolved target commit
- output workdir, defaulting to `.cache/codexsdk-upstream-<short-sha>`

If the target cannot be inferred from the user request, latest stable tag, or local context, ask before changing files.

## After Run

After completing the selected command, if execution revealed a concrete improvement to this skill, its helper scripts, or environment-specific assumptions, mention it briefly and ask whether to update the skill. Do not modify the skill during the original command unless the user explicitly asks.
