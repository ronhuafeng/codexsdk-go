# Changelog

All notable changes to this project are documented here.

## [Unreleased]

No changes yet.

## [0.2.0] - 2026-07-11

### Added

- Exact generated-param `ThreadRunner` start/resume composition and generic
  notification streams with immutable partial result snapshots.
- Full generated notification callbacks on a bounded serial dispatcher with
  explicit normal-close and failure-shutdown behavior.
- Generated `ServerRequest` callbacks with opaque typed response constructors
  covering every generated request kind.
- Typed `ProtocolError` and `TurnError` values that preserve exact Codex facts.
- Sanitized diagnostic references for malformed run notifications and a
  canonical handwritten API allowlist.

### Changed

- `ClientOptions.Initialize` accepts exact generated initialization params.
- Transport, protocol, handler, panic, and backpressure failures terminate the
  client with one preserved first cause and retain run evidence accepted first.
- `TextAndFiles` no longer silently discards blank paths; downstream input
  validation rejects them explicitly.

### Deprecated

- The v0.1 `ThreadClient`, copied request/result/event models, copied enums,
  conversion helpers, and projected server-request callback. Use generated
  `protocolv2` facts, `ThreadRunner`, and `ServerRequestHandler`.
