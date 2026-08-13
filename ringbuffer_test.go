package ringbuffer

import (
	"errors"
	"fmt"
	"math/bits"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"
)

func mustNew[T any](tb testing.TB, capacity int) *Ring[T] {
	tb.Helper()
	rb, err := New[T](capacity)
	if err != nil {
		tb.Fatalf("New(%d): %v", capacity, err)
	}
	return rb
}

// committedLen is a white-box assertion helper for tests that have stopped producer
// and consumer activity. It is intentionally not part of the public API.
func committedLen[T any](rb *Ring[T]) uint64 {
	return atomic.LoadUint64(&rb.wp) - atomic.LoadUint64(&rb.rp)
}

func TestRingInt(t *testing.T) {
	rb := mustNew[int](t, 100)
	_, ptr, err := rb.Set()
	if err != nil {
		t.Fatalf("Set error: %v", err)
	}
	*ptr = 88
	rb.SetAdv()

	_, ptr, err = rb.Get()
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if *ptr != 88 {
		t.Fatalf("want 88, got %v", *ptr)
	}
	rb.GetAdv()
}

func TestRingString(t *testing.T) {
	rb := mustNew[string](t, 100)
	_, ptr, err := rb.Set()
	if err != nil {
		t.Fatalf("Set error: %v", err)
	}
	*ptr = "hello"
	rb.SetAdv()

	_, ptr, err = rb.Get()
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if *ptr != "hello" {
		t.Fatalf("want hello, got %v", *ptr)
	}
	rb.GetAdv()
}

type structTest struct {
	hello string
}

func TestRingStruct(t *testing.T) {
	rb := mustNew[structTest](t, 100)
	_, ptr, err := rb.Set()
	if err != nil {
		t.Fatalf("Set error: %v", err)
	}
	ptr.hello = "hello world"
	rb.SetAdv()

	_, ptr, err = rb.Get()
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if ptr.hello != "hello world" {
		t.Fatalf("want hello world, got %v", ptr.hello)
	}
	rb.GetAdv()
}

func TestRingStructPtr(t *testing.T) {
	rb := mustNew[structTest](t, 100)
	m := structTest{hello: "test123"}
	_, ptr, err := rb.Set()
	if err != nil {
		t.Fatalf("Set error: %v", err)
	}
	*ptr = m // 存 pointer
	rb.SetAdv()

	_, ptr, err = rb.Get()
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if (*ptr).hello != "test123" {
		t.Fatalf("want test123, got %v", (*ptr).hello)
	}
	rb.GetAdv()
}

func BenchmarkRingString1024(b *testing.B) {
	rb := mustNew[string](b, 1024)
	for i := 0; i < b.N; i++ {
		val := "hello"
		for {
			_, ptr, err := rb.Set()
			if err == nil {
				*ptr = val
				rb.SetAdv()
				break
			}
		}
		for {
			_, ptr, err := rb.Get()
			if err == nil {
				_ = *ptr
				rb.GetAdv()
				break
			}
		}
	}
}

func BenchmarkRingString256(b *testing.B) {
	rb := mustNew[string](b, 256)
	for i := 0; i < b.N; i++ {
		val := "hello"
		for {
			_, ptr, err := rb.Set()
			if err == nil {
				*ptr = val
				rb.SetAdv()
				break
			}
		}
		for {
			_, ptr, err := rb.Get()
			if err == nil {
				_ = *ptr
				rb.GetAdv()
				break
			}
		}
	}
}

func BenchmarkRingString512(b *testing.B) {
	rb := mustNew[string](b, 512)
	for i := 0; i < b.N; i++ {
		val := "hello"
		for {
			_, ptr, err := rb.Set()
			if err == nil {
				*ptr = val
				rb.SetAdv()
				break
			}
		}
		for {
			_, ptr, err := rb.Get()
			if err == nil {
				_ = *ptr
				rb.GetAdv()
				break
			}
		}
	}
}

func BenchmarkRingString64(b *testing.B) {
	rb := mustNew[string](b, 64)
	for i := 0; i < b.N; i++ {
		val := "hello"
		for {
			_, ptr, err := rb.Set()
			if err == nil {
				*ptr = val
				rb.SetAdv()
				break
			}
		}
		for {
			_, ptr, err := rb.Get()
			if err == nil {
				_ = *ptr
				rb.GetAdv()
				break
			}
		}
	}
}

