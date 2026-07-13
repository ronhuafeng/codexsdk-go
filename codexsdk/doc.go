// Package codexsdk provides a concrete Client for interacting with a Codex
// app-server. New returns the only connected form, *Client. The safe, inert
// zero value closes successfully and returns ErrClientClosed from operations.
// Consumers should define narrow interfaces around the operations they use.
//
// Client owns process transport, client lifecycle, generated protocol
// facades, exact thread/turn composition, and typed callback delivery. Exact
// run notifications retain ingestion order across stream attachment: pending
// notifications are accepted before later live notifications for the same run.
// Exact run results retain complete immutable notification history independent
// of observation. Wait observes completion without consuming notifications;
// Next advances a cursor over the same ordered history. The configurable
// global notification-handler queue remains bounded, and its overflow closes
// the client with ErrNotificationBackpressure.
// Exact run history follows generated-schema identity: turn-scoped facts attach
// only to the matching turn; thread-scoped facts attach to every run currently
// active or attaching for that thread and are not retained for a later run;
// client/global facts never enter per-run evidence. Every validated generated
// notification is still enqueued for the global handler, in ingestion order,
// after its justified per-run append completes. A terminal exact notification
// cannot complete its affected stream until that notification's global handler
// invocation has returned and any handler failure is published as the client
// first cause.
// Client shutdown atomically closes callback admission and joins callbacks
// accepted before that boundary before releasing transport resources.
//
// It does not provide provider-neutral LLM abstractions, business validation,
// workflow policy, or application safety profiles.
package codexsdk
