#!/usr/bin/env python3

from __future__ import annotations

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
        self.assertNotIn("default: main", fix)
        self.assertNotIn("ref: ${{ inputs.land_ref", fix)

        self.assertIn("Verify landing ref policy", finalize)
        self.assertIn("Refusing to finalize landing ref", finalize)
        self.assertIn('--ref "${DEFAULT_BRANCH}"', finalize)
        self.assertIn('--branch "${DEFAULT_BRANCH}"', finalize)
        self.assertNotIn('--ref "${LANDED_REF}"', finalize)
        self.assertNotIn("ref: ${{ inputs.landed_ref", finalize)

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