func BenchmarkRingStruct64(b *testing.B) {
	rb := mustNew[structTest](b, 64)
	for i := 0; i < b.N; i++ {
		val := structTest{hello: "hello"}
		for {
			_, ptr, err := rb.Set()
			if err == nil {
				*ptr = val
				rb.SetAdv()
				break
			}
		}
		for {
			_, ptr, err := rb.Get()
			if err == nil {
				_ = *ptr
				rb.GetAdv()
				break
			}
		}
	}
}

// 1. 測試基本寫入與讀取
func TestBasic(t *testing.T) {
	// 建立容量為 4 的 Ring Buffer (內部會取 2 的冪，所以 4 就是 4)
	rb := mustNew[int](t, 4)

	if rb.Capacity() != 4 {
		t.Fatalf("expected capacity 4, got %d", rb.Capacity())
	}

	// 寫入數據
	idx, ptr, err := rb.Set()
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	*ptr = 100
	rb.SetAdv()

	// 讀取數據
	idx2, ptr2, err := rb.Get()
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if *ptr2 != 100 {
		t.Errorf("expected 100, got %d", *ptr2)
	}
	if idx != idx2 {
		t.Errorf("index mismatch: write %d, read %d", idx, idx2)
	}
	rb.GetAdv()

	if got := committedLen(rb); got != 0 {
		t.Errorf("expected empty buffer, got committed length %d", got)
	}
}

// 2. 測試滿與空的狀態
func TestFullAndEmpty(t *testing.T) {
	rb := mustNew[int](t, 2) // 容量 2

	// 讀取空
	_, _, err := rb.Get()
	if err != ErrRingEmpty {
		t.Errorf("expected ErrRingEmpty, got %v", err)
	}

	// 寫入直到滿
	rb.Set()
	rb.SetAdv() // 1
	rb.Set()
	rb.SetAdv() // 2

	// 再次寫入應該失敗
	_, _, err = rb.Set()
	if err != ErrRingFull {
		t.Errorf("expected ErrRingFull, got %v", err)
	}

	// 讀取一個
	rb.Get()
	rb.GetAdv()

	// 現在應該又有空間寫入
	_, _, err = rb.Set()
	if err != nil {
		t.Errorf("expected success after read, got %v", err)
	}
}

// 3. 測試繞圈 (Wrap Around)
// 驗證 mask 邏輯是否正確，能不能重複利用空間
func TestWrapAround(t *testing.T) {
	size := 4
	rb := mustNew[int](t, size)

	// 1. 填滿
	for i := 0; i < size; i++ {
		_, ptr, _ := rb.Set()
		*ptr = i
		rb.SetAdv()
	}

	// 2. 讀出 2 個
	for i := 0; i < 2; i++ {
		_, ptr, _ := rb.Get()
		if *ptr != i {
			t.Errorf("expected %d, got %d", i, *ptr)
		}
		rb.GetAdv()
	}

	// 3. 再寫入 2 個 (這時應該繞回陣列開頭)
	for i := 0; i < 2; i++ {
		_, ptr, err := rb.Set()
		if err != nil {
			t.Fatalf("wrap around write failed: %v", err)
		}
		*ptr = size + i // 寫入 4, 5
		rb.SetAdv()
	}

	// 4. 讀出剩餘所有
	expected := []int{2, 3, 4, 5}
	for _, exp := range expected {
		_, ptr, err := rb.Get()
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if *ptr != exp {
			t.Errorf("expected %d, got %d", exp, *ptr)
		}
		rb.GetAdv()
	}
}

