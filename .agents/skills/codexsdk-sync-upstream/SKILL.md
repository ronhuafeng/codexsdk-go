---
name: codexsdk-sync-upstream
description: Sync this codexsdk-go repository with a specific upstream OpenAI Codex tag, ref, or commit. Use when asked to update the checked-in Codex app-server schema baseline, compare protocol drift, refresh protocolv2 generated Go files, reconcile manifest/coverage metadata, or prepare a Codex SDK baseline update from an upstream tag/ref/commit.
---

# Codex SDK Upstream Sync

## Overview

Update the SDK by treating the checked-in app-server schema baseline as the source for generated Go code. Do not make the SDK follow the local `codex` binary implicitly during normal builds.

Use the repository's existing tracking script first, then review protocol drift before copying anything into the tree.

## Required Inputs

Collect or infer:

- upstream Codex commit or ref, preferably a full SHA after resolution
- local OpenAI Codex repo path, from the prompt, `CODEXSDK_CODEX_REPO`, or default `.cache/openai-codex`
- generator mode: default to `cargo`; use `binary` only when the provided `codex` binary is known to be built from the target commit
- output workdir, defaulting to `.cache/codexsdk-upstream-<short-sha>`

If the commit is missing and cannot be discovered locally, ask for it before changing files.

Keep upstream clones, drift artifacts, and Rust build cache under repo-local `.cache/` by default. Never check in `.cache` contents.

Use this cache topology unless the user provides an explicit alternative:

- upstream clone: `.cache/openai-codex`
- sync output: `.cache/codexsdk-upstream-<short-sha>`
- clean rerun output: `.cache/codexsdk-upstream-<short-sha>-clean`
- Rust build cache: `.cache/cargo-target/codex`

Treat `.cache/` as disposable generated state: it may be deleted and rebuilt at any time.

## Workflow

1. Inspect the current repository state.

   ```sh
   git status --short
   sed -n '1,80p' codexsdk/internal/protocolschema/appserver/v2/baseline_metadata.json
   ```

   Keep unrelated user changes intact.

2. Resolve the target commit and generate drift artifacts.

   Prepare the default cache locations:

   ```sh
   mkdir -p .cache
   if [ ! -d .cache/openai-codex/.git ]; then
     git clone https://github.com/openai/codex.git .cache/openai-codex
   else
     git -C .cache/openai-codex fetch origin
   fi
   ```

   ```sh
   CARGO_TARGET_DIR="$PWD/.cache/cargo-target/codex" \
     scripts/codexsdk_track_upstream.sh \
     --codex-repo "$PWD/.cache/openai-codex" \
     --commit <codex-commit> \
     --out "$PWD/.cache/codexsdk-upstream-<short-sha>"
   ```

   The script is intentionally read-only for the checked-in baseline. It writes candidate schemas under `schema/` and review reports under `reports/`.

3. Review the generated reports before editing.

   Read:

   - `.cache/codexsdk-upstream-<short-sha>/reports/SUMMARY.md`
   - `.cache/codexsdk-upstream-<short-sha>/reports/drift_summary.json`
   - `.cache/codexsdk-upstream-<short-sha>/reports/matrix_update_skeleton.json`

   Treat any added, removed, or changed schema as review-required. Classify the drift before updating `manifest.json` or `coverage_matrix.json`:

   - method drift: added or removed request/notification method entries
   - schema file drift: added, removed, or changed JSON schema files
   - generated Go type drift: generated struct, enum, union, or nullable field changes
   - handwritten SDK impact: facade/client/type/test/docs changes needed beyond generated code
   - coverage impact: `coverage_matrix.json` entries needed for new or changed surface

   For example, an optional nullable field added to an existing response type normally requires generated Go and focused tests, but no facade change unless handwritten SDK code exposes or interprets that field.

4. Update the checked-in baseline only after review.

   Update files under `codexsdk/internal/protocolschema/appserver/v2`:

   - copy reviewed schema changes from `.cache/codexsdk-upstream-<short-sha>/schema`
   - update `baseline_metadata.json` with public provenance only: upstream URL, full commit SHA, Codex version, source license, repo-relative source paths, schema count, and schema bundle checksum
   - update `manifest.json` according to `manifest_generation.json` rules and response mappings
   - update `coverage_matrix.json` with explicit support status, owner, reason, revisit trigger, and exit condition for new or changed surface
   - update `drift_report.json` and `matrix_update_skeleton.json` by rerunning the tracking script after the baseline matches the target schema and copying the clean reports

   Sanitize checked-in reports before staging:

   - replace local source repo paths with `https://github.com/openai/codex`
   - replace local generator worktree paths with `codex app-server generate-json-schema --experimental --out <tmpdir>`
   - include the baseline schema bundle checksum from `baseline_metadata.json`
   - keep the canonical-JSON comparison note when object member ordering is irrelevant

   Do not check in local absolute paths, `.cache/...` output paths, private repo paths, account data, or raw smoke-test transcripts.

