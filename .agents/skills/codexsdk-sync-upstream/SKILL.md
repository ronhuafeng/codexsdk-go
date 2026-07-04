---
name: codexsdk-sync-upstream
description: Sync codexsdk-go's checked-in Codex app-server protocol baseline to a selected upstream openai/codex tag, ref, or commit. Use for protocol drift issues, baseline metadata/report refresh, protocolv2 regeneration, validation, upstream sync tagging, and drift issue closure.
---

# Codex SDK Upstream Sync

## Contract

The checked-in app-server schema baseline is the source of truth for generated Go. Do not make the SDK implicitly follow a local `codex` binary during normal builds.

This skill is context and routing, not orchestration. It gives Codex the domain contract, safety boundaries, command index, and completion language. Inside a selected command, Codex may choose the shortest safe path from current evidence.

Use canonical scripts for deterministic work. Use Codex judgment for classification, repair decisions, focused validation choices, and recovery routing.

## Completion Layers

Report the highest completed layer precisely:

- `local sync complete`: files validate locally, but nothing was pushed
- `commit pushed`: the sync commit was pushed, but tag, CI, drift verification, or issue closure are still pending
- `drift issue fully resolved`: tag, pushed CI, drift verification, and issue closure are complete when applicable

Never call a drift issue solved at push time.

## Safety Boundaries

- Do not push directly to protected `main`.
- Do not introduce a PAT, GitHub App token, bot-token bypass, or synthetic required status.
- Do not delete or move upstream sync tags.
- Do not weaken branch protection or merge around failed required checks.
- Keep `action_required` documented as an expected maintainer rerun gate for sync PRs created by `GITHUB_TOKEN`.
- Keep auto-merge on the real protected-branch PR path after the required `Go` check passes.
- Keep generated reports free of local absolute paths, `.cache` output paths, private repo paths, account data, and raw smoke-test transcripts.
- Preserve unrelated user changes.

## Command Index

Commands live under [commands/](commands/). A caller may invoke any single command directly when its state and inputs match. The command-specific boundary wins over any example or prior conversation.

- [resolve-target](commands/resolve-target.md): resolve an upstream target and baseline provenance.
- [detect-drift](commands/detect-drift.md): run target policy and create local drift artifacts.
- [review-drift](commands/review-drift.md): classify compact drift evidence before checked-in changes.
- [apply-candidate](commands/apply-candidate.md): mechanically apply reviewed candidate artifacts.
- [repair-applied-candidate](commands/repair-applied-candidate.md): repair or confirm an already-applied candidate.
- [validate-local](commands/validate-local.md): validate local sync state.
- [publish-protected-pr](commands/publish-protected-pr.md): publish through the protected PR path when explicitly owned by the caller.
- [finalize-landed-sync](commands/finalize-landed-sync.md): tag and verify a landed sync when explicitly owned by the caller.
- [recover-failure](commands/recover-failure.md): recover failed checks, merge waits, finalize failures, drift verification failures, or tag conflicts.

References are optional context, not required linear playbooks:

- [references/local-sync.md](references/local-sync.md): local sync context and decision rules.
- [references/recovery.md](references/recovery.md): recovery recipes for known remote failure states.

## Input Policy

Collect only the inputs needed by the selected command. Do not chase upstream repo paths, generator mode, output workdirs, or target provenance in commands that already receive authoritative artifacts.

If a command needs an upstream target and it cannot be inferred from the request, latest stable tag, or local context, ask before changing files.

## After Run

If execution reveals a concrete improvement to this skill, a helper script, or an environment assumption, mention it briefly and ask whether to update the skill. Do not modify the skill during the original command unless the user explicitly asks.
