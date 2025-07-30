
# ringbuffer

高效能 SPSC（單生產單消費）**泛型環形緩衝區**，支援 Go 1.18+ 泛型，  
內建 cache line padding 以最佳化多核心 CPU 效能。  
專為**高頻生產/消費、即時佇列**等場景打造，極低延遲、低 GC 負擔。

---

## 特點

- **泛型支援**：`Ring[T]`，可存任意型別（`int`、`string`、struct、pointer…）
- **極速無鎖（SPSC）**：單生產/單消費場景下 lock-free、atomic 操作
- **自動 cache line padding**：防止 false sharing，提升多核心吞吐
- **API 簡單、易於整合**
- **可直接與 Go channel 效能對比**

---

## 安裝

```shell
go get github.com/laplace789/ringbuffer
```

---

## 用法說明

### 1. 匯入套件

```go
import "github.com/yourname/ringbuffer"
```

### 2. 建立環形緩衝區（指定型別與容量）

```go
rb := ringbuffer.New      // 存 int
rb := ringbuffer.New   // 存 string
rb := ringbuffer.New // 存結構體
rb := ringbuffer.New  // 存結構體指標
```

### 3. 生產與消費

```go
// 生產者寫入
if _, ptr, err := rb.Set(); err == nil {
    *ptr = 123           // 根據泛型型別指派
    rb.SetAdv()          // 必須明確前進寫指標
}

// 消費者讀取
if _, ptr, err := rb.Get(); err == nil {
    fmt.Println(*ptr)    // 讀取資料
    rb.GetAdv()          // 必須明確前進讀指標
}
```

### 4. 常用 API

- `Set()`：取得可寫 slot 指標（指標未前進）
- `SetAdv()`：寫入完畢，前進寫指標
- `Get()`：取得可讀 slot 指標（指標未前進）
- `GetAdv()`：讀取完畢，前進讀指標
- `Len()`：當前 buffer 內部資料量
- `Capacity()`：buffer 容量
- `Reset()`：重置讀寫指標（資料未清空）

---

## 範例：自訂 struct

```go
type MyMsg struct {
    ID   int
    Text string
}

rb := ringbuffer.New 

// 生產者
if _, ptr, err := rb.Set(); err == nil {
    ptr.ID = 99
    ptr.Text = "hello"
    rb.SetAdv()
}

// 消費者
if _, ptr, err := rb.Get(); err == nil {
    fmt.Println(ptr.ID, ptr.Text)
    rb.GetAdv()
}
```

---

## SPSC 注意事項

> **只能單一生產者（Set/SetAdv）、單一消費者（Get/GetAdv）各自獨立 goroutine 運作！**  
> 多生產/多消費請自行加鎖或使用其他 queue。

---

## 測試

```shell
go test -v ./ringbuffer
go test -race ./ringbuffer      # 偵測 data race（SPSC 模式應該通過）
go test -bench=. ./ringbuffer   # 跑 benchmark
```

- 已內建 int/string/struct/pointer 各型別單元測試
- 壓力測試與 wrap around 測試

---

## Benchmark 範例

```go
func BenchmarkRingInt(b *testing.B) {
	rb := ringbuffer.New 
	for i := 0; i < b.N; i++ {
		for {
			_, ptr, err := rb.Set()
			if err == nil {
				*ptr = i
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
```

可用 Go 原生 channel 作效能對比：

```go
func BenchmarkChanInt(b *testing.B) {
	ch := make(chan int, 1024)
	go func() {
		for i := 0; i < b.N; i++ {
			ch <- i
		}
		close(ch)
	}()
	for i := 0; i < b.N; i++ {
		_ = <-ch
	}
}
```

---

## 常見問題（FAQ）

**Q: 可否多 goroutine 同時寫/讀？**  
A: 不行！本設計只支援 SPSC（單一生產者、單一消費者），多生產/消費請自行加鎖或用其他 queue。

**Q: 如何判斷 buffer 已滿或為空？**  
A: `Set()` 失敗（回傳 `ErrRingFull`）代表滿，`Get()` 失敗（回傳 `ErrRingEmpty`）代表空。

**Q: buffer 實際佔用內存為多少？**  
A: buffer 大小為你設定的容量（2 的冪次，超過自動向上補足），每格一份資料＋內部指標＋padding。

**Q: Cache line size 可以調整嗎？**  
A: 可以，請改 `CacheLineSize` 常數（跨平台可用 build tag 切換）。

**Q: 如果指標爆掉（超大運行）會怎樣？**  
A: `uint64` 累加，理論極長期後會回繞（數百年級別），一般場景下可忽略。

---

## 專案限制

- 僅適合 SPSC 場景（Single Producer, Single Consumer）
- 不保證多生產/多消費時的資料安全
- 實作依賴 atomic，效能高度優化但非 thread-safe for MPMC

---

## 參考資料

- [Go 泛型官方說明](https://go.dev/doc/go1.18)
- [Cache line padding & false sharing in Go](https://dave.cheney.net/2014/06/30/how-to-write-benchmarks-in-go)
- [Disruptor concurrency pattern](https://lmax-exchange.github.io/disruptor/)
- [Goim](https://github.com/Terry-Mao/goim/blob/master/internal/comet/ring.go)
---

## 授權

MIT License
