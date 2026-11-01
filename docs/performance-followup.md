# 第二輪 Queue 效能改善報告

**Version note: the earlier sections preserve the working-tree experiment. See the final PR supplement for clean master versus the actual PR.**

日期：2026-11-01。環境：Apple M4 Pro、darwin/arm64、go version go1.26.10 darwin/arm64、`-cpu=12`。

## 結果與適用範圍

完成兩項修改：停用重試資訊日誌時略過參數配置，以及可選的 Ring 初始／最低保留容量。
預設縮容策略、任務容量限制、FIFO、timeout、retry 次數與 shutdown 排程維持原行為。

- 空 logger、重試 10 次：配置 **21 → 11 次**，**1,144 → 824 B/op**；時間中位數減少 **8.3%**。
- 每批 1,024 筆的 Ring 填滿／清空，啟用保留 1,024 格：穩態配置 **18 → 0 次／批**，時間減少 **29.2%**，換算吞吐量增加 **41.3%**。
- 多 producer 的端到端情境：啟用保留容量後每筆配置位元組約減少 **7.4–7.6%**；本次沒有證明端到端吞吐提升。
- 成本：Ring 清空後的 backing-array 槽位從 **32 B 增至 16,384 B**。這不是 RSS；已出列的 task 引用仍清空。

數字取四輪樣本中位數；時間百分比不是統計顯著性宣告，不適用於任意業務工作負載。

## 版本定義

基準為 `adf9a1cf9bd0424c9c1286754083bd6038c985df` 加上開始實作時已有的工作目錄修改，保存於
[fixtures.json / baseline](performance/2026-11-01/fixtures.json)。
這些既有修改只作為量測 fixture 保存，**不包含在本 PR 的產品程式修改中**。
本 PR 不包含 retry timer 重用與 Jitter 修正，因此此報告的絕對值是當時工作目錄比較，
不是乾淨 master 與 PR head 的直接比較；兩項優化的差異以固定 fixture 隔離。

這些既有修改涵蓋 `queue.go`、`handle_result_test.go`、`job/job.go`、`job/job_test.go`，四個版本皆包含。
因此不能把此報告當成該 commit 的乾淨版本與最終工作目錄直接比較。

| 版本 | 修改 |
| --- | --- |
| baseline | 實作前工作目錄 |
| logging | 僅增加可選 InfoEnabled 與重試呼叫端檢查 |
| ring | 僅增加 WithQueueMinCapacity 與 Ring 保留容量 |
| combined | 兩項合併 |

所有版本使用相同 benchmark。baseline／logging 沒有新 Ring API，僅在測試中提供 no-op
`WithQueueMinCapacity` adapter，讓 `retained=true` 標籤跑相同工作負載下的舊縮容策略。
這個 adapter 不修改舊版 Ring 行為；原版 `retained=true` 實際仍只保留 2 格。

## 1. 重試日誌

[logger.go](../logger.go) 新增可選介面 `InfoLogger`（含 `InfoEnabled() bool`）；既有 `Logger`
介面不變，未提供方法的 logger 繼續輸出重試日誌。`NewEmptyLogger()` 回報 false。
[queue.go](../queue.go) 在每次重試記錄之前檢查，避免建立 `...any` 參數；允許執行期間切換設定，
不把結果快取在建構時。自訂 logger 必須讓方法可安全併發呼叫。
這個檢查目前用於重試資訊日誌，不抑制錯誤／panic 日誌。

每次操作為一個完整 handler 執行；retry delay 為 1ns，以凸顯框架成本，並非真實網路重試延遲。

| 情境 | 原版 ns/op | 修改後 ns/op | 時間變化 | 吞吐量變化 | B/op 前 → 後 | allocs/op 前 → 後 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| 重試 0 次 | 763.85 | 752.55 | -1.5% | +1.5% | 576 → 576 | 8 → 8 |
| 重試 1 次 | 1,088.00 | 1,054.00 | -3.1% | +3.2% | 856 → 824 | 12 → 11 |
| 重試 3 次 | 2,044.00 | 1,937.00 | -5.2% | +5.5% | 920 → 824 | 14 → 11 |
| 重試 10 次 | 4,355.00 | 3,995.50 | -8.3% | +9.0% | 1144 → 824 | 21 → 11 |

## 2. Ring 保留容量

[options.go](../options.go) 新增 `WithQueueMinCapacity(n)`，只對內建 Ring 有效：

- 設定初始配置與縮容下限；小於 2 時用 2，預設仍為 2。
- 有正數 `WithQueueSize` 上限時取較小值，但至少保留 2 格；待處理任務上限仍獨立遵守。
- 支援非 2 的次方，選項先後順序不影響容量上限。
- 超過最低容量時仍可擴容；使用量降至 25% 時減半，但不低於設定下限。

