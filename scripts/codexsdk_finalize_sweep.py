#!/usr/bin/env python3
"""Select a merged upstream sync PR that is ready for finalization."""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
from pathlib import Path
from typing import Any


FINALIZE_EVIDENCE = "Finalized upstream protocol sync."


def metadata_from_body(body: str) -> dict[str, str]:
    match = re.search(r"<!--\s*codexsdk-upstream-sync\s*(.*?)-->", body or "", re.S)
    if match is None:
        return {}
    metadata: dict[str, str] = {}
    for raw_line in match.group(1).splitlines():
        line = raw_line.strip()
        if not line or ":" not in line:
            continue
        key, value = line.split(":", 1)
        metadata[key.strip()] = value.strip()
    return metadata


def read_json(path: Path) -> Any:
    return json.loads(path.read_text(encoding="utf-8"))


def gh_issue(issue_number: str) -> dict[str, Any]:
    completed = subprocess.run(
        ["gh", "issue", "view", issue_number, "--json", "state,comments,url"],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    return json.loads(completed.stdout)


def issue_from_dir(issue_dir: Path, issue_number: str) -> dict[str, Any]:
    return read_json(issue_dir / f"issue-{issue_number}.json")


def has_finalize_evidence(issue: dict[str, Any]) -> bool:
    return any(
        FINALIZE_EVIDENCE in (comment.get("body") or "")
        for comment in issue.get("comments", [])
    )


def resolve_metadata(*, pr: dict[str, Any], inputs: dict[str, str], default_branch: str) -> dict[str, str]:
    if pr and pr.get("state") != "MERGED":
        raise ValueError(f"sync PR #{pr.get('number')} is not merged; state={pr.get('state')}")

    body_metadata = metadata_from_body(pr.get("body", ""))
    merge_commit = (pr.get("mergeCommit") or {}).get("oid", "")
    return {
        "drift_sha": inputs.get("drift_sha", "") or body_metadata.get("drift_sha256", ""),
        "issue_number": inputs.get("issue_number", "") or body_metadata.get("issue_number", ""),
        "landed_commit": inputs.get("landed_commit", "") or merge_commit,
        "landed_ref": inputs.get("landed_ref", "") or pr.get("baseRefName", "") or default_branch,
        "pr_number": inputs.get("pr_number", "") or str(pr.get("number") or ""),
        "pr_url": pr.get("url", ""),
        "upstream_ref": inputs.get("upstream_ref", "") or body_metadata.get("upstream_ref", ""),
        "upstream_ref_kind": inputs.get("upstream_ref_kind", "") or body_metadata.get("upstream_ref_kind", ""),
        "upstream_sha": inputs.get("upstream_sha", "") or body_metadata.get("upstream_commit", ""),
    }


def select_candidate(
    *,
    active_runs: list[dict[str, Any]],
    default_branch: str,
    default_head: str,
    prs: list[dict[str, Any]],
    issue_lookup,
) -> tuple[dict[str, str], list[str]]:
    skipped: list[str] = []
    if active_runs:
        skipped.append("finalize run is already queued or in progress")
        return {"dispatch": "false"}, skipped

    candidates: list[tuple[dict[str, Any], dict[str, str]]] = []
    for pr in prs:
        metadata = metadata_from_body(pr.get("body") or "")
        if not metadata:
            continue
        number = pr.get("number")
        if metadata.get("phase") != "fix":
            skipped.append(f"PR #{number}: sync metadata phase is not fix")
            continue
        issue_number = metadata.get("issue_number", "")
        if not issue_number:
            skipped.append(f"PR #{number}: missing issue_number metadata")
            continue
        merge_commit = (pr.get("mergeCommit") or {}).get("oid", "")
        if merge_commit != default_head:
            skipped.append(f"PR #{number}: merge commit is not current {default_branch} head")
            continue
        candidates.append((pr, metadata))

    for pr, metadata in candidates:
        issue_number = metadata["issue_number"]
        issue = issue_lookup(issue_number)
        if issue.get("state") != "OPEN":
            skipped.append(f"PR #{pr['number']}: issue #{issue_number} is {issue.get('state')}")
            continue
        if has_finalize_evidence(issue):
            skipped.append(f"PR #{pr['number']}: issue #{issue_number} already has finalize evidence")
            continue
        return {
            "dispatch": "true",
            "issue_number": issue_number,
            "pr_number": str(pr["number"]),
        }, skipped

    return {"dispatch": "false"}, skipped


def write_outputs(path: str | None, values: dict[str, str]) -> None:
    if not path:
        return
    with Path(path).open("a", encoding="utf-8") as output:
        for key, value in values.items():
            output.write(f"{key}={value}\n")


def write_summary(path: str | None, default_branch: str, default_head: str, result: dict[str, str], skipped: list[str]) -> None:
    if not path:
        return
    with Path(path).open("a", encoding="utf-8") as summary:
        summary.write(f"Default branch {default_branch} head: {default_head}\n\n")
        summary.write("## Finalize sweep candidates\n\n")
        if skipped:
            summary.write("Skipped:\n")
            for item in skipped:
                summary.write(f"- {item}\n")
        if result["dispatch"] == "true":
            summary.write(
                f"\nSelected finalize candidate PR #{result['pr_number']} and issue #{result['issue_number']}.\n"
            )
        else:
            summary.write("\nNo pending sync PR matched the current default branch head and open drift issue.\n")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--active-runs-json", type=Path)
    parser.add_argument("--default-branch", required=True)
    parser.add_argument("--default-head")
    parser.add_argument("--github-output", default=os.environ.get("GITHUB_OUTPUT", ""))
    parser.add_argument("--input-drift-sha", default="")
    parser.add_argument("--input-issue-number", default="")
    parser.add_argument("--input-landed-commit", default="")
    parser.add_argument("--input-landed-ref", default="")
    parser.add_argument("--input-pr-number", default="")
    parser.add_argument("--input-upstream-ref", default="")
    parser.add_argument("--input-upstream-ref-kind", default="")
    parser.add_argument("--input-upstream-sha", default="")
    parser.add_argument("--merged-prs-json", type=Path)
    parser.add_argument("--sync-pr-json", type=Path, help="resolve finalize metadata from a PR JSON object instead of selecting a candidate")
    parser.add_argument("--summary", default=os.environ.get("GITHUB_STEP_SUMMARY", ""))
    parser.add_argument("--issue-json-dir", type=Path, help="test hook: read issue-<number>.json files instead of gh")
    args = parser.parse_args()

    if args.sync_pr_json is not None:
        try:
            result = resolve_metadata(
                pr=read_json(args.sync_pr_json),
                inputs={
                    "drift_sha": args.input_drift_sha,
                    "issue_number": args.input_issue_number,
                    "landed_commit": args.input_landed_commit,
                    "landed_ref": args.input_landed_ref,
                    "pr_number": args.input_pr_number,
                    "upstream_ref": args.input_upstream_ref,
                    "upstream_ref_kind": args.input_upstream_ref_kind,
                    "upstream_sha": args.input_upstream_sha,
                },
                default_branch=args.default_branch,
            )
        except ValueError as exc:
            raise SystemExit(str(exc))
        write_outputs(args.github_output, result)
        print(json.dumps({"result": result}, indent=2, sort_keys=True))
        return 0

    if args.active_runs_json is None or args.default_head is None or args.merged_prs_json is None:
        parser.error("--active-runs-json, --default-head, and --merged-prs-json are required unless --sync-pr-json is used")

    active_runs = read_json(args.active_runs_json)
    prs = read_json(args.merged_prs_json)
    issue_lookup = gh_issue if args.issue_json_dir is None else lambda issue_number: issue_from_dir(args.issue_json_dir, issue_number)

    result, skipped = select_candidate(
        active_runs=active_runs,
        default_branch=args.default_branch,
        default_head=args.default_head,
        prs=prs,
        issue_lookup=issue_lookup,
    )
    write_outputs(args.github_output, result)
    write_summary(args.summary, args.default_branch, args.default_head, result, skipped)
    print(json.dumps({"result": result, "skipped": skipped}, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
