#!/usr/bin/env python3

from __future__ import annotations

import hashlib
import json
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


def write_json(path: Path, value: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2) + "\n", encoding="utf-8")


class WriteDriftIssueArtifactsTest(unittest.TestCase):
    def test_cli_writes_artifacts_and_github_output(self) -> None:
        script = Path(__file__).with_name("codexsdk_write_drift_issue_artifacts.py")
        with tempfile.TemporaryDirectory() as tmp:
            artifact_dir = Path(tmp) / "artifact"
            drift_path = artifact_dir / "reports" / "drift_summary.json"
            write_json(drift_path, sample_drift())
            output = Path(tmp) / "github-output.txt"
            env = {**os.environ, "GITHUB_OUTPUT": str(output)}

            completed = subprocess.run(
                [
                    sys.executable,
                    str(script),
                    "--artifact-dir",
                    str(artifact_dir),
                    "--baseline-sha",
                    "b" * 40,
                    "--run-url",
                    "https://github.example/runs/1",
                    "--upstream-ref",
                    "rust-v0.141.0",
                    "--upstream-ref-kind",
                    "stable_rust_tag",
                    "--upstream-sha",
                    "a" * 40,
                    "--json",
                ],
                check=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
                env=env,
            )

            expected_sha = hashlib.sha256(drift_path.read_bytes()).hexdigest()
            payload = json.loads(completed.stdout)
            self.assertEqual(payload["status"], "review-required")
            self.assertEqual(payload["drift_sha"], expected_sha)
            self.assertEqual(completed.stderr, "")
            self.assertTrue((artifact_dir / "issue-title.txt").exists())
            self.assertIn("status=review-required", output.read_text(encoding="utf-8"))
            self.assertIn(f"drift_sha={expected_sha}", output.read_text(encoding="utf-8"))


def sample_drift() -> dict[str, object]:
    return {
        "status": "review-required",
        "file_diff": {
            "added": ["ThreadStartParams.json"],
            "changed": ["ClientRequest.json"],
            "removed": [],
        },
        "method_diff": {
            "ClientRequest.json": {
                "added": ["thread/start"],
                "removed": [],
            }
        },
    }


if __name__ == "__main__":
    unittest.main()
