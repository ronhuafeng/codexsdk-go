#!/usr/bin/env python3
"""Report generated compatibility impact from classified manifests."""

from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any


def load_manifest(path: Path) -> dict[str, Any]:
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"manifest {path} is not an object")
    return value


def surface_index(manifest: dict[str, Any]) -> dict[tuple[str, str], str]:
    result: dict[tuple[str, str], str] = {}
    for raw in manifest.get("surface", []):
        if not isinstance(raw, dict):
            raise ValueError("manifest surface entry is not an object")
        kind = raw.get("kind")
        name = raw.get("name")
        stability = raw.get("stability")
        if not all(isinstance(value, str) and value for value in (kind, name, stability)):
            raise ValueError(f"manifest surface entry is unclassified: {raw!r}")
        key = (kind, name)
        if key in result:
            raise ValueError(f"duplicate manifest surface identity: {kind} {name}")
        result[key] = stability
    if not result:
        raise ValueError("manifest has no classified generated surface")
    return result


def compatibility_report(base: dict[str, Any], target: dict[str, Any]) -> dict[str, Any]:
    before = surface_index(base)
    after = surface_index(target)
    added = []
    removed = []
    reclassified = []
    for kind, name in sorted(after.keys() - before.keys()):
        added.append({"kind": kind, "name": name, "classification": after[(kind, name)]})
    for kind, name in sorted(before.keys() - after.keys()):
        removed.append({"kind": kind, "name": name, "classification": before[(kind, name)]})
    for kind, name in sorted(before.keys() & after.keys()):
        if before[(kind, name)] != after[(kind, name)]:
            reclassified.append(
                {
                    "kind": kind,
                    "name": name,
                    "from": before[(kind, name)],
                    "to": after[(kind, name)],
                }
            )
    return {
        "policy": "classification_is_metadata_not_a_semver_exemption",
        "compatibility_impact": "incompatible" if removed else "additive_or_metadata_only",
        "added": added,
        "removed": removed,
        "reclassified": reclassified,
        "counts_by_classification": {
            stability: {
                "added": sum(item["classification"] == stability for item in added),
                "removed": sum(item["classification"] == stability for item in removed),
                "reclassified_from": sum(item["from"] == stability for item in reclassified),
                "reclassified_to": sum(item["to"] == stability for item in reclassified),
            }
            for stability in ("stable", "experimental", "mixed")
        },
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base-manifest", required=True, type=Path)
    parser.add_argument("--target-manifest", required=True, type=Path)
    parser.add_argument("--out", type=Path)
    args = parser.parse_args()
    report = compatibility_report(load_manifest(args.base_manifest), load_manifest(args.target_manifest))
    encoded = json.dumps(report, indent=2, ensure_ascii=False) + "\n"
    if args.out:
        args.out.write_text(encoded, encoding="utf-8")
    else:
        print(encoded, end="")
    return 1 if report["compatibility_impact"] == "incompatible" else 0


if __name__ == "__main__":
    raise SystemExit(main())
