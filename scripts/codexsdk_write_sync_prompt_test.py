#!/usr/bin/env python3

from __future__ import annotations

import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, os.path.dirname(__file__))

import codexsdk_write_sync_prompt as sync_prompt


class WriteSyncPromptTest(unittest.TestCase):
    def test_render_prompt_substitutes_template_values(self) -> None:
        prompt = sync_prompt.render_prompt(
            "ref=${UPSTREAM_REF}\nkind=${UPSTREAM_REF_KIND}\nsha=${UPSTREAM_SHA}\n",
            auto_sync_dir=".cache/codexsdk-auto-sync",
            candidate_dir=".cache/candidate",
            land_ref="main",
            upstream_ref="rust-v0.141.0",
            upstream_ref_kind="stable_rust_tag",
            upstream_sha="a" * 40,
        )

        self.assertIn("ref=rust-v0.141.0", prompt)
        self.assertIn("kind=stable_rust_tag", prompt)
        self.assertTrue(prompt.endswith("\n"))

    def test_build_prompt_includes_bounded_repair_contract(self) -> None:
        prompt = sync_prompt.build_prompt(
            auto_sync_dir=".cache/codexsdk-auto-sync",
            candidate_dir=".cache/codexsdk-upstream-abc123",
            land_ref="main",
            upstream_ref="rust-v0.141.0",
            upstream_ref_kind="stable_rust_tag",
            upstream_sha="a" * 40,
        )

        self.assertIn("Use codexsdk-sync-upstream command: repair-applied-candidate.", prompt)
        self.assertIn("Detect and apply have already completed", prompt)
        self.assertIn("Authoritative inputs", prompt)
        self.assertIn("Do not follow a global sync workflow", prompt)
        self.assertIn("shortest safe path", prompt)
        self.assertIn("- upstream ref: rust-v0.141.0", prompt)
        self.assertIn(".cache/codexsdk-upstream-abc123/schema", prompt)
        self.assertIn("Do not run `resolve-target`, `detect-drift`, `scripts/codexsdk_track_upstream.sh`", prompt)
        self.assertIn("Final output must include", prompt)
        self.assertIn("Do not re-copy schemas", prompt)
        self.assertIn("Do not commit, push, tag", prompt)
        self.assertNotIn("references/automation.md", prompt)

    def test_cli_writes_prompt_file_quietly(self) -> None:
        script = Path(__file__).with_name("codexsdk_write_sync_prompt.py")
        with tempfile.TemporaryDirectory() as tmp:
            out = Path(tmp) / "prompt.md"
            completed = subprocess.run(
                [
                    sys.executable,
                    str(script),
                    "--out",
                    str(out),
                    "--auto-sync-dir",
                    ".cache/codexsdk-auto-sync",
                    "--candidate-dir",
                    ".cache/candidate",
                    "--land-ref",
                    "main",
                    "--upstream-ref",
                    "rust-v0.141.0",
                    "--upstream-ref-kind",
                    "stable_rust_tag",
                    "--upstream-sha",
                    "a" * 40,
                ],
                check=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
            )

            self.assertEqual(completed.stdout, "")
            self.assertEqual(completed.stderr, "")
            self.assertIn("rust-v0.141.0", out.read_text(encoding="utf-8"))


if __name__ == "__main__":
    unittest.main()
