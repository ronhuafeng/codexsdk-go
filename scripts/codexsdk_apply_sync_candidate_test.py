#!/usr/bin/env python3

from __future__ import annotations

import json
import os
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, os.path.dirname(__file__))

import codexsdk_apply_sync_candidate as apply_sync


def write_json(path: Path, value: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2) + "\n", encoding="utf-8")


class ApplySyncCandidateTest(unittest.TestCase):
    def test_build_coverage_seeds_missing_fields_for_changed_object_schemas(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_json(
                root / "ChangedResponse.json",
                {
                    "title": "ChangedResponse",
                    "type": "object",
                    "required": ["value"],
                    "properties": {
                        "optional": {"type": "boolean"},
                        "value": {"type": "string"},
                    },
                },
            )
            old_coverage = {
                "fields": [],
                "methods": [],
                "schema_version": 1,
                "types": [{"schema": "ChangedResponse.json", "stability": "stable"}],
                "valid_statuses": ["supported", "supported-generated", "deferred", "intentionally-unsupported"],
            }
            coverage = apply_sync.build_coverage(
                root,
                old_coverage,
                {"entries": []},
                {"ChangedResponse.json"},
            )

            fields = {item["path"]: item for item in coverage["fields"]}
            self.assertEqual(
                fields["ChangedResponse.json#/properties/value"]["type"],
                "ChangedResponse",
            )
            self.assertTrue(fields["ChangedResponse.json#/properties/value"]["required"])
            self.assertFalse(fields["ChangedResponse.json#/properties/optional"]["required"])

    def test_build_coverage_preserves_existing_reviewed_field_for_changed_schema(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_json(
                root / "ChangedResponse.json",
                {
                    "title": "ChangedResponse",
                    "type": "object",
                    "required": ["value"],
                    "properties": {"value": {"type": "string"}},
                },
            )
            old_field = {
                "field": "value",
                "owner": "codex-go-sdk",
                "path": "ChangedResponse.json#/properties/value",
                "reason": "Reviewed custom field coverage.",
                "required": True,
                "schema": "ChangedResponse.json",
                "stability": "stable",
                "status": "supported",
                "type": "ChangedResponse",
            }
            coverage = apply_sync.build_coverage(
                root,
                {
                    "fields": [old_field],
                    "methods": [],
                    "schema_version": 1,
                    "types": [{"schema": "ChangedResponse.json", "stability": "stable"}],
                    "valid_statuses": ["supported", "supported-generated", "deferred", "intentionally-unsupported"],
                },
                {"entries": []},
                {"ChangedResponse.json"},
            )

            self.assertEqual(coverage["fields"], [old_field])


if __name__ == "__main__":
    unittest.main()
