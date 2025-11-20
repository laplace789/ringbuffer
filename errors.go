package ringbuffer

import "errors"

var (
	// ErrInvalidCapacity indicates that a requested capacity is non-positive,
	// cannot be represented after rounding, or cannot be allocated.
	ErrInvalidCapacity = errors.New("invalid ring buffer capacity")
	// ErrRingEmpty indicates that Get found no committed value.
	ErrRingEmpty = errors.New("ring buffer empty")
	// ErrRingFull indicates that Set found no writable slot.
	ErrRingFull = errors.New("ring buffer full")
	// ErrWriteReservationActive indicates that Set was called twice without SetAdv.
	ErrWriteReservationActive = errors.New("ring buffer write reservation already active")
	// ErrReadReservationActive indicates that Get was called twice without GetAdv.
	ErrReadReservationActive = errors.New("ring buffer read reservation already active")
	// ErrNoWriteReservation indicates that SetAdv has no matching successful Set.
	ErrNoWriteReservation = errors.New("ring buffer has no active write reservation")
	// ErrNoReadReservation indicates that GetAdv has no matching successful Get.
	ErrNoReadReservation = errors.New("ring buffer has no active read reservation")
)