// 4. 測試併發安全性 (SPSC)
// 生產者與消費者在不同 Goroutine，傳輸大量數據確保沒掉包、沒順序錯亂
func TestConcurrency(t *testing.T) {
	rb := mustNew[int](t, 1024) // 足夠大的緩衝區，減少阻塞，但也測試 shadow 指標
	count := 1_000_000          // 一般模式測試一百萬次傳輸
	if testing.Short() {
		count = 100_000
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Producer
	go func() {
		defer wg.Done()
		for i := 0; i < count; i++ {
			for {
				_, ptr, err := rb.Set()
				if err == nil {
					*ptr = i
					rb.SetAdv()
					break
				} else if err == ErrRingFull {
					runtime.Gosched() // 讓出 CPU 等待消費
					continue
				} else {
					t.Errorf("producer unexpected error: %v", err)
					return
				}
			}
		}
	}()

	// Consumer
	go func() {
		defer wg.Done()
		for i := 0; i < count; i++ {
			for {
				_, ptr, err := rb.Get()
				if err == nil {
					if *ptr != i {
						t.Errorf("race condition or logic error! expected %d, got %d", i, *ptr)
					}
					rb.GetAdv()
					break
				} else if err == ErrRingEmpty {
					runtime.Gosched() // 讓出 CPU 等待生產
					continue
				} else {
					t.Errorf("consumer unexpected error: %v", err)
					return
				}
			}
		}
	}()

	// 設定超時，防止 Deadlock
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("test timed out, possible deadlock or shadow pointer logic error")
	}
}

// 5. 測試 Memory Leak (驗證指標是否被清空)
// 這是為了驗證你在 GetAdv 中加入的 `r.data[idx] = zero` 是否生效
func TestMemoryLeak(t *testing.T) {
	// 使用指標類型的 Ring Buffer
	rb := mustNew[*int](t, 4)

	val := 123

	// 寫入
	idx, ptr, _ := rb.Set()
	*ptr = &val
	rb.SetAdv()

	// 檢查內部 data 陣列 (因為我們在同一個 package，可以直接存取私有欄位 data)
	// 注意：這裡是白箱測試，依賴內部實作細節
	if rb.data[idx] == nil {
		t.Fatal("data should prevent nil before read")
	}

	// 讀取
	readIdx, readPtr, _ := rb.Get()
	if readPtr == nil || *readPtr != &val {
		t.Fatal("read failed")
	}

	// 執行 GetAdv (這一步應該要清空內部 array 的引用)
	rb.GetAdv()

	// 再次檢查內部 data 陣列
	// 如果你的 GetAdv 沒有 `r.data[rp&r.mask] = zero`，這裡就會失敗
	if rb.data[readIdx] != nil {
		t.Errorf("Memory Leak Detected! Slot %d was not cleared after GetAdv", readIdx)
	}
}

func TestCapacityRoundsUpToPowerOfTwo(t *testing.T) {
	tests := []struct {
		requested int
		want      uint64
	}{
		{requested: 1, want: 1},
		{requested: 2, want: 2},
		{requested: 3, want: 4},
		{requested: 63, want: 64},
		{requested: 65, want: 128},
		{requested: 100, want: 128},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("requested=%d", tt.requested), func(t *testing.T) {
			rb := mustNew[int](t, tt.requested)
			if got := rb.Capacity(); got != tt.want {
				t.Fatalf("Capacity() = %d, want %d", got, tt.want)
			}
			if got := len(rb.data); got != int(tt.want) {
				t.Fatalf("backing storage length = %d, want %d", got, tt.want)
			}
			if rb.Capacity()&(rb.Capacity()-1) != 0 {
				t.Fatalf("capacity %d is not a power of two", rb.Capacity())
			}
		})
	}
}

func TestNewRejectsInvalidCapacity(t *testing.T) {
	for _, capacity := range []int{0, -1, -1024} {
		t.Run(fmt.Sprintf("capacity=%d", capacity), func(t *testing.T) {
			rb, err := New[int](capacity)
			if !errors.Is(err, ErrInvalidCapacity) {
				t.Fatalf("New(%d) error = %v, want ErrInvalidCapacity", capacity, err)
			}
			if rb != nil {
				t.Fatalf("New(%d) returned non-nil ring on error", capacity)
			}
		})
	}
}

