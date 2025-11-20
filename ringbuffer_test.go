package ringbuffer

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestRingInt(t *testing.T) {
	rb := New[int](100)
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
	rb := New[string](100)
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
	rb := New[structTest](100)
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
	rb := New[structTest](100)
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
	rb := New[string](1024)
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
	rb := New[string](256)
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
	rb := New[string](512)
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
	rb := New[string](64)
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
	rb := New[structTest](64)
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
	rb := New[int](4)

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

	if rb.Len() != 0 {
		t.Errorf("expected empty buffer, got len %d", rb.Len())
	}
}

// 2. 測試滿與空的狀態
func TestFullAndEmpty(t *testing.T) {
	rb := New[int](2) // 容量 2

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
	rb := New[int](size)

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
	rb := New[int](1024)  // 足夠大的緩衝區，減少阻塞，但也測試 shadow 指標
	const count = 1000000 // 測試一百萬次傳輸

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
	rb := New[*int](4)

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
