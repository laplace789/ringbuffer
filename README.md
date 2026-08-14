# spsc-ringbuffer

`spsc-ringbuffer` 是 Go 1.25.8 的泛型、bounded、lock-free SPSC（Single Producer,
Single Consumer）ring buffer。它使用零拷貝的兩階段 reservation API，適合單一
producer 與單一 consumer 間的高頻資料傳遞。

## 特點

- 泛型 `Ring[T]`，可保存 value、struct 或 pointer。
- 嚴格 SPSC：一個 producer goroutine 與一個 consumer goroutine。
- 熱路徑無 lock、無 allocation。
- 128-byte producer/consumer state separation，降低支援平台上的 false sharing。
- 明確偵測重複 reservation、未配對 advance 與無效容量。

## 安裝

```shell
go get github.com/laplace789/spsc-ringbuffer
```

## 快速開始

```go
package main

import (
	"fmt"
	"log"

	"github.com/laplace789/spsc-ringbuffer"
)

func main() {
	rb, err := ringbuffer.New[int](128)
	if err != nil {
		log.Fatal(err)
	}

	_, writeSlot, err := rb.Set()
	if err != nil {
		log.Fatal(err)
	}
	*writeSlot = 42
	if err := rb.SetAdv(); err != nil {
		log.Fatal(err)
	}

	_, readSlot, err := rb.Get()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(*readSlot)
	if err := rb.GetAdv(); err != nil {
		log.Fatal(err)
	}
}
```

等價的可編譯範例維護於 `example_test.go`，會由 `go test` 驗證，避免文件與公開
API 漂移。

## API 與狀態

- `New[T](capacity)`：建立 ring；容量向上取整為 2 的次方。
- `Set()`：producer reserve 一個可寫 slot；滿時回傳 `ErrRingFull`。
- `SetAdv()`：producer 發布已寫入的 slot。
- `Get()`：consumer reserve 一個可讀 slot；空時回傳 `ErrRingEmpty`。
- `GetAdv()`：consumer 釋放已讀取的 slot。
- `Capacity()`：回傳固定且已取整的容量。
- `Reset()`：停止兩側後清除資料、counters 與 reservations。

不提供併行長度快照。請直接以 `Set`／`Get` 的結果判斷當次操作，不要使用
「先檢查狀態、再操作」的流程。

## SPSC 與 pointer ownership 契約

- 只能有一個 goroutine 呼叫 `Set`／`SetAdv`。
- 只能有一個 goroutine 呼叫 `Get`／`GetAdv`。
- `Set` 成功至對應 `SetAdv` 成功前，slot pointer 由 producer 獨佔。
- `Get` 成功至對應 `GetAdv` 成功前，slot pointer 由 consumer 獨佔。
- Advance 成功後不得保留、解參考或修改先前的 pointer；slot 可能立即被重用。
- `Ring` 首次使用後不可 value-copy；只傳遞 `*Ring[T]`。`go vet` 可偵測常見複製。
- `Reset` 不具 concurrency safety，必須先停止 producer 與 consumer。

本套件不支援 MPSC、SPMC 或 MPMC。

## 測試與 benchmark

```shell
go test ./...
go test -short ./...
go test -race ./...
go vet ./...
go test -run '^$' -bench '^BenchmarkSPSCConcurrent$' -benchmem ./...
go test -run '^$' -fuzz '^FuzzRingStateMachine$' -fuzztime=10s .
```

Benchmark 同時測量真正雙 goroutine 的 Ring 與 buffered channel。CI 保存結果供
release 比較，但不使用受 runner 波動影響的絕對時間門檻。

## 支援範圍

- 正式工具鏈：Go 1.25.8。
- 正式架構：64-bit `amd64`、`arm64`。
- 其他架構僅 best-effort compile，不承諾 cache-line 最佳化。
- 固定 128-byte separation 涵蓋支援平台的常見 coherency granule，但不是所有
  CPU cache topology 的效能保證。

## 來源與授權

本專案衍生自 Terry.Mao 的
[goim ring buffer](https://github.com/Terry-Mao/goim/blob/master/internal/comet/ring.go)，
並加入泛型、lock-free SPSC 同步、shadow indices、嚴格 reservation state、測試、
benchmark 與 production hardening。

原始部分與本專案修改均依 [MIT License](LICENSE) 發布；完整 attribution 請見
[NOTICE](NOTICE)。
