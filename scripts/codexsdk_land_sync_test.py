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


def run_unchecked(
    args: list[str], *, cwd: Path, env: dict[str, str] | None = None
) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        args,
        cwd=cwd,
        env=env,
        check=False,
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

    def install_reject_tmp_push_until_merge_hook(self) -> None:
        hook = self.origin / "hooks" / "pre-receive"
        hook.write_text(
            textwrap.dedent(
                """\
                #!/usr/bin/env bash
                while read -r _old _new ref; do
                  if [ "$ref" = "refs/heads/tmp" ] && [ ! -f allow-tmp-merge ]; then
                    exit 1
                  fi
                done
                """
            ),
            encoding="utf-8",
        )
        hook.chmod(0o755)

    def install_gh_stub(self) -> Path:
        fake_bin = self.root / "fake-bin"
        fake_bin.mkdir()
        gh_log = self.root / "gh-log"
        gh = fake_bin / "gh"
        gh.write_text(
            textwrap.dedent(
                f"""\
                #!/usr/bin/env bash
                set -euo pipefail
                printf '%s\\n' "$*" >> {gh_log}
                case "$1 $2" in
                  "pr list")
                    exit 0
                    ;;
                  "pr create")
                    printf '%s\\n' "https://github.com/example/codexsdk-go/pull/42"
                    ;;
                  "pr ready")
                    exit 0
                    ;;
                  "pr checks")
                    printf '%s\\n' '[{{"name":"Go","bucket":"pass","state":"SUCCESS","link":"https://github.com/example/checks/1"}}]'
                    ;;
                  "pr merge")
                    touch {self.origin}/allow-tmp-merge
                    git push origin HEAD:refs/heads/tmp
                    git push origin :refs/heads/codex/sync-test >/dev/null 2>&1 || true
                    ;;
                  *)
                    printf 'unexpected gh command: %s\\n' "$*" >&2
                    exit 2
                    ;;
                esac
                """
            ),
            encoding="utf-8",
        )
        gh.chmod(0o755)
        return fake_bin


class LandSyncTest(unittest.TestCase):
    def test_lands_committed_sync_by_fast_forwarding_landing_ref(self) -> None:
        with land_sync_repo() as repo:
            output = repo.root / "github-output"
            completed = run(
                [
                    str(repo.script()),
                    "--land-ref",
                    "refs/heads/tmp",
                    "--work-branch",
                    "refs/heads/codex/sync-test",
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
            self.assertEqual(repo.remote_commit("codex/sync-test"), repo.sync_commit)
            self.assertIn(f"landed_commit={repo.sync_commit}", output.read_text(encoding="utf-8"))
            self.assertIn("landed_ref=tmp", output.read_text(encoding="utf-8"))
            self.assertIn("work_branch=codex/sync-test", output.read_text(encoding="utf-8"))
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
                    "--work-branch",
                    "codex/sync-test",
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

    def test_auto_merges_ready_pr_when_direct_landing_is_rejected(self) -> None:
        with land_sync_repo() as repo:
            repo.install_reject_tmp_push_until_merge_hook()
            fake_bin = repo.install_gh_stub()
            output = repo.root / "github-output"
            env = repo.env(output)
            env["PATH"] = f"{fake_bin}{os.pathsep}{env['PATH']}"
            env["CODEXSDK_PR_CHECK_TIMEOUT_SECONDS"] = "5"
            env["CODEXSDK_PR_CHECK_INTERVAL_SECONDS"] = "1"

            completed = run(
                [
                    str(repo.script()),
                    "--land-ref",
                    "tmp",
                    "--work-branch",
                    "codex/sync-test",
                    "--target-ref",
                    "rust-v0.0.1",
                    "--target-sha",
                    TARGET_SHA,
                    "--open-pr-on-failure",
                    "--auto-merge-pr-on-failure",
                ],
                cwd=repo.repo,
                env=env,
            )

            self.assertIn("Required PR checks passed", completed.stderr)
            self.assertEqual(repo.remote_commit("tmp"), repo.sync_commit)
            output_text = output.read_text(encoding="utf-8")
            self.assertIn("fallback_pr_url=https://github.com/example/codexsdk-go/pull/42", output_text)
            self.assertIn("fallback_pr_merged=true", output_text)
            self.assertIn(f"landed_commit={repo.sync_commit}", output_text)
            gh_log = (repo.root / "gh-log").read_text(encoding="utf-8")
            self.assertIn("pr create --base tmp --head codex/sync-test", gh_log)
            self.assertNotIn("pr create --draft", gh_log)
            self.assertIn("pr checks https://github.com/example/codexsdk-go/pull/42 --required", gh_log)
            self.assertIn("pr merge https://github.com/example/codexsdk-go/pull/42 --rebase --delete-branch", gh_log)

    def test_auto_merge_requires_bot_token_when_requested(self) -> None:
        with land_sync_repo() as repo:
            repo.install_reject_tmp_push_until_merge_hook()
            output = repo.root / "github-output"
            env = repo.env(output)
            env.pop("CODEXSDK_SYNC_BOT_TOKEN", None)

            completed = run_unchecked(
                [
                    str(repo.script()),
                    "--land-ref",
                    "tmp",
                    "--work-branch",
                    "codex/sync-test",
                    "--target-ref",
                    "rust-v0.0.1",
                    "--target-sha",
                    TARGET_SHA,
                    "--auto-merge-pr-on-failure",
                    "--require-bot-token-for-auto-merge",
                ],
                cwd=repo.repo,
                env=env,
            )

            self.assertNotEqual(completed.returncode, 0)
            self.assertIn("requires CODEXSDK_SYNC_BOT_TOKEN or a generated GitHub App token", completed.stderr)
            self.assertIn("required checks would never appear", completed.stderr)


if __name__ == "__main__":
    unittest.main()
