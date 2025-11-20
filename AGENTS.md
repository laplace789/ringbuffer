# AGENTS.md

## Project overview

This repository contains `github.com/laplace789/ringbuffer`, a bounded,
generic, lock-free ring buffer for exactly one producer and one consumer
(SPSC). The supported release scope is Go 1.25.8 on 64-bit `amd64` and
`arm64`. Other architectures are compile-only, best effort.

Correctness, memory ordering, zero-allocation hot paths, and the two-phase slot
ownership contract are core product requirements. Do not treat this as a
general-purpose MPSC, SPMC, or MPMC queue.

## Read this before editing

Get explicit user approval before changing any of the following:

- the public API or exported errors;
- SPSC ownership or memory-ordering behavior;
- the `Ring` field layout, padding, or `CacheLineSize`;
- the supported Go version or target architectures;
- `LICENSE`, `NOTICE`, or source attribution.

Internal refactors, tests, examples, and documentation that preserve these
contracts may proceed normally.

## Core invariants

### Concurrency model

- Exactly one goroutine owns `Set` and `SetAdv` calls.
- Exactly one goroutine owns `Get` and `GetAdv` calls. The producer and
  consumer roles may run sequentially in one goroutine or concurrently in two,
  but neither role may have multiple owners.
- `Reset` is not concurrency-safe. Both sides must be stopped before calling it.
- A `Ring` must not be copied after first use. `noCopy` exists so `go vet
  -copylocks` detects common value-copy mistakes. Pass `*Ring[T]` only.
- There is intentionally no public `Len` method. Full and empty observations
  come from the result of the attempted `Set` or `Get` operation.

### Slot ownership

- A successful `Set` gives the producer exclusive access to the returned slot
  pointer until the matching `SetAdv` succeeds.
- After `SetAdv`, the producer must not retain, read, or write that pointer.
- A successful `Get` gives the consumer exclusive access to the returned slot
  pointer until the matching `GetAdv` succeeds.
- After `GetAdv`, the consumer must not retain or dereference that pointer.
- A second reservation on the same side is an error. An advance without a
  matching successful reservation is an error and must not mutate ring state.

### Publication and reclamation

- The producer writes slot data before atomically advancing `wp` in `SetAdv`.
- The consumer observes `wp` atomically before reading committed slot data.
- The consumer clears the slot before atomically advancing `rp` in `GetAdv`.
- The producer observes `rp` atomically before reusing reclaimed storage.
- `shadowRp` is producer-owned and `shadowWp` is consumer-owned. Reservation
  flags are also owned only by their respective side. Do not make them shared
  casually or weaken the atomic publication/reclamation edges.

### Layout and performance

- Producer and consumer state are separated by at least 128 bytes.
- The current padding calculation includes two `uint64` fields and one `bool`
  per side. Reordering or adding fields requires recalculating padding and
  updating the white-box layout test.
- The 128-byte separation is a supported-platform optimization, not a universal
  guarantee about every CPU cache topology.
- Normal `Set`/`SetAdv`/`Get`/`GetAdv` operation must remain `0 allocs/op`.

## Public API

- `New[T](capacity) (*Ring[T], error)` rounds a positive capacity up to a power
  of two and rejects invalid, overflowing, or impossible allocation sizes.
- `Set() (idx uint64, ptr *T, err error)` reserves producer storage.
- `SetAdv() error` publishes a producer reservation.
- `Get() (idx uint64, ptr *T, err error)` reserves committed consumer data.
- `GetAdv() error` clears and releases a consumer reservation.
- `Reset()` clears slots, counters, shadow indices, and reservations while idle.
- `Capacity() uint64` returns the fixed rounded capacity.

All public sentinel errors live in `errors.go`. Preserve `errors.Is` behavior for
wrapped `ErrInvalidCapacity` results.

## Repository map

- `ringbuffer.go`: core data structure, constructor, and SPSC operations.
- `errors.go`: all exported sentinel errors.
- `no_copy.go`: zero-runtime-cost `go vet` copy protection.
- `doc.go`: package-level concurrency and ownership contract.
- `ringbuffer_test.go`: white-box correctness, race stress, allocation tests,
  layout checks, and benchmarks.
- `ringbuffer_fuzz_test.go`: model-based sequential state-machine fuzzing.
- `example_test.go`: external-package, executable public API examples.
- `README.md`: user-facing installation, usage, scope, and attribution.
- `LICENSE` and `NOTICE`: mandatory licensing and derivative-work attribution.
- `.github/workflows/ci.yml`: release gates and benchmark artifact retention.
- `PRODUCTION_READINESS.md`: local ignored historical checklist; do not depend on
  it as a versioned project contract.

## Editing guidance

- Keep production code free of test-only exported helpers. White-box layout
  checks may use `unsafe` in `_test.go` files; core code should not need it.
- Keep examples in external package `ringbuffer_test` so they verify the real
  import path and public surface.
- Keep model-based fuzzing in package `ringbuffer` when it needs internal
  invariant access.
- Use `gofmt` on every changed Go file.
- Preserve unrelated user changes in a dirty worktree.
- Update `README.md`, `doc.go`, examples, tests, and workflow commands together
  when an approved contract change affects them.

## Required validation

For normal code changes, run:

```shell
go test -short ./...
go test ./...
go test -race ./...
go vet ./...
```

For state-machine or reservation changes, also run:

```shell
go test -run '^$' -fuzz '^FuzzRingStateMachine$' -fuzztime=10s .
```

For release or platform-sensitive changes, compile supported targets:

```shell
GOOS=linux GOARCH=amd64 go test -c -o /tmp/ringbuffer-amd64.test .
GOOS=linux GOARCH=arm64 go test -c -o /tmp/ringbuffer-arm64.test .
```

If the environment cannot write the default Go build cache, prefix commands
with a task-specific cache such as `GOCACHE=/tmp/ringbuffer-gocache`.

### Hot-path changes

Any change to atomics, padding, shadow indices, slot clearing, reservation
state, or core operations requires before/after benchmark results using the
same machine and command:

```shell
go test -run '^$' -bench '^BenchmarkSPSCConcurrent$' -benchtime=200ms -benchmem .
```

Report `ns/op`, `B/op`, and `allocs/op`. Machine-dependent timing is not a hard
pass/fail threshold, but Ring hot paths must remain `0 allocs/op`.

## Common failure modes

- Calling `SetAdv` or `GetAdv` without checking that reservation succeeded.
- Accessing a slot pointer after its matching advance.
- Adding another producer or consumer because individual fields are atomic.
- Copying a `Ring` value and accidentally sharing its backing slice with
  independent counters.
- Adding fields without preserving 128-byte producer/consumer separation.
- Using check-then-act logic instead of handling `ErrRingFull` or
  `ErrRingEmpty` from the operation itself.
- Removing slot zeroing in `GetAdv`, which retains references and can increase
  GC pressure.
- Adding allocations, logging, callbacks, locks, or blocking behavior to the
  hot path without explicit approval and benchmark evidence.

## Licensing and attribution

This implementation is derived from Terry.Mao's goim ring buffer. The MIT
license requires preserving the original 2015 Terry.Mao notice. This project
also carries the 2026 laplace789 notice and a `NOTICE` describing the major
modifications. Never remove or rewrite this attribution without explicit user
approval and a licensing review.
