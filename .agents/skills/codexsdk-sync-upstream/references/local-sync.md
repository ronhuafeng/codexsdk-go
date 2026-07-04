# Local Sync Reference

Use this reference for local baseline syncs, target resolution, drift review, checked-in baseline updates, regeneration, validation, and upstream sync tagging.

## Contents

- [Inputs And Cache Layout](#inputs-and-cache-layout)
- [Branch Safety](#branch-safety)
- [Resolve Target And Policy](#resolve-target-and-policy)
- [Generate And Review Drift](#generate-and-review-drift)
- [Update Checked-In Baseline](#update-checked-in-baseline)
- [Regenerate And Reconcile SDK](#regenerate-and-reconcile-sdk)
- [Validate](#validate)
- [Target Movement](#target-movement)
- [Tag Landed Syncs](#tag-landed-syncs)
- [Decision Rules](#decision-rules)

## Inputs And Cache Layout

Keep upstream clones, drift artifacts, and Rust build cache under repo-local `.cache/` by default. Never check in `.cache` contents.

Use this cache topology unless the user provides an explicit alternative:

- upstream clone: `.cache/openai-codex`
- sync output: `.cache/codexsdk-upstream-<short-sha>`
- clean rerun output: `.cache/codexsdk-upstream-<short-sha>-clean`
- Rust build cache: `.cache/cargo-target/codex`

Treat `.cache/` as disposable generated state.

Record selected target provenance in checked-in metadata as:

- `source_ref_name`
- `source_ref_kind` (`stable_rust_tag`, `manual_ref`, or `manual_commit`)
- peeled full `source_commit`

## Branch Safety

Start by inspecting current state:

```sh
git status --short
git branch --show-current
sed -n '1,80p' codexsdk/internal/protocolschema/appserver/v2/baseline_metadata.json
```

Keep unrelated user changes intact.

Do not commit sync work on `dependabot/*`, `renovate/*`, `release/*`, `release-*`, `hotfix/*`, or `hotfix-*` unless the user explicitly asked to use that branch. If the branch is clearly for an unrelated PR or feature, stop before editing and create a sync branch such as `codex/sync-upstream-rust-vX.Y.Z` after resolving the target.

## Resolve Target And Policy

Use the canonical resolver. Do not hand-roll `sort`, prerelease filtering, or annotated-tag peeling in shell.

```sh
mkdir -p .cache
target_json=.cache/codexsdk-upstream-target.json
if [ -n "<requested_ref>" ]; then
  python3 scripts/codexsdk_resolve_upstream.py --upstream-ref "<requested_ref>" --json > "$target_json"
else
  python3 scripts/codexsdk_resolve_upstream.py --latest-stable --json > "$target_json"
fi
```

The resolver must provide:

- `upstream_ref`
- `upstream_ref_kind`
- `tag_sha`
- `peeled_commit_sha` / `upstream_sha`
- `target_explicit`

Before generating drift, run the policy gate:

```sh
python3 scripts/codexsdk_target_policy.py \
  --baseline codexsdk/internal/protocolschema/appserver/v2/baseline_metadata.json \
  --target-ref <upstream_ref> \
  --target-kind <stable_rust_tag|manual_ref|manual_commit> \
  --target-sha <upstream_sha> \
  --target-explicit <true|false> \
  --mode <scheduled|manual> \
  --allow-downgrade false
```

Policy decisions:

- `allow`: continue with drift generation
- `skip`: baseline already points at the selected target; close or update drift issue if needed and stop
- `block`: stop without generating drift because this is a track switch, downgrade, moved tag, or ambiguous target

Do not convert a `block` decision into a protocol drift issue. Report the exact baseline target, requested target, and policy reason instead.

## Generate And Review Drift

Generate drift artifacts only after policy allows the target:

```sh
CARGO_TARGET_DIR="$PWD/.cache/cargo-target/codex" \
  scripts/codexsdk_track_upstream.sh \
    --codex-repo "$PWD/.cache/openai-codex" \
    --commit <upstream_sha> \
    --source-ref <upstream_ref> \
    --source-ref-kind <upstream_ref_kind> \
    --out "$PWD/.cache/codexsdk-upstream-<short-sha>"
```

Let `scripts/codexsdk_track_upstream.sh` fetch the selected target narrowly. Do not run broad `git fetch origin --tags` or broad upstream refreshes unless the cache is missing or the user explicitly asks to refresh it.

Review before editing:

- `.cache/codexsdk-upstream-<short-sha>/reports/SUMMARY.md`
- `.cache/codexsdk-upstream-<short-sha>/reports/drift_summary.json`
- `.cache/codexsdk-upstream-<short-sha>/reports/matrix_update_skeleton.json`

Preserve a compact pre-sync drift summary before overwriting checked-in clean reports:

- files added, changed, and removed
- method deltas and added methods
- stable vs experimental classification for added methods
- handwritten SDK and coverage implications

Classify drift before updating `manifest.json`, `coverage_matrix.json`, or handwritten SDK files:

- method drift
- schema file drift
- generated Go type drift
- SDK surface class: `metadata-only`, `generated-only`, `public-facade-required`, or `ignored-internal`
- handwritten SDK impact, with a short reason for each file
- coverage impact for new or changed surface

If new methods appear, generate a non-experimental schema from the same target once and compare method presence before deciding public SDK surface:

```sh
repo="$PWD"
stable_out="$repo/.cache/codexsdk-upstream-<short-sha>-stable/schema"
stable_worktree="$repo/.cache/codexsdk-upstream-<short-sha>-stable/codex-worktree"
mkdir -p "$stable_out"
git -C "$repo/.cache/openai-codex" worktree add --detach "$stable_worktree" <upstream_sha>
(
  cd "$stable_worktree/codex-rs"
  CARGO_TARGET_DIR="$repo/.cache/cargo-target/codex" \
    cargo run -p codex-cli -- app-server generate-json-schema --out "$stable_out"
)
```

Tracking reports may not capture every field-level consequence. If generator output, coverage, or manifest validation points at a stale field, inspect the changed object schema properties and requiredness before adding code.

Keep implementation minimal:

- prefer removing stale entries over preserving removed upstream surface
- add focused generator support and focused tests only for schema shapes required by the selected target
- keep `codexsdk_schema_diff.py` and `codexsdk_sync_state.py` report-only unless the user explicitly asks for an automated updater design
- do not add a public facade for experimental or internal surface unless asked or existing manifest rules classify it as supported public SDK surface
- do not add raw JSON-RPC passthrough helpers when a typed `protocolv2` path exists
- do not add empty coverage, manifest, or test skeletons for unchanged methods

## Update Checked-In Baseline

Update files under `codexsdk/internal/protocolschema/appserver/v2` only after review:

- copy reviewed schema changes from the candidate `schema/`
- update `baseline_metadata.json` with public provenance only
- update `manifest.json` according to `manifest_generation.json` rules and response mappings
- update `coverage_matrix.json` with explicit support status, owner, reason, revisit trigger, and exit condition for new or changed surface
- update `drift_report.json` and `matrix_update_skeleton.json` by comparing the updated baseline with the trusted candidate schema and copying clean reports

Use compare-only for the final clean report only when the candidate schema was generated from the resolved target in this run:

```sh
scripts/codexsdk_track_upstream.sh \
  --compare-only \
  --baseline codexsdk/internal/protocolschema/appserver/v2 \
  --candidate "$PWD/.cache/codexsdk-upstream-<short-sha>/schema" \
  --commit <upstream_sha> \
  --source-ref <upstream_ref> \
  --source-ref-kind <upstream_ref_kind> \
  --out "$PWD/.cache/codexsdk-upstream-<short-sha>-clean" \
  --verbose
```

Run full tracking instead of compare-only if candidate provenance is uncertain, the generator changed, the selected target moved, or sync-state reports metadata mismatch.

Sanitize checked-in reports before staging:

- replace local source repo paths with `https://github.com/openai/codex`
- replace local generator worktree paths with `codex app-server generate-json-schema --experimental --out <tmpdir>`
- include the baseline schema bundle checksum from `baseline_metadata.json`
- keep the canonical-JSON comparison note when object member ordering is irrelevant

Do not check in local absolute paths, `.cache/...` output paths, private repo paths, account data, or raw smoke-test transcripts.

## Regenerate And Reconcile SDK

Regenerate protocol code:

```sh
GOWORK=off go run ./codexsdk/internal/cmd/protocolv2gen
```

This updates:

- `codexsdk/protocolv2/protocol_types.gen.go`
- `codexsdk/protocolv2/method_registry.gen.go`

Check handwritten SDK files only when schema changes require it:

- `codexsdk/client.go`
- `codexsdk/facades.go`
- `codexsdk/types.go`
- `codexsdk/*_test.go`
- `README.md`, `CHANGELOG.md`, `NOTICE`, and `docs/release.md` only if compatibility, provenance, or release guidance changed

Before editing handwritten SDK files, write down which reviewed drift item requires the change. A handwritten change is justified only when it preserves compatibility, exposes an already-supported stable method, fixes an existing facade broken by the new schema, or updates tests/docs for a real user-visible behavior change.

Prefer typed `protocolv2` params/responses over raw JSON-RPC escape hatches. Leave experimental, internal, or newly discovered upstream-only surface in generated `protocolv2` unless the user asks for a public facade.

## Validate

Run:

```sh
git ls-files -z -- '*.go' ':!:vendor/**' | xargs -0 gofmt -w
GOWORK=off go vet ./...
GOWORK=off go test ./...

tmp="$(mktemp -d)"
GOWORK=off go run ./codexsdk/internal/cmd/protocolv2gen -out "$tmp"
diff -u codexsdk/protocolv2/method_registry.gen.go "$tmp/method_registry.gen.go"
diff -u codexsdk/protocolv2/protocol_types.gen.go "$tmp/protocol_types.gen.go"

git diff --check
git diff --name-status

python3 scripts/codexsdk_sync_state.py \
  --baseline codexsdk/internal/protocolschema/appserver/v2

rg -n "/Users/|/home/|\\.cache/codexsdk-upstream|\\.cache/openai-codex" \
  codexsdk/internal/protocolschema/appserver/v2 \
  .agents/skills/codexsdk-sync-upstream \
  .gitignore
```

`scripts/codexsdk_sync_state.py` prints nothing when the checked-in baseline is valid. On failure, read stderr findings or rerun with `--json`.

The path scan may match intentional relative `.cache/...` instructions in this skill or `.gitignore`. It must not find local absolute paths in checked-in schema metadata or reports.

If validation fails on missing coverage fields or generated constants, first check for stale manifest, coverage, or handwritten references to removed upstream methods, schemas, or fields before adding abstractions.

If `protocolv2gen` fails, triage in this order:

| Symptom | Check first |
| --- | --- |
| Unknown scalar or alias type | scalar alias mapping in `codexsdk/internal/protocolgen` |
| Nullable `$ref` or pointer mismatch | nullable `$ref` dependency handling |
| Missing generated struct/field | generated struct checkpoint tests |
| Missing method constant or handler metadata | method registry count and manifest entries |
| Unexpected protocol type count | protocol type count tests and stale removed schemas |

Review `git diff --name-status` against the drift classification before staging:

- schema JSON, checked-in drift reports, metadata, manifest, coverage matrix, and generated `protocolv2` files are expected for real schema drift
- handwritten SDK files are expected only for `public-facade-required` or compatibility-preserving changes
- README, CHANGELOG, NOTICE, and release docs are expected only for provenance, compatibility, or release guidance changes
- no stale manifest or coverage entries may remain for removed upstream methods

Run the real app-server smoke test only when the user explicitly wants it or lifecycle behavior changed:

```sh
CODEXSDK_REAL_APP_SERVER_SMOKE=1 \
CODEXSDK_REAL_APP_SERVER_MODEL=<model> \
GOWORK=off go test ./codexsdk -run TestRealAppServerSmokeStartResumeFork -count=1
```

## Target Movement

Before committing, check whether the selected upstream target moved:

```sh
python3 scripts/codexsdk_resolve_upstream.py --latest-stable --json
```

For a default scheduled run, compare against the latest stable `rust-vX.Y.Z` tag, not `main`.

Target-movement rules:

- if the current baseline `source_ref_kind` is `stable_rust_tag`, move only forward by semantic tag version
- if the current baseline came from `manual_ref` or `manual_commit`, do not automatically move it to an older stable tag or unrelated ref
- if the latest stable tag changed after the selected target, run tracking against the new tag without changing checked-in files first
- if the new tag is drift-clean relative to the updated baseline, do not chase it with a provenance-only commit unless the user asked for exact latest provenance
- if it has real protocol drift, stop and explain the new target so the user can choose whether to continue

Do not enter an unbounded loop chasing moving upstream refs.

If Cargo fails because crate downloads hit low-speed timeouts, rerun the failed generation command with retry environment rather than changing code:

```sh
CARGO_HTTP_TIMEOUT=600 \
CARGO_HTTP_LOW_SPEED_LIMIT=0 \
CARGO_REGISTRIES_CRATES_IO_PROTOCOL=sparse \
CARGO_TARGET_DIR="$PWD/.cache/cargo-target/codex" \
  scripts/codexsdk_track_upstream.sh ...
```

## Tag Landed Syncs

Tag only after a successful baseline sync commit exists on the landing ref. Do not tag drift-check-only work, failed sync attempts, unmerged PR heads, or uncommitted changes.

```sh
python3 scripts/codexsdk_sync_tag.py --json
python3 scripts/codexsdk_sync_tag.py --create --push origin
```

Tag rules:

- stable upstream tags use `upstream-codex-rust-vX.Y.Z`
- manual upstream commits and manual refs do not get upstream sync tags
- do not create Go module-style `vX.Y.Z` tags for upstream sync markers
- do not move existing upstream sync tags
- if the same upstream tag needs a follow-up codexsdk-go fix, use `python3 scripts/codexsdk_sync_tag.py --next-suffix --create --push origin` to create `-sync.N`

## Decision Rules

- If drift is clean and the user only asked to check a tag/ref/commit, report that no SDK update is needed.
- If drift is clean but the user explicitly asked to move baseline provenance, update provenance and clean drift artifacts without changing schema-derived Go output.
- If the target policy gate returns `block`, stop before generating drift.
- If generated Go fails because a new schema shape is unsupported, update `codexsdk/internal/protocolgen` with a reviewed generation rule and focused tests before regenerating.
- If a method disappears upstream, preserve compatibility only when it is safe and intentional; otherwise document the breaking change.
- If upstream adds experimental surface, mark it experimental unless it appears in the non-experimental schema comparison and existing manifest rules say otherwise.
- If compare-only and full tracking disagree, trust full tracking output and investigate candidate provenance before editing checked-in files.
- If caller-owned automation closes drift issues by label, verify the existing issue has the required drift label before running that automation.
- If a scheduled or manual drift issue already exists, update or close that issue; do not create a new issue for each upstream target while the old one is unresolved.
