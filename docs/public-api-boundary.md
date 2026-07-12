# Pre-v1 Public API Boundary

The accepted target for issue #44 is recorded in
[ADR 0001](adr/0001-pre-v1-public-api-boundary.md). This page translates the
decision into consumer guidance and migration consequences. It describes a
future pre-v1 change; the current release still returns the existing
`codexsdk.Client` interface.

## Target shape

The root client will be a concrete `*codexsdk.Client` created only by
`codexsdk.New`. It will continue to own `Close`, exact thread/turn lifecycle,
typed callbacks, and access to all generated facades. Generated additions then
add methods to a concrete type instead of enlarging an interface every SDK
consumer may have implemented.

The SDK will not publish a replacement umbrella interface. Declare interfaces
at the consuming seam instead:

```go
type RunStarter interface {
	Start(context.Context, codexsdk.StartThreadRunRequest) (codexsdk.StartedThreadRun, error)
}

func startRun(ctx context.Context, runner RunStarter, req codexsdk.StartThreadRunRequest) error {
	_, err := runner.Start(ctx, req)
	return err
}
```

A facade consumer follows the same rule:

```go
type ModelLister interface {
	List(context.Context, protocolv2.ModelListParams) (protocolv2.ModelListResponse, error)
}

func listModels(ctx context.Context, models ModelLister) error {
	_, err := models.List(ctx, protocolv2.ModelListParams{})
	return err
}
```

The repository compile-checks these patterns without requiring a consumer to
mock `SDKSurface`.

## Construction and ownership

`New` requires `ClientOptions` and starts and initializes one app-server
connection. On success, the caller owns the returned root and must call
`Close`. The zero value is deliberately inert, not an alternate construction
path: it does not launch a process, `Close` is safe, and operations return
`ErrClientClosed`.

No `NewClient`, default global, lazy connection, or constructor wrapper is part
of the target. Applications may wrap construction when they own additional
policy, but that policy does not belong to this SDK.

## Generated stable and experimental surface

`protocolv2` and the exact facades remain complete and colocated. The checked-in
classified manifest owns the stable/experimental fact; documentation and
release reports consume it instead of maintaining handwritten lists.

- Before v1, classification guides migration, promotion, documentation, and
  support expectations.
- At v1, every exported generated name in this module receives normal source-
  compatibility protection, including names classified experimental.
- Generated drift that would remove or incompatibly change an exported name
  requires an honest additive compatibility path or the next major version.
- Runtime `ExperimentalAPI` capability opt-in does not alter compile-time
  availability or compatibility classification.
- A separately versioned experimental module is the only clean boundary for
  post-v1 minor-release source breakage, and is rejected until concrete demand
  justifies its additional complexity.

The current manifest classifies methods. A later bounded issue must extend the
same source of truth across generated types and members and add enforcement;
this design PR does not modify generated protocol metadata or code. Issue #45
is closed because its design was folded into #44; new implementation work must
use new bounded issues rather than reopening or repurposing #45.

## Migration window

The concrete-root change is scheduled for a bounded release no earlier than
v0.3 and before v1. It will be delivered atomically with migration,
compatibility inventory, changelog, package documentation, and clean-consumer
evidence.

Typical construction using type inference is unchanged in shape:

```go
root, err := codexsdk.New(options)
```

Explicit storage changes from `codexsdk.Client` to `*codexsdk.Client`. Tests
and adapters should replace broad SDK-root mocks with the narrow interfaces
their code actually consumes. There will be no parallel SDK-owned interface or
alternate constructor transition period.
