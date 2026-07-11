# Release and Baseline Checklist

Use this checklist before tagging or publishing a release.

## Versioning

- Pre-v1.0: minor versions may contain breaking changes. Patch versions should
  remain compatible unless a security fix requires otherwise.
- v1.0 and later: follow SemVer for the public API in `codexsdk` and
  `codexsdk/protocolv2`.
- Generated additions to `protocolv2` are usually minor changes.
- Removing or changing public generated types, method constants, facade
  methods, request structs, response structs, or thread helper behavior is a
  major change after v1.0 unless preserving compatibility would be unsafe.

The handwritten public API is mechanically recorded in
`codexsdk/testdata/handwritten-api.txt`. Generated `protocolv2` declarations and
generated facades are intentionally excluded because their inventory is a
protocol fact guarded by generator reproducibility tests.

## Baseline Update Flow

1. Pick an upstream OpenAI Codex commit.
2. Generate review artifacts:

   ```sh
   scripts/codexsdk_track_upstream.sh \
     --codex-repo /path/to/openai/codex \
     --commit <codex-commit> \
     --out /tmp/codexsdk-upstream
   ```

3. Review `/tmp/codexsdk-upstream/reports/SUMMARY.md`.
4. Compare the generated schema with the checked-in baseline.
5. Update the schema files only after reviewing added, removed, and changed
   protocol surface.
6. Update `baseline_metadata.json` with public provenance: upstream URL, full
   commit SHA, source license, Codex version, generation command, schema file
   count, and schema bundle checksum.
7. Update `manifest.json`, `coverage_matrix.json`, `drift_report.json`, and
   `matrix_update_skeleton.json`.
8. Regenerate Go code:

   ```sh
   GOWORK=off go run ./codexsdk/internal/cmd/protocolv2gen
   ```

9. Update README, CHANGELOG, NOTICE, and this checklist if the upstream source,
   license, compatibility story, generated-code process, or smoke-test process
   changed.

## Release Checks

The following GitHub Actions jobs are release blockers for changes targeting a
release:

- `Go minimum` verifies the standalone module with the minimum Go version
  declared by `go.mod`.
- `Go` uses the current stable Go release for module metadata, formatting, vet,
  the full test suite, script tests, and generator reproducibility.
- `Go race` runs the full suite with the race detector.
- `Runtime repeat` repeats only synchronization-driven runtime concurrency
  contracts; it supplements rather than replaces their deterministic runs.
- `Tag clean module` runs only after a `v*` tag is pushed. It resolves that tag
  from `proxy.golang.org` in a new consumer module and compiles an import of the
  SDK. A failed tag smoke test blocks publishing or announcing that release.

All Go CI jobs run with `GOWORK=off`. The `Go`, `Go minimum`, `Go race`, and
`Runtime repeat` jobs must pass on the exact pull-request head. The tag-only
`Tag clean module` job must pass on the exact release tag. None of these gates
changes the separately enforced protocol-baseline finalization policy.

```sh
git ls-files -z -- '*.go' ':!:vendor/**' | xargs -0 gofmt -w
GOWORK=off go vet ./...
GOWORK=off go test ./...

GOWORK=off go run ./codexsdk/internal/cmd/protocolv2gen -stdout method-registry |
  diff -u codexsdk/protocolv2/method_registry.gen.go -
GOWORK=off go run ./codexsdk/internal/cmd/protocolv2gen -stdout protocol-types |
  diff -u codexsdk/protocolv2/protocol_types.gen.go -
```

Check for public-readiness leaks:

```sh
rg -n '(/Users/|/opt/homebrew|SECRET|TOKEN|PASSWORD|BEGIN (RSA|OPENSSH|EC|PRIVATE) KEY|sk-[A-Za-z0-9]{20,})' .
```

Review every match before release.

## Tagging

1. Confirm `git status --short` shows only intended release changes.
2. Confirm `CHANGELOG.md` has the release date and highlights compatibility
   impact.
3. Create an annotated tag:

   ```sh
   git tag -a vX.Y.Z -m "vX.Y.Z"
   ```

4. Push the tag from a clean worktree after CI passes.

## Repository Settings

Before making the repository public, enable GitHub branch protection, Dependabot
alerts/security updates, private vulnerability reporting, and secret scanning.
Add CodeQL or OpenSSF Scorecard workflows once those features are available for
the repository visibility and organization plan.
