# Queue 效能改善報告

日期：2026-10-31。基準 commit：`9721be66093a0e2889129e4751154d8388698b98`。
比較對象是該 commit 與本次工作目錄的修改；沒有更動 public API 或新增依賴。

## 結果

兩項修改都已完成。容量邊界的 Ring 擴縮容抖動已消除；一般短任務的端到端
每筆時間中位數降低約 **10–53%**，配置次數由 **17 降至 11 次／任務**。
這是單機空任務 benchmark 的結果，不代表實際 I/O 或 CPU 密集工作也會有相同增幅。

下表皆取 **6 次樣本的中位數**。Queue 的 ns/op 是整批任務的平均處理成本，
不是個別任務的排隊延遲或 p99。吞吐量增幅由修改前時間除以修改後時間計算。

| 端到端情境 | 原版 ns/task | 修改後 ns/task | 每筆時間降低 | 換算吞吐量增幅 | allocs/task |
| --- | ---: | ---: | ---: | ---: | ---: |
| 1 worker | 2,537.5 | 1,196.0 | 52.9% | 112.2% | 17 → 11 |
| 4 workers | 1,610.5 | 1,438.5 | 10.7% | 12.0% | 17 → 11 |
| 12 workers | 2,065.0 | 1,859.0 | 10.0% | 11.1% | 17 → 11 |
| 4 workers，含 afterFn | 1,654.0 | 1,358.5 | 17.9% | 21.8% | 17 → 13 |

一般任務的配置量約從 **1,000–1,002 B/task 降為 854 B/task**，下降約 14.6–14.7%；
有 afterFn 時從 **1,001 B/task 降為 894 B/task**，下降約 10.7%。

4／12 workers 的時間樣本範圍有重疊，不能把中位數差異當成已證明的統計顯著改善。
配置次數的減少則在本次所有樣本中一致。

## 1. Ring 縮容門檻

[ring.go](../ring.go) 原本在使用量降至容量的 50% 時就把容量減半。
當 backlog 在 1,024 與 1,025 之間反覆變動，每次 enqueue/dequeue 都會
在 1,024 與 2,048 容量之間切換，重新配置並複製整個 backlog，期間持有 Ring mutex。

修改後：滿載仍擴大一倍，但使用量降至 **25% 或以下**才縮成一半。
最大待處理數限制、FIFO 與最小 buffer 大小 2 均保留。

| 穩定 backlog；每次加入＋取出 | 原版 ns/op | 修改後 ns/op | 原版 B/op | 修改後 B/op | allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: |
| 64 筆 | 351.25 | 9.36 | 3,456 | 0 | 2 → 0 |
| 1,024 筆 | 3,453.50 | 9.14 | 51,200 | 0 | 2 → 0 |

這是刻意涵蓋容量邊界的 microbenchmark，不能解讀成整個 Queue 快數百倍。
穩態不再配置；首次擴容仍會配置，攤提到大量迭代後 benchmark 顯示 0 B/op、0 allocs/op。
代價是縮容更晚，部分負載下會保留較大的 backing array，以換取較少的複製與配置。

## 2. 重用 dispatcher 與 worker

[queue.go](../queue.go) 改為單一常駐 dispatcher 直接呼叫 `Request()`，
再將任務交給可重用的 worker goroutine。backend 的 Request 仍然序列執行。
動態增加 worker 會建立新的 worker；減少時，退役的 worker 會先完成目前任務。
`Start()` 改為冪等，避免重複呼叫造成多個 dispatcher。

| 每筆任務新增的 goroutine | 原版 | 修改後 |
| --- | ---: | ---: |
| 沒有 afterFn | 3：Request、work、handler | 1：handler |
| 有 afterFn | 3：callback 在 work 中執行 | 2：handler、callback |

上述數字是程式碼執行路徑的計數，不是 runtime profile 的取樣結果；
不包含初始化與擴增 worker 時建立的常駐 goroutine。

`handle()` 的 timeout、retry 與 panic 處理維持原實作。
保留獨立 handler goroutine，才能維持 handler 不理會 context 時，timeout 仍釋放
Queue 執行名額的行為；這次沒有改成同步執行 handler。

原本 afterFn 不占用執行名額，可能等待後續任務完成。因此修改後以受 routineGroup
追蹤的獨立 goroutine 執行 callback，保留此行為，且 `Release()` 仍會等待 callback。
[options.go](../options.go) 的 callback 說明也已修正為實際行為。

常駐 worker 的取捨：Queue 閒置時也會保留 worker goroutine，直到縮減或釋放；
它們阻塞等待任務，不忙迴圈輪詢。阻塞式 backend 的 Request 仍需由 backend 自行
在 Shutdown 時解除阻塞，與既有介面契約一致。

## 分離兩項修改的貢獻

