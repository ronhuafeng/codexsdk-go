#!/usr/bin/env python3
"""Write drift issue title, body, and comment artifacts from drift_summary.json."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
from string import Template
from typing import Any


REPO_ROOT = Path(__file__).resolve().parents[1]
DEFAULT_BODY_TEMPLATE = REPO_ROOT / ".github/templates/codexsdk-drift-issue-body.md"
DEFAULT_COMMENT_TEMPLATE = REPO_ROOT / ".github/templates/codexsdk-drift-issue-comment.md"


def short_list(items: list[str], limit: int = 20) -> str:
    if not items:
        return "_none_"
    lines = [f"- `{item}`" for item in items[:limit]]
    if len(items) > limit:
        lines.append(f"- ...and {len(items) - limit} more")
    return "\n".join(lines)


def method_diff_markdown(method_diff: dict[str, Any]) -> str:
    method_lines: list[str] = []
    for schema, diff in method_diff.items():
        added = diff.get("added", [])
        removed = diff.get("removed", [])
        if added or removed:
            method_lines.extend(
                [
                    f"### {schema}",
                    "",
                    "Added:",
                    short_list(added),
                    "",
                    "Removed:",
                    short_list(removed),
                    "",
                ]
            )
    if not method_lines:
        return "_No aggregate method additions or removals._"
    return "\n".join(method_lines).rstrip()


def render_artifacts(
    *,
    baseline_sha: str,
    drift: dict[str, Any],
    drift_sha: str,
    run_url: str,
    upstream_ref: str,
    upstream_ref_kind: str,
    upstream_sha: str,
    body_template: str | None = None,
    comment_template: str | None = None,
) -> dict[str, str]:
    status = drift["status"]
    file_diff = drift["file_diff"]
    method_diff = drift["method_diff"]
    upstream_display = f"`{upstream_ref}` (`{upstream_sha}`)"
    title = f"Protocol drift detected against openai/codex {upstream_ref}"
    body_template = body_template if body_template is not None else DEFAULT_BODY_TEMPLATE.read_text(encoding="utf-8")
    comment_template = comment_template if comment_template is not None else DEFAULT_COMMENT_TEMPLATE.read_text(encoding="utf-8")

    substitutions = {
        "ADDED_COUNT": str(len(file_diff["added"])),
        "ADDED_SCHEMAS": short_list(file_diff["added"]),
        "BASELINE_SHA": baseline_sha,
        "CHANGED_COUNT": str(len(file_diff["changed"])),
        "CHANGED_SCHEMAS": short_list(file_diff["changed"]),
        "DRIFT_SHA": drift_sha,
        "METHOD_DIFF": method_diff_markdown(method_diff),
        "REMOVED_COUNT": str(len(file_diff["removed"])),
        "REMOVED_SCHEMAS": short_list(file_diff["removed"]),
        "RUN_URL": run_url,
        "STATUS": status,
        "UPSTREAM_DISPLAY": upstream_display,
        "UPSTREAM_REF": upstream_ref,
        "UPSTREAM_REF_KIND": upstream_ref_kind,
        "UPSTREAM_SHA": upstream_sha,
    }
    body = Template(body_template).substitute(substitutions)
    comment = Template(comment_template).substitute(substitutions)
    if not body.endswith("\n"):
        body += "\n"
    if not comment.endswith("\n"):
        comment += "\n"

    return {
        "comment": comment,
        "status": status,
        "title": title,
        "body": body,
    }


def write_github_output(status: str, drift_sha: str) -> None:
    output_path = os.environ.get("GITHUB_OUTPUT")
    if not output_path:
        return
    with open(output_path, "a", encoding="utf-8") as output:
        output.write(f"status={status}\n")
        output.write(f"drift_sha={drift_sha}\n")


def write_artifacts(
    *,
    artifact_dir: Path,
    baseline_sha: str,
    run_url: str,
    upstream_ref: str,
    upstream_ref_kind: str,
    upstream_sha: str,
    body_template: Path = DEFAULT_BODY_TEMPLATE,
    comment_template: Path = DEFAULT_COMMENT_TEMPLATE,
) -> dict[str, str]:
    reports = artifact_dir / "reports"
    drift_path = reports / "drift_summary.json"
    drift = json.loads(drift_path.read_text(encoding="utf-8"))
    drift_sha = hashlib.sha256(drift_path.read_bytes()).hexdigest()
    rendered = render_artifacts(
        baseline_sha=baseline_sha,
        body_template=body_template.read_text(encoding="utf-8"),
        comment_template=comment_template.read_text(encoding="utf-8"),
        drift=drift,
        drift_sha=drift_sha,
        run_url=run_url,
        upstream_ref=upstream_ref,
        upstream_ref_kind=upstream_ref_kind,
        upstream_sha=upstream_sha,
    )

    (artifact_dir / "issue-title.txt").write_text(rendered["title"] + "\n", encoding="utf-8")
    (artifact_dir / "issue-body.md").write_text(rendered["body"], encoding="utf-8")
    (artifact_dir / "issue-comment.md").write_text(rendered["comment"], encoding="utf-8")
    write_github_output(rendered["status"], drift_sha)

    return {
        "drift_sha": drift_sha,
        "status": rendered["status"],
        "title": rendered["title"],
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--artifact-dir", required=True, type=Path, help="drift artifact directory containing reports/")
    parser.add_argument("--baseline-sha", required=True, help="current checked-in upstream baseline commit")
    parser.add_argument("--body-template", type=Path, default=DEFAULT_BODY_TEMPLATE, help="Markdown issue body template")
    parser.add_argument("--comment-template", type=Path, default=DEFAULT_COMMENT_TEMPLATE, help="Markdown issue comment template")
    parser.add_argument("--run-url", required=True, help="GitHub Actions run URL")
    parser.add_argument("--upstream-ref", required=True, help="selected upstream ref")
    parser.add_argument("--upstream-ref-kind", required=True, help="selected upstream ref kind")
    parser.add_argument("--upstream-sha", required=True, help="selected upstream commit SHA")
    parser.add_argument("--json", action="store_true", help="print a machine-readable summary")
    args = parser.parse_args()

    payload = write_artifacts(
        artifact_dir=args.artifact_dir,
        baseline_sha=args.baseline_sha,
        body_template=args.body_template,
        comment_template=args.comment_template,
        run_url=args.run_url,
        upstream_ref=args.upstream_ref,
        upstream_ref_kind=args.upstream_ref_kind,
        upstream_sha=args.upstream_sha,
    )
    if args.json:
        print(json.dumps(payload, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
