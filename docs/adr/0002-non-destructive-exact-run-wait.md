# Observe exact-run completion without a cursor

Exact streams expose a non-destructive `Wait` operation whose callers wait
independently for the same terminal run and receive isolated result snapshots.
A waiter's context only bounds that caller's wait; `Stream.Close`, client
shutdown or failure, and protocol terminal completion remain the shared run
boundaries. This reuses the complete ordered evidence already retained in the
result instead of introducing cursors, subscriber registries, or a second
history model, while the existing `Next` path keeps its bounded live-delivery
and compatibility semantics.
