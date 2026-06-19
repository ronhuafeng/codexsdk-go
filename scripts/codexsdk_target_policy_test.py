#!/usr/bin/env python3

import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(__file__))

import codexsdk_target_policy as policy


OLD_SHA = "1" * 40
NEW_SHA = "2" * 40


def baseline(ref_name, ref_kind, commit=OLD_SHA):
    return {
        "source_commit": commit,
        "source_ref_name": ref_name,
        "source_ref_kind": ref_kind,
    }


def decide(base, target_ref, target_kind, target_sha=NEW_SHA, explicit=False, downgrade=False):
    return policy.evaluate_policy(
        base,
        target_ref=target_ref,
        target_kind=target_kind,
        target_sha=target_sha,
        target_explicit=explicit,
        mode="manual",
        allow_downgrade=downgrade,
    )


class TargetPolicyTest(unittest.TestCase):
    def test_same_commit_skips(self):
        decision = decide(
            baseline("rust-v0.140.0", "stable_rust_tag"),
            "rust-v0.141.0",
            "stable_rust_tag",
            target_sha=OLD_SHA,
        )
        self.assertEqual(decision["decision"], "skip")

    def test_stable_tag_forward_allows_without_ancestry(self):
        decision = decide(
            baseline("rust-v0.140.0", "stable_rust_tag"),
            "rust-v0.141.0",
            "stable_rust_tag",
        )
        self.assertEqual(decision["decision"], "allow")
        self.assertIn("moves forward", decision["reason"])

    def test_stable_tag_downgrade_blocks_by_default(self):
        decision = decide(
            baseline("rust-v0.141.0", "stable_rust_tag"),
            "rust-v0.140.0",
            "stable_rust_tag",
            explicit=True,
        )
        self.assertEqual(decision["decision"], "block")

    def test_stable_tag_downgrade_allows_when_explicitly_enabled(self):
        decision = decide(
            baseline("rust-v0.141.0", "stable_rust_tag"),
            "rust-v0.140.0",
            "stable_rust_tag",
            explicit=True,
            downgrade=True,
        )
        self.assertEqual(decision["decision"], "allow")
        self.assertIn("downgrade", decision["reason"])

    def test_stable_tag_same_version_different_sha_blocks(self):
        decision = decide(
            baseline("rust-v0.140.0", "stable_rust_tag"),
            "rust-v0.140.0",
            "stable_rust_tag",
        )
        self.assertEqual(decision["decision"], "block")
        self.assertIn("peeled commit changed", decision["reason"])

    def test_manual_baseline_to_default_stable_blocks(self):
        decision = decide(
            baseline(OLD_SHA, "manual_commit"),
            "rust-v0.141.0",
            "stable_rust_tag",
            explicit=False,
        )
        self.assertEqual(decision["decision"], "block")
        self.assertIn("switch tracks", decision["reason"])

    def test_manual_baseline_to_explicit_stable_allows_track_switch(self):
        decision = decide(
            baseline(OLD_SHA, "manual_commit"),
            "rust-v0.141.0",
            "stable_rust_tag",
            explicit=True,
        )
        self.assertEqual(decision["decision"], "allow")
        self.assertIn("track switch", decision["reason"])

    def test_explicit_manual_commit_allows(self):
        decision = decide(
            baseline("rust-v0.140.0", "stable_rust_tag"),
            NEW_SHA,
            "manual_commit",
            explicit=True,
        )
        self.assertEqual(decision["decision"], "allow")
        self.assertIn("manual upstream target", decision["reason"])

    def test_invalid_stable_target_blocks(self):
        decision = decide(
            baseline("rust-v0.140.0", "stable_rust_tag"),
            "main",
            "stable_rust_tag",
        )
        self.assertEqual(decision["decision"], "block")
        self.assertIn("not a rust-vX.Y.Z tag", decision["reason"])


if __name__ == "__main__":
    unittest.main()