```go
q := queue.NewPool(4,
    queue.WithLogger(queue.NewEmptyLogger()),
    queue.WithQueueMinCapacity(1024),
)
defer q.Release()
```

此例預先配置 16 KiB 的 Ring 槽位空間（本機每個 interface 槽位 16 B）；無 backlog 時仍保留。
未設定 `WithQueueSize` 時，這不會把佇列限制為 1,024 筆。

以下每次操作為 **1,024 次入列＋1,024 次出列**；不是單筆任務執行時間。
預配置位於 ResetTimer 之前，表中的 0 B/op 不包含建構時配置。

| 情境 | 原版 ns/op | 修改後 ns/op | 時間變化 | 吞吐量變化 | B/op 前 → 後 | allocs/op 前 → 後 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| 預設容量策略 | 14,202.50 | 14,164.00 | -0.3% | +0.3% | 55968 → 55968 | 18 → 18 |
| 保留 1,024 格 | 14,235.50 | 10,072.50 | -29.2% | +41.3% | 55968 → 0 | 18 → 0 |

## 3. 端到端：含公開提交、handler 完成與 Release

每批最多 1,024 筆，使用空任務與空 logger。多 producer 情境固定 4 workers，
計入 producer goroutine 建立成本。入列失敗會使 benchmark 失敗，結尾確認成功數量。
每筆平均成本包含排程與同步；不是排隊時間、單筆 latency 或 p99。

| 情境 | 原版 ns/op | 修改後 ns/op | 時間變化 | 吞吐量變化 | B/op 前 → 後 | allocs/op 前 → 後 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| 1 producer，預設容量 | 1,389.00 | 1,408.00 | +1.4% | -1.3% | 727 → 727 | 9 → 9 |
| 1 producer，保留 1,024 格 | 1,388.50 | 1,394.50 | +0.4% | -0.4% | 726 → 672 | 9 → 9 |
| 4 producer，預設容量 | 1,466.50 | 1,575.50 | +7.4% | -6.9% | 727 → 727 | 9 → 9 |
| 4 producer，保留 1,024 格 | 1,470.00 | 1,489.00 | +1.3% | -1.3% | 727 → 672 | 9 → 9 |

以下為既有端到端 benchmark，全部維持預設容量：

| 情境 | 原版 ns/op | 修改後 ns/op | 時間變化 | 吞吐量變化 | B/op 前 → 後 | allocs/op 前 → 後 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| 1 workers | 1,176.50 | 1,170.00 | -0.6% | +0.6% | 726 → 726 | 9 → 9 |
| 4 workers | 1,364.50 | 1,441.50 | +5.6% | -5.3% | 726 → 726 | 9 → 9 |
| 12 workers | 1,872.50 | 1,929.00 | +3.0% | -2.9% | 726 → 726 | 9 → 9 |
| 4 workers＋afterFn | 1,411.00 | 1,403.50 | -0.5% | +0.5% | 766 → 766 | 11 → 11 |

保留容量減少的是 Ring backing-array 的配置位元組；Message、context、handler goroutine 等仍存在。
攤提後的 `allocs/op` 為整數，因此端到端仍顯示 9，不能解讀成完全沒有減少 Ring 配置。
首輪系列中有些端到端情境變慢，以下保留所有結果及樣本範圍，不將其略去或宣稱沒有退步。

## 預設路徑的延長複測

初始四輪出現 4-worker 及 4-producer 預設情境變慢，因此另以每個情境 **1 秒、6 個樣本**，
baseline／combined 交替先後執行；兩個測試分開啟動。結果不與先前 300ms 樣本混算。

| 情境 | baseline 中位數（範圍）ns/task | combined 中位數（範圍）ns/task | 時間變化 |
| --- | ---: | ---: | ---: |
| 4 workers，單 producer，預設容量 | 1,327.5（1,265–1,427） | 1,374（1,285–1,402） | +3.5% |
| 4 workers，4 producers，預設容量 | 1,420.5（1,358–1,471） | 1,421（1,326–1,544） | +0.04% |

4-producer 情境未重現先前 +7.4% 的差距；4-worker 仍有 +3.5% 的中位數差異。
兩者樣本範圍皆重疊，沒有足夠證據判定穩定退步，也不能保證效能等效。
因此本次主張限於重試日誌與 burst 配置改善，不宣稱預設路徑加速；
若部署主要是極短任務且關注幾個百分點的吞吐差異，應在目標環境進一步確認此項風險。

原始結果：[baseline 複測](performance/2026-11-01/baseline-followup.txt)、
[combined 複測](performance/2026-11-01/combined-followup.txt)、
[複測摘要](performance/2026-11-01/followup-summary.json)。重現：

```sh
rtk proxy python3 docs/performance/2026-11-01/run.py /tmp/queue-performance-followup-rerun --followup
```

