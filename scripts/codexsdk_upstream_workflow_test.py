#!/usr/bin/env python3

from __future__ import annotations

import subprocess
import unittest
from pathlib import Path


REPO = Path(__file__).resolve().parents[1]


def read(rel: str) -> str:
    return (REPO / rel).read_text(encoding="utf-8")


class UpstreamWorkflowContractTest(unittest.TestCase):
    def test_drift_workflow_is_detect_phase_and_explicitly_dispatches_fix_only_when_requested(self) -> None:
        workflow = read(".github/workflows/upstream-protocol-drift.yml")

        self.assertIn("Current phase: \\`detect\\`", workflow)
        self.assertIn("force_compare", workflow)
        self.assertIn("dispatch_fix", workflow)
        self.assertIn("gh workflow run upstream-protocol-auto-sync.yml", workflow)
        self.assertIn("steps.issue_state.outputs.issue_number", workflow)
        self.assertIn("Fail verification when drift remains", workflow)
        self.assertIn("inputs.force_compare", workflow)
        self.assertIn("steps.drift.outputs.status != 'clean'", workflow)
        self.assertIn("steps.drift.outputs.status == 'clean' && !inputs.force_compare", workflow)

    def test_workflows_use_only_supported_concurrency_keys(self) -> None:
        for rel in (
            ".github/workflows/upstream-protocol-drift.yml",
            ".github/workflows/upstream-protocol-auto-sync.yml",
            ".github/workflows/upstream-protocol-finalize.yml",
        ):
            with self.subTest(workflow=rel):
                self.assertNotIn("queue: max", read(rel))

    def test_workflows_constrain_control_plane_and_landing_refs(self) -> None:
        drift = read(".github/workflows/upstream-protocol-drift.yml")
        fix = read(".github/workflows/upstream-protocol-auto-sync.yml")
        finalize = read(".github/workflows/upstream-protocol-finalize.yml")

        for workflow in (drift, fix, finalize):
            with self.subTest(workflow=workflow.splitlines()[0]):
                self.assertIn("Guard control-plane ref", workflow)
                self.assertIn("github.event.repository.default_branch", workflow)
                self.assertIn("GITHUB_REF_NAME", workflow)

        self.assertIn('--ref "${DEFAULT_BRANCH}"', drift)
        self.assertNotIn('--ref "${GITHUB_REF_NAME}"', drift)

        self.assertIn("Resolve landing ref", fix)
        self.assertIn("Refusing landing ref", fix)
        self.assertIn("steps.landing.outputs.land_ref", fix)
        self.assertIn("--default-branch", fix)
        self.assertNotIn("default: main", fix)
        self.assertNotIn("ref: ${{ inputs.land_ref", fix)

        self.assertIn("Verify landing ref policy", finalize)
        self.assertIn("Refusing to finalize landing ref", finalize)
        self.assertIn('--ref "${DEFAULT_BRANCH}"', finalize)
        self.assertIn('--branch "${DEFAULT_BRANCH}"', finalize)
        self.assertNotIn('--ref "${LANDED_REF}"', finalize)
        self.assertNotIn("ref: ${{ inputs.landed_ref", finalize)

    def test_publish_script_enforces_default_branch_before_other_validation(self) -> None:
        completed = subprocess.run(
            [
                "bash",
                "scripts/codexsdk_publish_sync_pr.sh",
                "--land-ref",
                "release",
                "--default-branch",
                "main",
                "--target-ref",
                "rust-v0.1.0",
                "--target-sha",
                "a" * 40,
                "--skip-pr",
            ],
            cwd=REPO,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
        )

        self.assertNotEqual(completed.returncode, 0)
        self.assertIn("Refusing landing ref release", completed.stderr)
        self.assertNotIn("worktree must be clean", completed.stderr)

    def test_validate_sync_checks_gofmt_without_rewriting_tracked_go(self) -> None:
        script = read("scripts/codexsdk_validate_sync.sh")

        self.assertIn("gofmt -l", script)
        self.assertIn("tracked Go files are not gofmt-formatted", script)
        self.assertNotIn("git ls-files -z -- '*.go' ':!:vendor/**' | xargs -0 gofmt -w", script)

    def test_skill_command_contracts_match_script_boundaries(self) -> None:
        skill = read(".agents/skills/codexsdk-sync-upstream/SKILL.md")
        resolve = read(".agents/skills/codexsdk-sync-upstream/commands/resolve-target.md")
        review = read(".agents/skills/codexsdk-sync-upstream/commands/review-drift.md")
        detect = read(".agents/skills/codexsdk-sync-upstream/commands/detect-drift.md")
        apply_candidate = read(".agents/skills/codexsdk-sync-upstream/commands/apply-candidate.md")
        commit_local = read(".agents/skills/codexsdk-sync-upstream/commands/commit-local-sync.md")
        validate = read(".agents/skills/codexsdk-sync-upstream/commands/validate-local.md")
        publish = read(".agents/skills/codexsdk-sync-upstream/commands/publish-protected-pr.md")
        finalize = read(".agents/skills/codexsdk-sync-upstream/commands/finalize-landed-sync.md")
        repair = read(".agents/skills/codexsdk-sync-upstream/commands/repair-applied-candidate.md")
        local_sync = read(".agents/skills/codexsdk-sync-upstream/references/local-sync.md")
        recovery = read(".agents/skills/codexsdk-sync-upstream/references/recovery.md")

        self.assertIn("caller or target policy owns baseline provenance", skill)
        self.assertIn("stable-tag sync tag handling when applicable", skill)
        self.assertIn("checked-in baseline metadata and checked-in reports", skill)
        self.assertIn("[commit-local-sync](commands/commit-local-sync.md)", skill)
        self.assertNotIn("Baseline metadata path", resolve)
        self.assertNotIn("current baseline ref/commit", resolve)
        self.assertIn("Baseline metadata is read by the caller or by target-policy checks", resolve)
        self.assertIn("accepts them syntactically", resolve)
        self.assertIn("advanced/manual targets", resolve)

        self.assertIn("`file_diff`, `method_diff`", review)
        self.assertIn("new request response mappings", review)
        self.assertIn("public facade impact", review)

        self.assertIn("stop drift generation in both cases", detect)
        self.assertIn("caller-owned issue close/update was not explicitly requested", detect)

        self.assertIn("`common.rs` source SHA", apply_candidate)
        self.assertIn("content is verified from `target_sha:codex-rs/app-server-protocol/src/protocol/common.rs`", apply_candidate)
        self.assertIn("separate diff name-status evidence from `git diff --name-status` or `git status --short`", apply_candidate)
        self.assertIn("Changed files from separate git diff/status evidence", apply_candidate)

        self.assertIn("May stage and commit only reviewed local sync changes", commit_local)
        self.assertIn("Must preserve unrelated user changes", commit_local)
        self.assertIn("must not create the commit itself", local_sync)

        self.assertIn("checked-in baseline metadata or checked-in reports", validate)

        self.assertIn("`HEAD` is the committed sync change to publish", publish)
        self.assertIn("worktree and index are clean", publish)

        self.assertIn("choose suffixes from remote tag state when pushing", finalize)
        self.assertIn("remote tag state before selecting", recovery)

        self.assertIn("read-only context", repair)
        self.assertIn("must not drive branch checkout", repair)
        self.assertIn("common.rs` provenance", repair)

        self.assertIn("Recommended disposable locations", local_sync)
        self.assertNotIn("Default disposable locations", local_sync)
        self.assertIn("requires `--codex-repo` or `CODEXSDK_CODEX_REPO`", local_sync)
        self.assertIn("in `--compare-only` mode it needs only a resolved `--commit`", local_sync)
        self.assertIn("close/update caller-owned drift state only when explicitly requested", local_sync)
        self.assertIn("bare `manual_commit` SHA targets as explicit advanced inputs", local_sync)
        self.assertIn("`common.rs` must be bound to the same target commit", local_sync)
        self.assertIn("temporary `/tmp/codexsdk-upstream.*`", local_sync)

    def test_fix_workflow_stops_at_protected_pr_publication(self) -> None:
        workflow = read(".github/workflows/upstream-protocol-auto-sync.yml")

        self.assertIn("name: Codex Upstream Protocol Fix", workflow)
        self.assertIn("workflow_dispatch:", workflow)
        self.assertNotIn("schedule:", workflow)
        self.assertIn("issue_number:", workflow)
        self.assertIn("upstream_ref:", workflow)
        self.assertIn("drift_sha:", workflow)
        self.assertIn("--phase \"fix\"", workflow)
        self.assertIn("scripts/codexsdk_publish_sync_pr.sh", workflow)
        self.assertIn("common.rs.source_sha", workflow)
        self.assertIn("--common-rs-source-sha", workflow)
        self.assertIn("Commit local sync changes", workflow)
        self.assertIn("commit-local-sync", workflow)
        self.assertNotIn("scripts/codexsdk_wait_sync_pr_merge.sh", workflow)
        self.assertNotIn("scripts/codexsdk_sync_tag.py", workflow)
        self.assertNotIn("gh issue close", workflow)

    def test_finalize_workflow_owns_tag_verification_and_issue_closure(self) -> None:
        workflow = read(".github/workflows/upstream-protocol-finalize.yml")

        self.assertIn("name: Codex Upstream Protocol Finalize", workflow)
        self.assertIn("workflow_dispatch:", workflow)
        self.assertIn("scripts/codexsdk_sync_tag.py", workflow)
        self.assertIn("gh workflow run upstream-protocol-drift.yml", workflow)
        self.assertIn("-f \"force_compare=true\"", workflow)
        self.assertIn("createdAt >=", workflow)
        self.assertIn("gh issue close", workflow)
        self.assertIn("drift issue fully resolved", workflow)
        self.assertLess(
            workflow.index("Fail when drift verification failed"),
            workflow.index("Create upstream sync tag"),
        )
        self.assertLess(
            workflow.index("Create upstream sync tag"),
            workflow.index("Close drift issue after verified finalize"),
        )

    def test_repair_prompt_exposes_phase_and_side_effect_boundaries(self) -> None:
        prompt = read(".github/prompts/codexsdk-upstream-sync-repair.md")

        self.assertIn("Current phase: `${PHASE}`.", prompt)
        self.assertIn("Allowed side effects", prompt)
        self.assertIn("Forbidden side effects", prompt)
        self.assertIn("Final output must include", prompt)


if __name__ == "__main__":
    unittest.main()