額外測量「只改 Ring、保持原始 queue.go」版本。下表單位均為 ns/op 中位數。

| 情境 | 原版 | 僅修 Ring | 兩項都修正 |
| --- | ---: | ---: | ---: |
| Ring backlog 64 | 351.25 | 9.32 | 9.36 |
| Ring backlog 1,024 | 3,453.50 | 9.15 | 9.14 |
| Queue，1 worker | 2,537.5 | 2,537.5 | 1,196.0 |
| Queue，4 workers | 1,610.5 | 1,669.0 | 1,438.5 |
| Queue，12 workers | 2,065.0 | 2,067.0 | 1,859.0 |
| Queue，4 workers＋afterFn | 1,654.0 | 1,636.0 | 1,358.5 |

Ring 修正的主要收益出現在容量邊界。本次批次端到端情境沒有重現同樣程度的
擴縮容抖動；一般任務仍為 17 allocs/task，必須加上排程修改才降至 11。
因此不能把 Ring 邊界測試的倍數套用到一般吞吐量，也不把這裡 Ring-only 的小幅波動視為改善。

## 測量方式與限制

- 環境：Apple M4 Pro、darwin/arm64、Go 1.26.10、`GOMAXPROCS=12`（`-cpu=12`）。
- 三個版本使用完全相同的 [performance_test.go](../performance_test.go)。
- 每個情境 500ms，3 輪、每輪 2 個樣本，共 6 個樣本。
- 輪序：原版 → Ring-only → 修改後；修改後 → 原版 → Ring-only；Ring-only → 修改後 → 原版。
- 版本依序執行，沒有同時執行其他測試或 lint；未使用 race instrumentation 做效能測量。
- Queue 為單 producer、空任務，每批最多 1,024 筆。每批等到 handler 完成訊號，
  callback 情境則等到 callback 訊號；最後 `Release()` 納入計時，確認全部任務成功。
- 每次入列都檢查錯誤，不把容量滿載後的拒絕算成成功吞吐量。
- Queue 建構呼叫位於 ResetTimer 之前，非同步啟動與最後釋放的成本仍可能攤提在測量內。
- 未測真實業務任務、多 producer、外部 broker、長時間穩態、RSS 或 p95/p99。
  CPU 排程、主機背景負載及 GC 都會造成波動；本報告未宣稱統計顯著性。

原版與修改後的 6 次時間樣本範圍：

| Queue 情境 | 原版 ns/task 範圍 | 修改後 ns/task 範圍 |
| --- | ---: | ---: |
| 1 worker | 2,511–2,557 | 1,183–1,256 |
| 4 workers | 1,544–1,658 | 1,299–1,607 |
| 12 workers | 1,894–2,118 | 1,846–1,952 |
| 4 workers＋afterFn | 1,536–1,688 | 1,341–1,421 |

## 重現與原始資料

將基準 commit 匯出到兩個暫存目錄，兩者都複製本次 `performance_test.go`；
其中一個再覆蓋本次 `ring.go` 作為 Ring-only。修改後版本使用目前工作目錄。
每輪在各自目錄執行下列命令，依上述輪序重複三輪：

```sh
rtk proxy go test -run '^$' \
  -bench '^(BenchmarkRingBoundary|BenchmarkQueueEndToEnd|BenchmarkQueueAfterFnEndToEnd)$' \
  -benchmem -benchtime=500ms -count=2 -cpu=12 .
```

原始資料包含每個樣本、執行時間與程式碼 SHA256，可核對測試與版本：

- [原版](performance/baseline.txt)
- [僅修 Ring](performance/ring-only.txt)
- [兩項都修正](performance/optimized.txt)
- [中位數與範圍 JSON](performance/summary.json)

## 正確性驗證

- 修改前 `go test -race ./...` 通過。
- 修改後全套 `go test -race -covermode=atomic -coverprofile=/tmp/queue-performance-coverage.out ./...` 通過。
  Queue package statement coverage 為 **93.2%**；既有 goleak TestMain 同時檢查 goroutine 洩漏。
- `go tool -modfile=go.tools.mod golangci-lint run`：**0 issues**。
- 在 `-cpu=1,4,12` 各重複 30 次、開啟 race detector 的四項測試全部通過：
  動態增減／暫停恢復、錯誤／panic／retry／timeout 後重用 worker、阻塞 backend／重複 Start、
  afterFn 等待下一筆任務。
- [scheduling_test.go](../scheduling_test.go) 另外驗證 FIFO 跨越擴縮容、25% 縮容門檻，
  以及 Release 等待尚未結束的 callback；原有 shutdown、容量限制等測試亦通過。
- afterFn 等待後續任務的回歸測試已確認在原版通過，最終修改版也通過。

此為核心排程修改；正式合併前建議由模組維護者與另一位 reviewer 逐行檢查。