## 個別修改的貢獻

| 情境（ns/op 中位數） | baseline | logging | ring | combined |
| --- | ---: | ---: | ---: | ---: |
| 重試10次 | 4,355.00 | 4,020.00 | 4,326.50 | 3,995.50 |
| Ring burst，保留1024格 | 14,235.50 | 14,264.00 | 10,067.00 | 10,072.50 |
| 4 producers，保留1024格 | 1,470.00 | 1,440.00 | 1,511.50 | 1,489.00 |

## 樣本範圍與限制

| 情境 | baseline ns/op 範圍 | combined ns/op 範圍 |
| --- | ---: | ---: |
| `RingBurstDrain/retained=false` | 13948–14337 | 14019–14193 |
| `RingBurstDrain/retained=true` | 14098–14584 | 10064–10081 |
| `QueueBurstProducers/producers=1/retained=false` | 1345–1614 | 1374–1544 |
| `QueueBurstProducers/producers=1/retained=true` | 1269–1555 | 1315–1584 |
| `QueueBurstProducers/producers=4/retained=false` | 1432–1619 | 1459–1683 |
| `QueueBurstProducers/producers=4/retained=true` | 1461–1721 | 1316–1682 |
| `HandleRetry/retries=0` | 754.6–805.4 | 726.7–786.7 |
| `HandleRetry/retries=1` | 1068–1139 | 1023–1093 |
| `HandleRetry/retries=3` | 2034–2121 | 1867–1956 |
| `HandleRetry/retries=10` | 4303–4414 | 3913–4019 |
| `QueueEndToEnd/workers=1` | 1128–1209 | 1118–1202 |
| `QueueEndToEnd/workers=4` | 1230–1400 | 1371–1508 |
| `QueueEndToEnd/workers=12` | 1867–1894 | 1803–1957 |
| `QueueAfterFnEndToEnd` | 1343–1481 | 1327–1546 |

僅四個樣本，不提供顯著性或跨機器保證；重疊範圍也不能證明兩者等效。
本次未量測 RSS、p95／p99、真實 I/O 任務、外部 broker 或長期 soak。
保留容量是否值得，應依實際 burst 規模與同時存在的 Queue 數量選擇；1,024 只是本次測量值。

## 驗證與重現

- 修改前、修改後 `rtk proxy go test -race ./...` 均通過（含既有 goleak 檢查）。
- 新增測試涵蓋：舊 logger 相容性、資訊日誌啟用／停用／動態切換、重試次數、非 2 次方容量、
  FIFO 跨 wrap／resize、多次 burst、空槽釋放引用、容量拒絕、shutdown 拒絕及 NewPool 選項傳遞。
- `golangci-lint run` 尚有 **1 個既有問題**：`job/job_test.go:47` 的 `Bool(enabled)` 被 modernize 建議改為 `new(enabled)`。
  該檔與基準 snapshot 相同，本次沒有更動；本次新增／修改部分無其餘 lint 問題。不宣稱全套 lint 通過。
- 此為 CORE CODE：影響共享 queue、public options 與日誌擴充契約；失敗會影響所有使用者。
  需由維護者逐行審查，正式合併建議包含 module owner 的兩位 reviewer；PR 交付狀態與乾淨分支驗證另見下方 PR 補充。

每個情境 300ms、每版本四個樣本、`-cpu=12`、不使用 race instrumentation 量測效能。
輪序：baseline → logging → ring → combined；反序；ring → baseline → combined → logging；
logging → combined → baseline → ring。版本序列執行，benchmark 期間沒有同時跑 Go tests 或 lint。

重建四版本並量測（輸出目錄必須不存在）：

```sh
rtk proxy python3 docs/performance/2026-11-01/run.py /tmp/queue-performance-rerun
rtk proxy python3 docs/performance/2026-11-01/summarize.py /tmp/queue-performance-rerun
```

重現腳本從基準 commit 加上當時 diff 還原版本，套用 [fixtures.json / implementation](performance/2026-11-01/fixtures.json) 與相同 benchmark，
並核對 SHA256；已執行 `--prepare-only` 驗證四個版本均可完整重建且 hash 一致。
量測實作由 fixture patch 保存，不依賴目前 queue.go 是否包含其他本機修改；benchmark 檔案仍須符合 hash。

- 原始數據：[baseline](performance/2026-11-01/baseline.txt)、[logging](performance/2026-11-01/logging.txt)、[ring](performance/2026-11-01/ring.txt)、[combined](performance/2026-11-01/combined.txt)。
- [四輪摘要與完整樣本 JSON](performance/2026-11-01/summary.json)、[版本與檔案 SHA256](performance/2026-11-01/manifest.json)。
- [重現腳本](performance/2026-11-01/run.py)、[彙整腳本](performance/2026-11-01/summarize.py)。