func TestNewCapacityRepresentabilityBoundary(t *testing.T) {
	maxPowerOfTwo := int(uint(1) << (bits.UintSize - 2))

	// A zero-sized element verifies the representable boundary without reserving a
	// massive amount of memory.
	rb, err := New[struct{}](maxPowerOfTwo)
	if err != nil {
		t.Fatalf("New(maxPowerOfTwo=%d): %v", maxPowerOfTwo, err)
	}
	if got := rb.Capacity(); got != uint64(maxPowerOfTwo) {
		t.Fatalf("Capacity() = %d, want %d", got, maxPowerOfTwo)
	}

	rb, err = New[struct{}](maxPowerOfTwo + 1)
	if !errors.Is(err, ErrInvalidCapacity) {
		t.Fatalf("New(first overflowing capacity) error = %v, want ErrInvalidCapacity", err)
	}
	if rb != nil {
		t.Fatal("New(first overflowing capacity) returned non-nil ring")
	}

	if bits.UintSize == 64 {
		allocated, allocationErr := New[byte](maxPowerOfTwo)
		if !errors.Is(allocationErr, ErrInvalidCapacity) {
			t.Fatalf("New(impossible byte allocation) error = %v, want ErrInvalidCapacity", allocationErr)
		}
		if allocated != nil {
			t.Fatal("New(impossible byte allocation) returned non-nil ring")
		}
	}
}

func TestReservationStateMachine(t *testing.T) {
	rb := mustNew[int](t, 1)

	if err := rb.SetAdv(); err != ErrNoWriteReservation {
		t.Fatalf("SetAdv() without Set() error = %v, want %v", err, ErrNoWriteReservation)
	}
	if err := rb.GetAdv(); err != ErrNoReadReservation {
		t.Fatalf("GetAdv() without Get() error = %v, want %v", err, ErrNoReadReservation)
	}
	if got := committedLen(rb); got != 0 {
		t.Fatalf("invalid advances changed committed length to %d", got)
	}

	_, writeSlot, err := rb.Set()
	if err != nil {
		t.Fatalf("Set(): %v", err)
	}
	*writeSlot = 42
	if idx, ptr, err := rb.Set(); err != ErrWriteReservationActive || idx != 0 || ptr != nil {
		t.Fatalf("second Set(): idx=%d ptr=%p err=%v, want active reservation error", idx, ptr, err)
	}
	if err := rb.SetAdv(); err != nil {
		t.Fatalf("SetAdv(): %v", err)
	}
	if err := rb.SetAdv(); err != ErrNoWriteReservation {
		t.Fatalf("second SetAdv() error = %v, want %v", err, ErrNoWriteReservation)
	}
	if got := committedLen(rb); got != 1 {
		t.Fatalf("double SetAdv() changed committed length to %d, want 1", got)
	}

	_, readSlot, err := rb.Get()
	if err != nil {
		t.Fatalf("Get(): %v", err)
	}
	if *readSlot != 42 {
		t.Fatalf("Get() value = %d, want 42", *readSlot)
	}
	if idx, ptr, err := rb.Get(); err != ErrReadReservationActive || idx != 0 || ptr != nil {
		t.Fatalf("second Get(): idx=%d ptr=%p err=%v, want active reservation error", idx, ptr, err)
	}
	if err := rb.GetAdv(); err != nil {
		t.Fatalf("GetAdv(): %v", err)
	}
	if err := rb.GetAdv(); err != ErrNoReadReservation {
		t.Fatalf("second GetAdv() error = %v, want %v", err, ErrNoReadReservation)
	}
	if got := committedLen(rb); got != 0 {
		t.Fatalf("double GetAdv() changed committed length to %d, want 0", got)
	}
}