5. Regenerate protocol code.

   ```sh
   GOWORK=off go run ./codexsdk/internal/cmd/protocolv2gen
   ```

   This updates:

   - `codexsdk/protocolv2/protocol_types.gen.go`
   - `codexsdk/protocolv2/method_registry.gen.go`

6. Reconcile handwritten SDK behavior when schema changes require it.

   Check at least:

   - `codexsdk/client.go`
   - `codexsdk/facades.go`
   - `codexsdk/types.go`
   - `codexsdk/*_test.go`
   - `README.md`, `CHANGELOG.md`, `NOTICE`, and `docs/release.md` if compatibility, provenance, or release guidance changed

   Prefer typed `protocolv2` params/responses over raw JSON-RPC escape hatches.

7. Validate.

   ```sh
   git ls-files -z -- '*.go' ':!:vendor/**' | xargs -0 gofmt -w
   GOWORK=off go vet ./...
   GOWORK=off go test ./...

   tmp="$(mktemp -d)"
   GOWORK=off go run ./codexsdk/internal/cmd/protocolv2gen -out "$tmp"
   diff -u codexsdk/protocolv2/method_registry.gen.go "$tmp/method_registry.gen.go"
   diff -u codexsdk/protocolv2/protocol_types.gen.go "$tmp/protocol_types.gen.go"

   rg -n "/Users/|/home/|\\.cache/codexsdk-upstream|\\.cache/openai-codex" \
     codexsdk/internal/protocolschema/appserver/v2 \
     .agents/skills/codexsdk-sync-upstream/SKILL.md \
     .gitignore
   ```

   The path scan may match intentional relative `.cache/...` instructions in this skill or `.gitignore`; it must not find local absolute paths in checked-in schema metadata or reports.

   Run the real app-server smoke test only when the user explicitly wants it or the change affects lifecycle behavior:

   ```sh
   CODEXSDK_REAL_APP_SERVER_SMOKE=1 \
   CODEXSDK_REAL_APP_SERVER_MODEL=<model> \
   GOWORK=off go test ./codexsdk -run TestRealAppServerSmokeStartResumeFork -count=1
   ```

8. Before committing, check whether the selected upstream target moved.

   ```sh
   git ls-remote --tags --refs https://github.com/openai/codex.git 'refs/tags/rust-v*' \
     | awk '{print $2}' \
     | sed 's#refs/tags/##' \
     | grep -E '^rust-v[0-9]+[.][0-9]+[.][0-9]+$' \
     | sort -V \
     | tail -n 1
   ```

   For the default scheduled workflow, compare against the latest stable `rust-vX.Y.Z` tag, not `main`. If the latest stable tag changed after the selected target, run the tracking script against the new tag without changing checked-in files first. If the new tag is drift-clean relative to the updated baseline, do not chase it with a provenance-only commit unless the user asked for exact latest provenance. If it has real protocol drift, stop and explain the new target so the user can choose whether to continue. Do not enter an unbounded loop chasing moving upstream refs.

9. After pushing, monitor repository automation when the task is to solve a drift issue.

   Watch the push CI run and the Codex Upstream Protocol Drift workflow run. Cold GitHub runners may spend several minutes compiling Rust before the drift report step advances.

   Confirm the drift issue outcome:

   - if drift is clean, the workflow should close the existing drift issue
   - if drift remains, the workflow should update the existing drift issue rather than creating duplicates
   - if the selected upstream tag/ref moved or a newer stable tag exists, report the exact tag/ref and commit and whether the remaining drift is real or clean

## Decision Rules

- If drift is clean and the user only asked to check a commit, report that no SDK update is needed.
- If drift is clean but the user explicitly asked to move the baseline provenance to that commit, update provenance and clean drift artifacts without changing schema-derived Go output.
- If generated Go fails because a new schema shape is unsupported, update `codexsdk/internal/protocolgen` with a reviewed generation rule and focused tests before regenerating.
- If a method disappears upstream, preserve compatibility only when it is safe and intentional; otherwise document the breaking change.
- If upstream adds experimental surface, mark it experimental unless it appears in the non-experimental schema comparison and the existing manifest rules say otherwise.
- If a scheduled or manual drift issue already exists, update or close that issue; do not create a new issue for each upstream commit while the old one is unresolved.
