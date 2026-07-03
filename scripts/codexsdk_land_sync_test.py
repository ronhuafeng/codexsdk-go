#!/usr/bin/env python3

from __future__ import annotations

import os
import shutil
import subprocess
import tempfile
import textwrap
import unittest
from pathlib import Path


TARGET_SHA = "a" * 40


def run(args: list[str], *, cwd: Path, env: dict[str, str] | None = None) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        args,
        cwd=cwd,
        env=env,
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )


class land_sync_repo:
    def __enter__(self) -> "land_sync_repo":
        self.tempdir = tempfile.TemporaryDirectory()
        self.root = Path(self.tempdir.name)
        self.origin = self.root / "origin.git"
        self.repo = self.root / "repo"
        run(["git", "init", "--bare", str(self.origin)], cwd=self.root)
        run(["git", "init", str(self.repo)], cwd=self.root)
        run(["git", "config", "user.email", "codex@example.com"], cwd=self.repo)
        run(["git", "config", "user.name", "Codex"], cwd=self.repo)
        run(["git", "remote", "add", "origin", str(self.origin)], cwd=self.repo)

        scripts = self.repo / "scripts"
        scripts.mkdir()
        source_script = Path(__file__).with_name("codexsdk_land_sync.sh")
        shutil.copy2(source_script, scripts / "codexsdk_land_sync.sh")
        (scripts / "codexsdk_validate_sync.sh").write_text(
            textwrap.dedent(
                """\
                #!/usr/bin/env bash
                set -euo pipefail
                printf '%s\\n' "$*" >> .validate-log
                """
            ),
            encoding="utf-8",
        )
        (scripts / "codexsdk_resolve_upstream.py").write_text(
            textwrap.dedent(
                """\
                #!/usr/bin/env python3
                import json
                import os
                print(json.dumps({"upstream_sha": os.environ["CODEXSDK_TEST_TARGET_SHA"]}))
                """
            ),
            encoding="utf-8",
        )
        for path in scripts.iterdir():
            path.chmod(0o755)

        (self.repo / "README.md").write_text("base\n", encoding="utf-8")
        run(["git", "add", "README.md"], cwd=self.repo)
        run(["git", "commit", "-m", "base"], cwd=self.repo)
        self.base_commit = run(["git", "rev-parse", "HEAD"], cwd=self.repo).stdout.strip()
        run(["git", "branch", "-M", "tmp"], cwd=self.repo)
        run(["git", "push", "origin", "tmp"], cwd=self.repo)
        run(["git", "switch", "-c", "codex/sync-test"], cwd=self.repo)
        (self.repo / "README.md").write_text("sync\n", encoding="utf-8")
        run(["git", "commit", "-am", "sync"], cwd=self.repo)
        self.sync_commit = run(["git", "rev-parse", "HEAD"], cwd=self.repo).stdout.strip()
        return self

    def __exit__(self, *exc: object) -> None:
        self.tempdir.cleanup()

    def env(self, output_path: Path) -> dict[str, str]:
        env = os.environ.copy()
        env["CODEXSDK_TEST_TARGET_SHA"] = TARGET_SHA
        env["GITHUB_OUTPUT"] = str(output_path)
        return env

    def script(self) -> Path:
        return self.repo / "scripts" / "codexsdk_land_sync.sh"

    def remote_commit(self, branch: str) -> str:
        output = run(["git", "ls-remote", "origin", f"refs/heads/{branch}"], cwd=self.repo).stdout.strip()
        return output.split()[0]

    def remote_branch_exists(self, branch: str) -> bool:
        output = run(["git", "ls-remote", "origin", f"refs/heads/{branch}"], cwd=self.repo).stdout.strip()
        return bool(output)

    def install_reject_first_tmp_push_hook(self) -> None:
        hook = self.origin / "hooks" / "pre-receive"
        hook.write_text(
            textwrap.dedent(
                """\
                #!/usr/bin/env bash
                while read -r _old _new ref; do
                  if [ "$ref" = "refs/heads/tmp" ] && [ ! -f rejected-tmp-once ]; then
                    touch rejected-tmp-once
                    exit 1
                  fi
                done
                """
            ),
            encoding="utf-8",
        )
        hook.chmod(0o755)


class LandSyncTest(unittest.TestCase):
    def test_lands_committed_sync_by_fast_forwarding_landing_ref(self) -> None:
        with land_sync_repo() as repo:
            output = repo.root / "github-output"
            completed = run(
                [
                    str(repo.script()),
                    "--land-ref",
                    "refs/heads/tmp",
                    "--target-ref",
                    "rust-v0.0.1",
                    "--target-sha",
                    TARGET_SHA,
                ],
                cwd=repo.repo,
                env=repo.env(output),
            )

            self.assertIn("up to date", completed.stdout)
            self.assertEqual(repo.remote_commit("tmp"), repo.sync_commit)
            self.assertFalse(repo.remote_branch_exists("codex/sync-test"))
            self.assertIn(f"landed_commit={repo.sync_commit}", output.read_text(encoding="utf-8"))
            self.assertIn("landed_ref=tmp", output.read_text(encoding="utf-8"))
            validate_lines = (repo.repo / ".validate-log").read_text(encoding="utf-8").splitlines()
            self.assertGreaterEqual(len(validate_lines), 2)

    def test_retries_once_when_first_landing_push_is_rejected(self) -> None:
        with land_sync_repo() as repo:
            repo.install_reject_first_tmp_push_hook()
            output = repo.root / "github-output"
            completed = run(
                [
                    str(repo.script()),
                    "--land-ref",
                    "tmp",
                    "--target-ref",
                    "rust-v0.0.1",
                    "--target-sha",
                    TARGET_SHA,
                ],
                cwd=repo.repo,
                env=repo.env(output),
            )

            self.assertIn("retrying once after rebase", completed.stderr)
            self.assertTrue((repo.origin / "rejected-tmp-once").exists())
            self.assertEqual(repo.remote_commit("tmp"), repo.sync_commit)
            validate_lines = (repo.repo / ".validate-log").read_text(encoding="utf-8").splitlines()
            self.assertGreaterEqual(len(validate_lines), 3)


if __name__ == "__main__":
    unittest.main()