func TestFailedReserveDoesNotAuthorizeAdvance(t *testing.T) {
	rb := mustNew[int](t, 1)

	_, slot, err := rb.Set()
	if err != nil {
		t.Fatalf("Set(): %v", err)
	}
	*slot = 7
	if err := rb.SetAdv(); err != nil {
		t.Fatalf("SetAdv(): %v", err)
	}
	if _, _, err := rb.Set(); err != ErrRingFull {
		t.Fatalf("Set() on full ring error = %v, want %v", err, ErrRingFull)
	}
	if err := rb.SetAdv(); err != ErrNoWriteReservation {
		t.Fatalf("SetAdv() after full error = %v, want %v", err, ErrNoWriteReservation)
	}

	_, slot, err = rb.Get()
	if err != nil {
		t.Fatalf("Get(): %v", err)
	}
	if *slot != 7 {
		t.Fatalf("Get() value = %d, want 7", *slot)
	}
	if err := rb.GetAdv(); err != nil {
		t.Fatalf("GetAdv(): %v", err)
	}
	if _, _, err := rb.Get(); err != ErrRingEmpty {
		t.Fatalf("Get() on empty ring error = %v, want %v", err, ErrRingEmpty)
	}
	if err := rb.GetAdv(); err != ErrNoReadReservation {
		t.Fatalf("GetAdv() after empty error = %v, want %v", err, ErrNoReadReservation)
	}
	if got := committedLen(rb); got != 0 {
		t.Fatalf("failed reservations/advances changed committed length to %d", got)
	}
}

func TestResetCancelsReservations(t *testing.T) {
	rb := mustNew[int](t, 1)
	if _, _, err := rb.Set(); err != nil {
		t.Fatalf("Set(): %v", err)
	}
	rb.Reset()
	if err := rb.SetAdv(); err != ErrNoWriteReservation {
		t.Fatalf("SetAdv() after Reset() error = %v, want %v", err, ErrNoWriteReservation)
	}

	_, slot, err := rb.Set()
	if err != nil {
		t.Fatalf("Set() after Reset(): %v", err)
	}
	*slot = 1
	if err := rb.SetAdv(); err != nil {
		t.Fatalf("SetAdv(): %v", err)
	}
	if _, _, err := rb.Get(); err != nil {
		t.Fatalf("Get(): %v", err)
	}
	rb.Reset()
	if err := rb.GetAdv(); err != ErrNoReadReservation {
		t.Fatalf("GetAdv() after Reset() error = %v, want %v", err, ErrNoReadReservation)
	}
}

func TestFailedOperationsDoNotChangeState(t *testing.T) {
	rb := mustNew[int](t, 2)

	idx, ptr, err := rb.Get()
	if err != ErrRingEmpty {
		t.Fatalf("Get() error = %v, want %v", err, ErrRingEmpty)
	}
	if got := committedLen(rb); idx != 0 || ptr != nil || got != 0 {
		t.Fatalf("failed Get() changed state: idx=%d ptr=%p committed=%d", idx, ptr, got)
	}

	for i := 0; i < 2; i++ {
		_, slot, setErr := rb.Set()
		if setErr != nil {
			t.Fatalf("Set() %d: %v", i, setErr)
		}
		*slot = i
		rb.SetAdv()
	}

	idx, ptr, err = rb.Set()
	if err != ErrRingFull {
		t.Fatalf("Set() error = %v, want %v", err, ErrRingFull)
	}
	if got := committedLen(rb); idx != 0 || ptr != nil || got != rb.Capacity() {
		t.Fatalf("failed Set() changed state: idx=%d ptr=%p committed=%d", idx, ptr, got)
	}
}

func TestFIFOAcrossManyWraps(t *testing.T) {
	const (
		capacity = 8
		cycles   = 4096
	)
	rb := mustNew[int](t, capacity)

	for base := 0; base < cycles; base += capacity {
		for i := 0; i < capacity; i++ {
			idx, slot, err := rb.Set()
			if err != nil {
				t.Fatalf("Set(%d): %v", base+i, err)
			}
			if want := uint64(base+i) & (capacity - 1); idx != want {
				t.Fatalf("write index = %d, want %d", idx, want)
			}
			*slot = base + i
			rb.SetAdv()
		}

		for i := 0; i < capacity; i++ {
			idx, slot, err := rb.Get()
			if err != nil {
				t.Fatalf("Get(%d): %v", base+i, err)
			}
			if want := uint64(base+i) & (capacity - 1); idx != want {
				t.Fatalf("read index = %d, want %d", idx, want)
			}
			if got := *slot; got != base+i {
				t.Fatalf("FIFO value = %d, want %d", got, base+i)
			}
			rb.GetAdv()
		}
	}
}

