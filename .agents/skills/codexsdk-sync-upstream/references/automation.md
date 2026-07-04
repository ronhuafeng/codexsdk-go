# Split Workflow Automation Reference

Use this reference for the split GitHub Actions drift/fix/finalize workflows, protected PR path, drift issue behavior, auto-merge behavior, drift verification, and remote completion layers.

## Contents

- [Workflow Contract](#workflow-contract)
- [Detect Job](#detect-job)
- [Sync PR Job](#sync-pr-job)
- [Finalize Job](#finalize-job)
- [Remote Completion Layers](#remote-completion-layers)
- [Helper Boundaries](#helper-boundaries)

## Workflow Contract

Routine drift handling is split across detect, fix, and finalize workflows. Scheduled or manual detect creates or updates issue state; fix is explicitly dispatched from trusted inputs; finalize runs only after the protected PR has landed.

The high-level job structure must stay recognizable:

1. `detect`: resolve target, run policy, generate drift, upload candidate, and create/update/close drift issue state when caller-owned.
2. `fix`: download or regenerate candidate, apply mechanical sync, ask Codex for bounded maintainer review, validate, commit, and publish a protected sync PR.
3. protected branch auto-merge: GitHub merges only after required real checks pass.
4. `finalize`: verify the landed commit, tag landed stable syncs when applicable, dispatch drift verification when requested, and close/update issue state when caller-owned.

Invariants:

- landing ref is the repository default branch
- no direct push to protected `main`
- no PAT, GitHub App token, or bot-token bypass
- no synthetic required status writing
- no tag deletion or tag movement
- auto-merge continues through real PR `Go` checks
- merge queue, if enabled, may enqueue instead of immediate merge

Sync PRs created by `GITHUB_TOKEN` may require one maintainer approval or rerun before GitHub schedules the required `Go` pull request check. This `action_required` state is an expected GitHub Actions safety gate, not a sync failure.

## Detect Workflow

Detect must:

- resolve landing ref as the repository default branch
- check out the landing ref with full history
- resolve upstream target through `scripts/codexsdk_resolve_upstream.py`
- evaluate target policy through `scripts/codexsdk_target_policy.py`
- stop cleanly on policy `block`
- stop drift generation on policy `skip`, then close or update the protocol-drift issue only when the caller owns that side effect
- clone upstream only after policy `allow`
- generate drift with `scripts/codexsdk_track_upstream.sh`
- capture upstream `common.rs` response mappings for the sync job
- render drift issue title/body/comment as deterministic artifacts
- upload the full candidate artifact directory for the sync job

When drift status is `clean`, close an existing protocol-drift issue. When drift status is `review-required`, create or update one open protocol-drift issue by label and comment only when the drift fingerprint changed.

Do not create drift issues for target policy blocks.

## Fix Workflow

Fix must run only when explicitly dispatched from trusted issue metadata or explicit target/fingerprint inputs.

The job must:

- check out the landing ref
- set up Go and Codex Rust
- download the candidate artifact from detect
- configure a bot git identity
- apply the candidate mechanically with `scripts/codexsdk_apply_sync_candidate.py`
- build a bounded Codex review prompt
- run `openai/codex-action@v1` against the current worktree
- validate with `scripts/codexsdk_validate_sync.sh`
- commit the validated local sync through the `commit-local-sync` boundary only when there are real changes
- publish a sync PR with `scripts/codexsdk_publish_sync_pr.sh`
- stop at PR publication unless a separate caller-owned merge/finalize command was selected

The Codex review prompt must keep judgment bounded:

- read the sync skill before editing
- inspect apply summary, compact drift reports, separate git diff/status evidence, `common.rs` provenance, and the required drift review checklist
- do not re-copy schemas or rerun the full Rust schema generator
- do not print large schema, report, manifest, coverage, or generated Go files
- make only small, concrete fixes found by review
- leave repository side effects to the workflow
- keep handwritten SDK changes minimal and tied to reviewed drift

Post-publication diagnostics must keep merge decisions on the protected PR path:

- PR state, merge state, review decision, and status rollup
- `gh pr checks`
- recent `ci.yml` runs for the sync branch
- reminder that an empty `action_required` first run needs one maintainer rerun

## Finalize Workflow

Finalize must run only after the sync PR has landed.

The job must:

- check out the landed ref and exact landed commit
- configure git identity
- create an upstream sync tag only for `stable_rust_tag`
- if the base tag already exists at another commit, use the suffix path through `scripts/codexsdk_sync_tag.py --next-suffix`
- when pushing, let `scripts/codexsdk_sync_tag.py` choose the base tag or next `-sync.N` suffix from remote tag state
- dispatch `upstream-protocol-drift.yml` when scheduled or when `dispatch_drift_check` is true
- watch the dispatched drift workflow to completion
- report landing ref, landed commit, sync PR, upstream target, upstream commit, and sync tag when present

Manual refs and manual commits do not get upstream sync tags.

## Remote Completion Layers

Report remote completion precisely:

- `commit pushed`: sync PR branch exists, but protected merge, tag, CI, drift workflow, or issue closure is pending
- `drift issue fully resolved`: sync PR merged, stable-tag sync tag handling completed when applicable, pushed CI passed, drift verification passed, and the drift issue closed when applicable

After a sync PR merges into `main`, the Codex Upstream Protocol Drift workflow runs on schedule or `workflow_dispatch`, not ordinary pushes. If resolving a drift issue, dispatch drift verification for the selected upstream ref and watch the run:

```sh
gh workflow run upstream-protocol-drift.yml --ref main -f upstream_ref=<target_ref>
gh run watch <run-id> --exit-status
```

Cold GitHub runners may spend several minutes compiling Rust before the drift report step advances.

Confirm outcome:

- if drift is clean, the workflow should close the existing drift issue
- if drift remains, the workflow should update the existing drift issue rather than creating duplicates
- if the selected upstream tag/ref moved or a newer stable tag exists, report the exact tag/ref and commit and whether remaining drift is real or clean

## Helper Boundaries

Workflow helper scripts may perform deterministic formatting, rendering, validation, polling, or writing. They must not decide analysis strategy, model, prompt policy, task split, repair logic, branch protection strategy, or sync target policy.

Good helpers in this workflow include:

- rendering the Codex prompt file from explicit inputs
- rendering drift issue title/body/comment artifacts from `drift_summary.json`
- polling a PR merge and printing diagnostics
- validating generated outputs and sync state

Do not replace the workflow with a monolithic auto-sync pipeline.
