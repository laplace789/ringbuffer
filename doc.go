// Package ringbuffer provides a bounded, lock-free, generic ring buffer for a
// single producer and a single consumer (SPSC).
//
// Exactly one producer goroutine may use Set and SetAdv, while exactly one
// consumer goroutine may use Get and GetAdv. A successful Set gives the producer
// exclusive ownership of the returned slot pointer until SetAdv succeeds. A
// successful Get gives the consumer exclusive ownership until GetAdv succeeds.
// A slot pointer must not be retained or accessed after its matching advance.
//
// A Ring must not be copied after first use. Reset is not concurrency-safe and
// may only be called after producer and consumer activity has stopped. The
// package deliberately provides no concurrent length snapshot; callers should
// use ErrRingFull and ErrRingEmpty as the results of individual operations.
package ringbuffer