func TestResetRestoresInitialStateAndClearsReferences(t *testing.T) {
	rb := mustNew[*int](t, 4)
	values := [3]int{10, 20, 30}
	for i := range values {
		_, slot, err := rb.Set()
		if err != nil {
			t.Fatalf("Set(%d): %v", i, err)
		}
		*slot = &values[i]
		rb.SetAdv()
	}

	// Populate the consumer shadow before reset as well.
	if _, _, err := rb.Get(); err != nil {
		t.Fatalf("Get() before Reset(): %v", err)
	}
	rb.Reset()

	if got := committedLen(rb); got != 0 || rb.rp != 0 || rb.wp != 0 || rb.shadowRp != 0 || rb.shadowWp != 0 {
		t.Fatalf("Reset() did not restore counters: rp=%d wp=%d shadowRp=%d shadowWp=%d committed=%d",
			rb.rp, rb.wp, rb.shadowRp, rb.shadowWp, got)
	}
	for i, slot := range rb.data {
		if slot != nil {
			t.Fatalf("Reset() retained reference in slot %d", i)
		}
	}
	if _, _, err := rb.Get(); err != ErrRingEmpty {
		t.Fatalf("Get() after Reset() error = %v, want %v", err, ErrRingEmpty)
	}
	idx, _, err := rb.Set()
	if err != nil || idx != 0 {
		t.Fatalf("first Set() after Reset(): idx=%d err=%v, want idx=0 and nil error", idx, err)
	}
}

func TestProducerConsumerCountersAreCacheLineSeparated(t *testing.T) {
	rb := mustNew[int](t, 1)
	wpAddr := uintptr(unsafe.Pointer(&rb.wp))
	rpAddr := uintptr(unsafe.Pointer(&rb.rp))
	distance := rpAddr - wpAddr
	if got := distance; got < CacheLineSize {
		t.Fatalf("producer/consumer counter distance = %d bytes, want at least %d", got, CacheLineSize)
	}
}

func TestHotPathHasNoAllocations(t *testing.T) {
	rb := mustNew[int](t, 1)
	allocs := testing.AllocsPerRun(1000, func() {
		_, writeSlot, err := rb.Set()
		if err != nil {
			panic(err)
		}
		*writeSlot = 42
		rb.SetAdv()

		_, readSlot, err := rb.Get()
		if err != nil {
			panic(err)
		}
		if *readSlot != 42 {
			panic("unexpected ring value")
		}
		rb.GetAdv()
	})
	if allocs != 0 {
		t.Fatalf("hot path allocations = %f, want 0", allocs)
	}
}

func BenchmarkSPSCConcurrent(b *testing.B) {
	for _, capacity := range []int{64, 1024} {
		b.Run(fmt.Sprintf("Ring/cap=%d", capacity), func(b *testing.B) {
			benchmarkConcurrentRing(b, capacity)
		})
		b.Run(fmt.Sprintf("Channel/cap=%d", capacity), func(b *testing.B) {
			benchmarkConcurrentChannel(b, capacity)
		})
	}
}

func benchmarkConcurrentRing(b *testing.B, capacity int) {
	rb := mustNew[int](b, capacity)
	done := make(chan struct{})
	b.ReportAllocs()
	b.ResetTimer()

	go func() {
		defer close(done)
		for i := 0; i < b.N; i++ {
			for {
				_, slot, err := rb.Set()
				if err == nil {
					*slot = i
					rb.SetAdv()
					break
				}
				runtime.Gosched()
			}
		}
	}()

	last := 0
	for i := 0; i < b.N; i++ {
		for {
			_, slot, err := rb.Get()
			if err == nil {
				last = *slot
				rb.GetAdv()
				break
			}
			runtime.Gosched()
		}
	}
	<-done
	runtime.KeepAlive(last)
}

func benchmarkConcurrentChannel(b *testing.B, capacity int) {
	ch := make(chan int, capacity)
	done := make(chan struct{})
	b.ReportAllocs()
	b.ResetTimer()

	go func() {
		defer close(done)
		for i := 0; i < b.N; i++ {
			ch <- i
		}
	}()

	last := 0
	for i := 0; i < b.N; i++ {
		last = <-ch
	}
	<-done
	runtime.KeepAlive(last)
}
