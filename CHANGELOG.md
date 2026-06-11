# Changelog

This project follows Semantic Versioning once it reaches v1.0. Before v1.0,
minor versions may contain breaking changes when required by upstream protocol
drift or SDK API corrections.

## [Unreleased]

No changes yet.

## [0.1.0] - 2026-06-11

### Added

- Initial public release.
- Root SDK client, typed app-server protocol v2 package, generated protocol
  surface, tests, and schema baseline.
- Upstream tracking script for generating Codex app-server schema drift review
  artifacts without modifying the checked-in baseline.
- Open-source project documentation covering status, usage, real app-server
  smoke tests, schema provenance, API compatibility, maintenance, security,
  support, release, code of conduct, issue templates, pull request template,
  and upstream/dependency notices.
- GitHub Actions CI for gofmt, vet, tests, and generated protocol code
  reproducibility.
- Dependabot configuration for Go modules and GitHub Actions.

### Changed

- Schema baseline provenance uses public upstream URLs and repo-relative source
  references rather than local machine paths.
