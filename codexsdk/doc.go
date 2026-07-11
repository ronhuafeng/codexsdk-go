// Package codexsdk provides Go interfaces for interacting with a Codex
// app-server. It owns process transport, client lifecycle, generated protocol
// facades, exact thread/turn composition, and typed callback delivery. Exact
// run notifications retain ingestion order across stream attachment: pending
// notifications are accepted before later live notifications for the same run.
// Each exact run stream has a bounded delivery queue. If that queue overflows,
// the generated notification remains in the run's partial history, the first
// backpressure failure closes the client and all active streams, and Close
// returns that same cause. This per-run capacity is independent of the
// configurable global notification-handler queue capacity. Per-run overflow
// errors include turn_id context; global-handler overflow has no run context.
// Exact run history follows generated-schema identity: turn-scoped facts attach
// only to the matching turn; thread-scoped facts attach to every run currently
// active or attaching for that thread and are not retained for a later run;
// client/global facts never enter per-run evidence. Every validated generated
// notification is still enqueued for the global handler, in ingestion order,
// after its justified per-run append completes.
// Client shutdown atomically closes callback admission and joins callbacks
// accepted before that boundary before releasing transport resources.
//
// It does not provide provider-neutral LLM abstractions, business validation,
// workflow policy, or application safety profiles.
package codexsdk
