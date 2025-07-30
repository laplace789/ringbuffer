package ringbuffer

import (
	"errors"
	"sync/atomic"
	"unsafe"
)

// 預設 cache line size，若要跨平台用 build tag 改這裡即可
const CacheLineSize = 64

// 自動 padding 結構
type cacheLinePad struct {
	_ [CacheLineSize]byte
}

// SPSC（單生產單消費）高效環形緩衝區
type Ring[T any] struct {
	wp    uint64       // 生產者指標
	_pad0 cacheLinePad // padding 防止 false sharing
	rp    uint64       // 消費者指標
	_pad1 cacheLinePad // padding

	num  uint64 // 必為2的次方
	mask uint64
	data []T
}

// 常見錯誤
var (
	ErrRingEmpty = errors.New("ring buffer empty")
	ErrRingFull  = errors.New("ring buffer full")
)

// 取最近的2的冪
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

// 建構子
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

// 提供可寫slot指標（Set），寫完請呼叫 SetAdv
func (r *Ring[T]) Set() (idx uint64, ptr *T, err error) {
	rp := atomic.LoadUint64(&r.rp)
	wp := atomic.LoadUint64(&r.wp)
	if wp-rp >= r.num {
		return 0, nil, ErrRingFull
	}
	idx = wp & r.mask
	return idx, &r.data[idx], nil
}

// 寫完前進
func (r *Ring[T]) SetAdv() {
	atomic.AddUint64(&r.wp, 1)
}

// 提供可讀slot指標（Get），讀完請呼叫 GetAdv
func (r *Ring[T]) Get() (idx uint64, ptr *T, err error) {
	wp := atomic.LoadUint64(&r.wp)
	rp := atomic.LoadUint64(&r.rp)
	if rp == wp {
		return 0, nil, ErrRingEmpty
	}
	idx = rp & r.mask
	return idx, &r.data[idx], nil
}

// 讀完前進
func (r *Ring[T]) GetAdv() {
	atomic.AddUint64(&r.rp, 1)
}

// Reset: 重置指標
func (r *Ring[T]) Reset() {
	atomic.StoreUint64(&r.rp, 0)
	atomic.StoreUint64(&r.wp, 0)
}

// Capacity: 返回容量
func (r *Ring[T]) Capacity() uint64 {
	return r.num
}

// Len: 返回目前數量
func (r *Ring[T]) Len() uint64 {
	wp := atomic.LoadUint64(&r.wp)
	rp := atomic.LoadUint64(&r.rp)
	return wp - rp
}

// 確認 padding 是否正確
func (r *Ring[T]) PaddingBytes() uintptr {
	wpAddr := uintptr(unsafe.Pointer(&r.wp))
	rpAddr := uintptr(unsafe.Pointer(&r.rp))
	return rpAddr - wpAddr
}
