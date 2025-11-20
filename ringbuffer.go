package ringbuffer

import (
	"fmt"
	"sync/atomic"
)

// CacheLineSize is the conservative cache-line separation used on supported targets.
const CacheLineSize = 128

// Ring is a single-producer, single-consumer ring buffer.
//
// A Ring must not be copied after first use. Exactly one producer goroutine may call
// Set and SetAdv, and exactly one consumer goroutine may call Get and GetAdv. Reset
// must only be called while both sides are stopped.
type Ring[T any] struct {
	// --- Producer Cache Line ---
	wp            uint64
	shadowRp      uint64                   // Producer 本地緩存的消費者位置
	writeReserved bool                     // Producer 是否持有可寫 slot
	_p1           [CacheLineSize - 17]byte // Padding 到 CacheLineSize

	// --- Consumer Cache Line ---
	rp           uint64
	shadowWp     uint64                   // Consumer 本地緩存的生產者位置
	readReserved bool                     // Consumer 是否持有可讀 slot
	_p2          [CacheLineSize - 17]byte // Padding 到 CacheLineSize

	// --- Shared Read-Only Data ---
	noCopy noCopy
	num    uint64
	mask   uint64
	data   []T
}

func roundUpPowerOfTwo(n uint64) uint64 {
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	n++
	return n
}

// New creates a Ring whose capacity is the smallest power of two greater than
// or equal to num. It returns ErrInvalidCapacity for invalid or unrepresentable
// capacities and for allocation-size panics.
func New[T any](num int) (*Ring[T], error) {
	if num <= 0 {
		return nil, fmt.Errorf("%w: got %d, must be greater than zero", ErrInvalidCapacity, num)
	}

	capacity := roundUpPowerOfTwo(uint64(num))
	maxInt := uint64(^uint(0) >> 1)
	if capacity == 0 || capacity > maxInt {
		return nil, fmt.Errorf("%w: %d rounds beyond the maximum int", ErrInvalidCapacity, num)
	}

	data, err := allocate[T](int(capacity))
	if err != nil {
		return nil, err
	}
	return &Ring[T]{
		data: data,
		num:  capacity,
		mask: capacity - 1,
	}, nil
}

func allocate[T any](capacity int) (data []T, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			data = nil
			err = fmt.Errorf("%w: cannot allocate capacity %d: %v", ErrInvalidCapacity, capacity, recovered)
		}
	}()
	return make([]T, capacity), nil
}

// Set reserves a writable slot for the producer.
//
// From a successful Set until the matching SetAdv, the returned pointer is owned
// exclusively by the producer. The producer must not retain, read, or write the
// pointer after SetAdv. Calling Set again before SetAdv returns
// ErrWriteReservationActive.
func (r *Ring[T]) Set() (idx uint64, ptr *T, err error) {
	if r.writeReserved {
		return 0, nil, ErrWriteReservationActive
	}

	// 1. 讀取自己的 wp (不需要 atomic，因為只有我在寫)
	// 但為了配合 atomic.AddUint64 的記憶體模型一致性，使用 Load 也可以，差異極微。
	wp := atomic.LoadUint64(&r.wp)

	// 2. 檢查 shadow
	if wp-r.shadowRp < r.num {
		idx = wp & r.mask
		r.writeReserved = true
		return idx, &r.data[idx], nil
	}

	// 3. 讀取真正的 rp
	rp := atomic.LoadUint64(&r.rp)
	r.shadowRp = rp // Update Shadow

	// 4. 再次檢查
	if wp-rp >= r.num {
		return 0, nil, ErrRingFull
	}
	idx = wp & r.mask
	r.writeReserved = true
	return idx, &r.data[idx], nil
}

// SetAdv publishes the slot reserved by Set to the consumer.
// After it returns successfully, the producer must no longer access the slot pointer.
func (r *Ring[T]) SetAdv() error {
	if !r.writeReserved {
		return ErrNoWriteReservation
	}
	r.writeReserved = false
	atomic.AddUint64(&r.wp, 1)
	return nil
}

// Get reserves a readable slot for the consumer.
//
// From a successful Get until the matching GetAdv, the returned pointer is owned
// exclusively by the consumer. The consumer must not retain or dereference the
// pointer after GetAdv. Calling Get again before GetAdv returns
// ErrReadReservationActive.
func (r *Ring[T]) Get() (idx uint64, ptr *T, err error) {
	if r.readReserved {
		return 0, nil, ErrReadReservationActive
	}

	// 1. 讀取自己的 rp
	rp := atomic.LoadUint64(&r.rp)

	// 2. 檢查 shadow
	if r.shadowWp > rp {
		idx = rp & r.mask
		r.readReserved = true
		return idx, &r.data[idx], nil
	}

	// 3. 讀取真正的 wp
	wp := atomic.LoadUint64(&r.wp)
	r.shadowWp = wp // [FIX] 更新 Shadow，這是原本漏掉的關鍵

	if rp == wp {
		return 0, nil, ErrRingEmpty
	}
	idx = rp & r.mask
	r.readReserved = true
	return idx, &r.data[idx], nil
}

// GetAdv releases the slot reserved by Get back to the producer.
// After it returns successfully, the consumer must no longer access the slot pointer.
func (r *Ring[T]) GetAdv() error {
	if !r.readReserved {
		return ErrNoReadReservation
	}
	rp := atomic.LoadUint64(&r.rp)

	//清空slot 避免leak
	var zero T
	r.data[rp&r.mask] = zero

	r.readReserved = false
	atomic.AddUint64(&r.rp, 1)
	return nil
}

// Reset clears all slots, counters, and reservations.
// It is not concurrency-safe and may only be called while both sides are stopped.
func (r *Ring[T]) Reset() {
	atomic.StoreUint64(&r.rp, 0)
	atomic.StoreUint64(&r.wp, 0)
	r.shadowRp = 0
	r.shadowWp = 0
	r.writeReserved = false
	r.readReserved = false
	// Optional: Clear all data to fix leaks on reset
	var zero T
	for i := range r.data {
		r.data[i] = zero
	}
}

// Capacity returns the rounded, fixed capacity of the Ring.
func (r *Ring[T]) Capacity() uint64 {
	return r.num
}
