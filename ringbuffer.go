package ringbuffer

import (
	"errors"
	"sync/atomic"
	"unsafe"
)

// CacheLineSize 預設 64 byte
const CacheLineSize = 64

type Ring[T any] struct {
	// --- Producer Cache Line ---
	wp       uint64
	shadowRp uint64                   // Producer 本地緩存的消費者位置
	_p1      [CacheLineSize - 16]byte // Padding 到 64 bytes

	// --- Consumer Cache Line ---
	rp       uint64
	shadowWp uint64                   // Consumer 本地緩存的生產者位置
	_p2      [CacheLineSize - 16]byte // Padding 到 64 bytes

	// --- Shared Read-Only Data ---
	num  uint64
	mask uint64
	data []T
}

var (
	ErrRingEmpty = errors.New("ring buffer empty")
	ErrRingFull  = errors.New("ring buffer full")
)

func roundUpPowerOfTwo(n uint64) uint64 {
	if n == 0 {
		return 1
	}
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

func New[T any](num int) *Ring[T] {
	r := new(Ring[T])
	r.init(uint64(num))
	return r
}

func (r *Ring[T]) init(num uint64) {
	num = roundUpPowerOfTwo(num)
	r.data = make([]T, num)
	r.num = num
	r.mask = num - 1
}

// Set 提供可寫 slot。注意：寫入 data 後必須呼叫 SetAdv。
func (r *Ring[T]) Set() (idx uint64, ptr *T, err error) {
	// 1. 讀取自己的 wp (不需要 atomic，因為只有我在寫)
	// 但為了配合 atomic.AddUint64 的記憶體模型一致性，使用 Load 也可以，差異極微。
	wp := atomic.LoadUint64(&r.wp)

	// 2. 檢查 shadow
	if wp-r.shadowRp < r.num {
		idx = wp & r.mask
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
	return idx, &r.data[idx], nil
}

func (r *Ring[T]) SetAdv() {
	atomic.AddUint64(&r.wp, 1)
}

// Get 提供可讀 slot。注意：讀取後必須呼叫 GetAdv。
func (r *Ring[T]) Get() (idx uint64, ptr *T, err error) {
	// 1. 讀取自己的 rp
	rp := atomic.LoadUint64(&r.rp)

	// 2. 檢查 shadow
	if r.shadowWp > rp {
		idx = rp & r.mask
		return idx, &r.data[idx], nil
	}

	// 3. 讀取真正的 wp
	wp := atomic.LoadUint64(&r.wp)
	r.shadowWp = wp // [FIX] 更新 Shadow，這是原本漏掉的關鍵

	if rp == wp {
		return 0, nil, ErrRingEmpty
	}
	idx = rp & r.mask
	return idx, &r.data[idx], nil
}

// GetAdv 推進讀取指標
func (r *Ring[T]) GetAdv() {
	rp := atomic.LoadUint64(&r.rp)

	//清空slot 避免leak
	var zero T
	r.data[rp&r.mask] = zero

	atomic.AddUint64(&r.rp, 1)
}

// Reset 非線程安全，請確保無讀寫時呼叫
func (r *Ring[T]) Reset() {
	atomic.StoreUint64(&r.rp, 0)
	atomic.StoreUint64(&r.wp, 0)
	r.shadowRp = 0
	r.shadowWp = 0
	// Optional: Clear all data to fix leaks on reset
	var zero T
	for i := range r.data {
		r.data[i] = zero
	}
}

// Capacity 返回容量
func (r *Ring[T]) Capacity() uint64 {
	return r.num
}

// Len 返回目前數量
func (r *Ring[T]) Len() uint64 {
	wp := atomic.LoadUint64(&r.wp)
	rp := atomic.LoadUint64(&r.rp)
	return wp - rp
}

// PaddingBytes 用於測試對齊
func (r *Ring[T]) PaddingBytes() uintptr {
	wpAddr := uintptr(unsafe.Pointer(&r.wp))
	rpAddr := uintptr(unsafe.Pointer(&r.rp))
	if rpAddr > wpAddr {
		return rpAddr - wpAddr
	}
	return wpAddr - rpAddr
}
