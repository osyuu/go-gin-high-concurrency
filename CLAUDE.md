# go-gin-high-concurrency

高並發搶票系統，使用 Gin + PostgreSQL + Redis Streams。

## 架構

```
HTTP Handler → Service → Repository (PostgreSQL)
                ↓
          Redis Inventory (Lua 原子扣庫存)
                ↓
          Redis Stream Queue → Order Worker → DB 持久化
```

**搶票流程（兩階段）：**
1. `PrepareOrder`：Lua 腳本原子扣 Redis 庫存，push 到 Redis Stream，立即回傳
2. `DispatchOrder`：Worker 消費 Stream，寫入 PostgreSQL（含 retry / poison message）

**Worker**：與 HTTP server 在同一 process 內啟動（`cmd/server/main.go`），使用獨立的 context，graceful shutdown 時先停 HTTP 再停 Worker。

**Poison message 策略**：失敗訊息留在 PEL，閒置 5s 後由 `XAUTOCLAIM` 回收重試，重試達 5 次後 ACK 丟棄並 log warning。

## 常用指令

### 開發環境

```bash
# 啟動 PostgreSQL（port 5432）、PostgreSQL-test（port 5433）、Redis（port 6379）
# 並自動執行 migrations
docker compose up -d

# 停止並清除 volume
docker compose down -v

# 啟動應用程式（預設 :8080）
go run ./cmd/server
```

### 測試

```bash
# 執行所有測試（需要 docker compose up -d 先跑起來）
go test ./...

# 執行單一 package 測試（test 目錄下有 service / integration / queue / handler / repository / cache / worker）
go test ./test/internal/service/...
```

> **注意**：integration / queue 測試共用同一個 Redis instance，測試之間若有 stream 狀態污染（NOGROUP 錯誤），重新執行即可，與程式碼無關。

### UAT（完整容器化，含 app）

```bash
# 啟動 UAT 環境（app 跑在 :8081）
docker compose -f docker-compose.uat.yml up -d --build

# 停止 UAT
docker compose -f docker-compose.uat.yml down -v
```

### Mock 重新生成

```bash
# 需要安裝 mockery：go install github.com/vektra/mockery/v2@latest
# 設定檔為 .mockery.yml，直接執行即可
mockery
```

## 環境變數

應用程式從環境變數讀設定，預設值見 `config/config.go`。測試環境固定使用 port 5433 / Redis DB 1（見 `config.LoadTestConfig()`）。

## 重要設計決策

- **Lua 腳本**：`DecreStock` / `RollbackStock` 使用 `redis.NewScript()` 預編譯，以 EVALSHA 執行，避免每次發送完整腳本
- **Logger**：使用 `logger.MQ` / `logger.Handler` / `logger.Service` / `logger.Worker` 等預建 logger，不要呼叫 `logger.WithComponent()`（已標記 Deprecated）
- **sync.Pool**：queue 的 `model.Order` 物件透過 pool 重用；取出後必須先 `*order = model.Order{}` reset 再 unmarshal
- **pgxpool**：MaxConns 25，MinConns 5，連線已由 pool 管理，不需手動控制
