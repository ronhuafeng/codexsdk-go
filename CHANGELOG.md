# Changelog

All notable changes to this project are documented here.

## [Unreleased]

No changes yet.

## [0.3.0] - 2026-07-13

### Fixed

- Notifications accepted while an exact stream attaches are now published in
  acceptance order, so pending replay cannot be overtaken by live delivery.

### Changed

- Replaced the SDK-owned `Client`/`SDKSurface` umbrella interfaces with an
  exported concrete `Client`; `New(ClientOptions)` now returns `*Client` while
  retaining lifecycle methods, `ThreadRunner`, typed callbacks, partial
  evidence, and every exact generated facade.
- The zero-value `Client` is safe and inert: `Close` succeeds and operations
  return `ErrClientClosed`. Consumers should define narrow interfaces at their
  own test and adapter seams.
- Removed the deprecated v0.1 `ThreadClient`, projected streams, copied
  protocol models and enums, conversion helpers, compatibility initialization
  fields, and legacy server-request callback path. Use `ThreadRunner`, exact
  `protocolv2` values, generated facades, and `ServerRequestHandler`.

### Added

- Non-destructive exact `Stream.Wait` observation for independent concurrent
  consumers, with immutable complete or partial result snapshots and
  caller-local context cancellation.

### Fixed

- Transport failure during exact-stream attachment now retains notifications
  and run evidence accepted before terminalization, preserving the first cause
  together with the immutable partial result.

## [0.2.1] - 2026-07-12

### Fixed

- Exact-run server requests now stay on the generated handler path. A missing
  handler returns a request-kind-aware fail-closed response when safe, while
  requests that require application data fail with a typed cause and preserve
  partial run evidence.
- Notifications accepted before exact-stream attachment remain ordered before
  live notifications, including terminal, usage, reroute, and diagnostic
  evidence.
- Per-run notification backpressure now has timing-independent, client-wide
  first-failure semantics for both pending replay and live delivery.
- Handler admission and shutdown now share one atomic boundary, so accepted
  callbacks are cancelled or joined before transport teardown and late
  callbacks cannot start.
- Notification evidence accepted by an exact run is committed before a failing
  notification handler terminates the run, preserving the handler cause and
  partial result together.
- Generated notification kinds now use a schema-derived attribution policy;
  global facts are not copied into unrelated run histories, and turn- and
  thread-scoped facts reach only their documented targets.
- Exact-run turn attribution is synchronized across publication, routing, and
  lifecycle cleanup.

### Compatibility

- The deprecated v0.1 `ThreadClient` surface remains available and covered by
  compatibility tests. The generated protocol baseline is synchronized to the
  stable `rust-v0.144.1` schema; this patch adds no public convenience APIs.

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
