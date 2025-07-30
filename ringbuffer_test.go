package ringbuffer

import "testing"

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
