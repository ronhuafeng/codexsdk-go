#!/usr/bin/env python3
"""Resolve an upstream openai/codex target for protocol sync workflows."""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from dataclasses import asdict, dataclass


DEFAULT_REMOTE = "https://github.com/openai/codex.git"
RUST_TAG_RE = re.compile(r"^rust-v([0-9]+)[.]([0-9]+)[.]([0-9]+)$")
SHA_RE = re.compile(r"^[0-9a-f]{40}$")


@dataclass(frozen=True)
class UpstreamTarget:
    upstream_ref: str
    upstream_ref_kind: str
    upstream_sha: str
    target_explicit: bool


def trim_ref(value: str) -> str:
    return value.strip()


def infer_ref_kind(ref: str) -> str:
    if RUST_TAG_RE.fullmatch(ref):
        return "stable_rust_tag"
    if SHA_RE.fullmatch(ref):
        return "manual_commit"
    return "manual_ref"


def parse_stable_tag(ref: str) -> tuple[int, int, int] | None:
    match = RUST_TAG_RE.fullmatch(ref)
    if match is None:
        return None
    return tuple(int(part) for part in match.groups())


def git_ls_remote(remote: str, *patterns: str) -> str:
    completed = subprocess.run(
        ["git", "ls-remote", remote, *patterns],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    return completed.stdout


def latest_stable_rust_tag(ls_remote_output: str) -> str:
    tags: list[tuple[tuple[int, int, int], str]] = []
    for line in ls_remote_output.splitlines():
        parts = line.split()
        if len(parts) != 2:
            continue
        ref = parts[1]
        prefix = "refs/tags/"
        if not ref.startswith(prefix):
            continue
        tag = ref[len(prefix) :]
        version = parse_stable_tag(tag)
        if version is not None:
            tags.append((version, tag))
    if not tags:
        raise ValueError("unable to resolve an upstream rust-vX.Y.Z tag")
    return max(tags, key=lambda item: item[0])[1]


def first_remote_sha(ls_remote_output: str) -> str:
    for line in ls_remote_output.splitlines():
        parts = line.split()
        if len(parts) == 2 and parts[0]:
            return parts[0]
    return ""


def resolve_remote_ref(remote: str, ref: str) -> str:
    if SHA_RE.fullmatch(ref):
        return ref
    candidates = [
        f"refs/tags/{ref}^{{}}",
        f"{ref}^{{}}",
        f"refs/tags/{ref}",
        f"refs/heads/{ref}",
        ref,
    ]
    for candidate in candidates:
        sha = first_remote_sha(git_ls_remote(remote, candidate))
        if sha:
            return sha
    raise ValueError(f"unable to resolve upstream ref in openai/codex: {ref}")


def resolve_upstream(remote: str, requested_ref: str) -> UpstreamTarget:
    requested_ref = trim_ref(requested_ref)
    if requested_ref:
        upstream_ref = requested_ref
        target_explicit = True
    else:
        upstream_ref = latest_stable_rust_tag(git_ls_remote(remote, "refs/tags/rust-v*"))
        target_explicit = False
    return UpstreamTarget(
        upstream_ref=upstream_ref,
        upstream_ref_kind=infer_ref_kind(upstream_ref),
        upstream_sha=resolve_remote_ref(remote, upstream_ref),
        target_explicit=target_explicit,
    )


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--remote", default=DEFAULT_REMOTE, help="upstream Codex git remote URL")
    parser.add_argument("--upstream-ref", default="", help="optional openai/codex tag, ref, or full SHA")
    parser.add_argument("--json", action="store_true", help="print machine-readable target metadata")
    args = parser.parse_args()

    target = resolve_upstream(args.remote, args.upstream_ref)
    if args.json:
        print(json.dumps(asdict(target), sort_keys=True))
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (subprocess.CalledProcessError, ValueError) as exc:
        print(str(exc), file=sys.stderr)
        raise SystemExit(1)