回復本次修改時，只移除 InfoLogger／重試檢查及最低容量選項相關差異；
不要還原整份 queue.go，以免覆蓋開始前既有的使用者修改。

## PR supplement: clean master versus the actual PR

The PR branch `perf/queue-allocation-costs` includes only logging checks, Ring retention, tests and evidence.
This comparison uses clean `adf9a1c` versus the actual PR implementation, excluding the pre-existing local
retry-timer reuse and Jitter fixes. It is separate from the historical working-tree comparison above.

Same machine, 300ms per case, two rounds of three samples each (six samples per version).
Order: base then head; head then base. Runs were sequential, without concurrent Go tests or lint.
The base uses a test-only no-op retention adapter, retaining the original resizing policy.

| Scenario | base ns/op median (range) | PR ns/op median (range) | Time change | B/op before -> after | allocs/op before -> after |
| --- | ---: | ---: | ---: | ---: | ---: |
| `RingBurstDrain/retained=false` | 14293 (14050-14449) | 14329 (13951-14447) | +0.3% | 55968 -> 55968 | 18 -> 18 |
| `RingBurstDrain/retained=true` | 14229 (13945-14566) | 10170 (10080-10326) | -28.5% | 55968 -> 0 | 18 -> 0 |
| `QueueBurstProducers/producers=1/retained=false` | 1357.5 (1298-1452) | 1385 (1333-1458) | +2.0% | 726 -> 726 | 9 -> 9 |
| `QueueBurstProducers/producers=1/retained=true` | 1360.5 (1272-1460) | 1408 (1327-1434) | +3.5% | 726 -> 672 | 9 -> 9 |
| `QueueBurstProducers/producers=4/retained=false` | 1412 (1369-1535) | 1452.5 (1377-1536) | +2.9% | 727 -> 727 | 9 -> 9 |
| `QueueBurstProducers/producers=4/retained=true` | 1482.5 (1397-1561) | 1446.5 (1321-1549) | -2.4% | 727 -> 672 | 9 -> 9 |
| `HandleRetry/retries=0` | 753.15 (734.3-796.5) | 769 (747.9-800.7) | +2.1% | 576 -> 576 | 8 -> 8 |
| `HandleRetry/retries=1` | 1081.5 (1028-1137) | 1040.5 (1020-1064) | -3.8% | 856 -> 824 | 12 -> 11 |
| `HandleRetry/retries=3` | 1704.5 (1613-1785) | 1619 (1609-1680) | -5.0% | 1416 -> 1320 | 20 -> 17 |
| `HandleRetry/retries=10` | 3705 (3519-3881) | 3471.5 (3414-3552) | -6.3% | 3376 -> 3056 | 48 -> 38 |
| `QueueEndToEnd/workers=1` | 1196.5 (1131-1212) | 1157 (1132-1191) | -3.3% | 726 -> 726 | 9 -> 9 |
| `QueueEndToEnd/workers=4` | 1412.5 (1297-1506) | 1371.5 (1273-1450) | -2.9% | 726 -> 726 | 9 -> 9 |
| `QueueEndToEnd/workers=12` | 1859.5 (1718-2086) | 1863 (1776-1901) | +0.2% | 726 -> 726 | 9 -> 9 |
| `QueueAfterFnEndToEnd` | 1458 (1281-1717) | 1347 (1325-1369) | -7.6% | 766 -> 766 | 11 -> 11 |

The evidence supports lower allocation costs, not a general end-to-end throughput improvement.
Default-capacity timings vary; neither acceleration nor equivalence is guaranteed by these medians.
Ring burst units are per 1,024-task batch; other cases are per task. Initial preallocation is excluded
from timing, while idle backing storage still increases from 32 B to 16 KiB when retention is enabled.

Isolated PR validation: `go test -race ./...` passed; `go tool -modfile=go.tools.mod golangci-lint run`
reported **0 issues**; `git diff --check` passed. The historical lint warning above is not part of this PR.
The historical reproduction script also passed `--prepare-only` from the isolated worktree,
matching all four source manifests.

```sh
rtk proxy python3 docs/performance/2026-11-01/pr-compare.py /tmp/queue-pr-comparison
```

The destination must not exist. The script extracts the clean base twice and overlays the PR's four
production files on the head variant, using identical benchmarks. It writes source hashes; compare
those against this report when reproducing from a later source version.

- [PR base raw samples](performance/2026-11-01/pr-base.txt), [PR head raw samples](performance/2026-11-01/pr-head.txt).
- [PR summary](performance/2026-11-01/pr-summary.json), [PR source hashes](performance/2026-11-01/pr-manifest.json).
- [PR comparison script](performance/2026-11-01/pr-compare.py).
