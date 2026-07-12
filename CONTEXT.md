# Codex SDK Public Boundary

This context names the compatibility boundaries between the SDK, the generated
Codex protocol, and applications that consume them.

## Language

**Root Client**:
The application-owned connection to one locally launched Codex app-server,
including its lifecycle and access to exact protocol operations.
_Avoid_: SDK interface, service container

**Lifecycle API**:
A handwritten operation that owns app-server process or thread/turn lifecycle
while preserving exact generated protocol facts.
_Avoid_: workflow API, convenience DSL

**Generated Facade**:
A protocol-derived group of exact Codex app-server operations exposed by the
Root Client.
_Avoid_: service abstraction, provider API

**Stable Generated Surface**:
Generated declarations and members reachable from the non-experimental Codex
schema and documented as supported without experimental runtime opt-in.
_Avoid_: frozen protocol

**Experimental Generated Surface**:
Generated declarations or members present only in the experimental Codex
schema and documented as requiring experimental runtime opt-in or carrying
weaker support expectations.
_Avoid_: private API, runtime-enabled surface

**Consumer-Owned Interface**:
A narrow interface declared by an application at the point where it consumes a
Lifecycle API or Generated Facade.
_Avoid_: SDK umbrella interface

**Exact Run**:
One composed thread/turn execution together with its ordered attributable
protocol evidence, partial result, and stable terminal cause.
_Avoid_: workflow, request

**Exact Run Waiter**:
An independent, non-destructive observer of an Exact Run's completion and
immutable result snapshot.
_Avoid_: subscriber, cursor, stream consumer

**Shared Run Cancellation**:
An explicit lifecycle boundary that terminates an Exact Run for every observer.
_Avoid_: waiter cancellation, timeout
