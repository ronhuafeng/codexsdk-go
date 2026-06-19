#!/usr/bin/env python3

import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(__file__))

import codexsdk_sync_tag as sync_tag


UPSTREAM_SHA = "c73296a000000000000000000000000000000000"
SDK_SHA = "1" * 40


class SyncTagTest(unittest.TestCase):
    def test_stable_rust_tag_uses_upstream_namespace(self):
        metadata = {
            "source_ref_kind": "stable_rust_tag",
            "source_ref_name": "rust-v0.140.0",
            "source_commit": UPSTREAM_SHA,
        }
        self.assertEqual(sync_tag.tag_name(metadata), "upstream-codex-rust-v0.140.0")

    def test_manual_commit_has_no_fallback_tag(self):
        metadata = {
            "source_ref_kind": "manual_commit",
            "source_ref_name": UPSTREAM_SHA,
            "source_commit": UPSTREAM_SHA,
        }
        with self.assertRaisesRegex(ValueError, "stable_rust_tag"):
            sync_tag.tag_name(metadata)

    def test_manual_ref_has_no_fallback_tag(self):
        metadata = {
            "source_ref_kind": "manual_ref",
            "source_ref_name": "refs/heads/main",
            "source_commit": UPSTREAM_SHA,
        }
        with self.assertRaisesRegex(ValueError, "stable_rust_tag"):
            sync_tag.tag_name(metadata)

    def test_existing_base_tag_blocks_without_suffix(self):
        choice = sync_tag.choose_tag(
            "upstream-codex-rust-v0.140.0",
            {"upstream-codex-rust-v0.140.0": SDK_SHA},
            "2" * 40,
            next_suffix=False,
        )
        self.assertEqual(choice.action, "block")

    def test_existing_base_tag_uses_next_suffix(self):
        choice = sync_tag.choose_tag(
            "upstream-codex-rust-v0.140.0",
            {"upstream-codex-rust-v0.140.0": SDK_SHA},
            "2" * 40,
            next_suffix=True,
        )
        self.assertEqual(choice.action, "create")
        self.assertEqual(choice.tag_name, "upstream-codex-rust-v0.140.0-sync.2")

    def test_existing_follow_up_tag_reuses_current_commit(self):
        choice = sync_tag.choose_tag(
            "upstream-codex-rust-v0.140.0",
            {
                "upstream-codex-rust-v0.140.0": SDK_SHA,
                "upstream-codex-rust-v0.140.0-sync.2": "2" * 40,
            },
            "2" * 40,
            next_suffix=True,
        )
        self.assertEqual(choice.action, "exists")
        self.assertEqual(choice.tag_name, "upstream-codex-rust-v0.140.0-sync.2")

    def test_tag_message_includes_upstream_and_sdk_commits(self):
        metadata = {
            "source_repo": "https://github.com/openai/codex",
            "source_ref_kind": "stable_rust_tag",
            "source_ref_name": "rust-v0.140.0",
            "source_commit": UPSTREAM_SHA,
            "schema_bundle_sha256": "a" * 64,
            "codex_version": "codex-cli 0.1.0",
        }
        message = sync_tag.sync_tag_message(metadata, SDK_SHA)
        self.assertIn("upstream_ref: rust-v0.140.0", message)
        self.assertIn(f"upstream_commit: {UPSTREAM_SHA}", message)
        self.assertIn(f"codexsdk_commit: {SDK_SHA}", message)


if __name__ == "__main__":
    unittest.main()
