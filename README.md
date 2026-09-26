# Originx

Originx 是 Binance USDⓈ-M 合约行情工作台，提供自选交易对、K 线图表、实时价格和图表工具。前端使用 KLineCharts Pro；Go 服务端访问 Binance 行情，并通过 HTTP 和 Server-Sent Events（SSE）向浏览器提供数据。

## 功能

- **共享 Watchlist**：添加、选择、删除和拖动排序合约交易对；显示最新价格和 24 小时涨跌幅。所有访问同一部署的客户端共用 PostgreSQL 中的列表。
- **可定制 K 线图表**：选择 Watchlist 中的交易对与周期，按需添加 MA、EMA、BOLL、MACD 和未平仓持仓量（OI）指标，并使用图表绘图工具。
- **行情状态**：区分连接中、实时更新、REST 备用更新和行情不可用。只有收到有效实时事件后才显示“实时更新”；实时流中断后会尝试 REST 备用行情。
- **本地保存图表设置**：指标设置和绘图保存在当前浏览器；Watchlist 和顺序保存在服务端 PostgreSQL。图表设置不会在浏览器之间同步。
- **分析 API**：后端保留趋势、技术指标、支撑/压力位、蜡烛图形态和市场结构的综合分析接口；当前主界面以图表和 Watchlist 为主。

## 快速开始

本地开发和 Docker 部署的完整步骤见 [QUICKSTART.md](QUICKSTART.md)。本地开发需要 Go 1.23+、Node.js（20.19+ 或 22.12+）、npm 和 Docker Compose v2：

```bash
cp .env.example .env
# 编辑 .env：设置 POSTGRES_PASSWORD，并在 WATCHLIST_DATABASE_URL 中使用相同的 URL 安全密码
docker compose up -d postgres
./start.sh
```

打开 `http://localhost:5173`。脚本会从 `.env` 读取数据库连接配置、编译后端并启动前后端；按 Ctrl+C 停止服务。后端默认使用 8080 端口；若该端口被占用，可运行 `PORT=18080 ./start.sh`。

也可以用 Docker 启动完整服务：

```bash
docker compose up -d --build
```

打开 `http://localhost:9999`。Docker 模式还会在 `http://localhost:8888` 发布后端 API；PostgreSQL 默认只绑定主机的 `127.0.0.1:5432`。

## 数据持久化与备份

PostgreSQL 的 `postgres-data` 卷保存共享 Watchlist。数据库初始化时，旧版浏览器 Watchlist 只会由首个完成同步的客户端导入一次；之后所有客户端以数据库中的列表为准。新部署没有账户隔离，同一部署的访问者共用一份 Watchlist。

停止或重建容器时不要使用 `docker compose down -v`，该命令会删除 PostgreSQL 数据卷。普通的 `docker compose down` 会保留该卷，但卷持久化不能代替备份。

备份数据库：

```bash
docker compose exec -T postgres sh -c 'pg_dump -Fc -U "$POSTGRES_USER" "$POSTGRES_DB"' > watchlist.dump
```

恢复会清理并重建目标数据库中的对象。确认目标数据库可以被覆盖后执行：

```bash
docker compose exec -T postgres sh -c 'pg_restore -U "$POSTGRES_USER" -d "$POSTGRES_DB" --clean --if-exists --no-owner' < watchlist.dump
```

请将备份文件保存在仓库之外，并按部署环境的保留策略进行加密和异地存放。

## 行情状态说明

K 线历史数据和 OI 由后端从 Binance 获取；实时 K 线与 Watchlist 报价通过后端 SSE 推送到浏览器。SSE 的 `ready` 事件只表示浏览器已连接到后端，不表示 Binance 行情已经到达。

实时事件中断约 15 秒后，页面会尝试通过 REST 更新行情，并将状态标为“备用更新（非实时）”；实时推送恢复后会切回“实时更新”。如果行情停止变化，请同时检查页面状态和“最近行情”时间。

## API

所有 API 路径以 `/api` 开头。需要 Binance USDⓈ-M 合约交易对的接口会校验交易对格式和数据源支持情况。

| 方法与路径 | 用途 |
| --- | --- |
| `GET /api/health` | 健康检查 |
| `GET /api/symbols/validate?symbol=BTCUSDT` | 校验并规范化合约交易对 |
| `GET /api/kline?symbol=ETHUSDT&interval=1d&limit=100` | 获取历史 K 线；支持 `endTime` 毫秒时间戳分页，最多 500 根 |
| `GET /api/open-interest?symbol=ETHUSDT&interval=1d&limit=100` | 获取 OI 历史数据；支持 `startTime` 和 `endTime` |
| `GET /api/analysis?symbol=ETHUSDT&interval=1d&limit=100` | 获取综合分析结果 |
| `GET /api/stream?symbol=ETHUSDT&interval=1m` | 通过 SSE 推送 K 线事件和连接状态 |
| `GET /api/watchlist` | 获取共享 Watchlist 与版本号 |
| `POST /api/watchlist/import-legacy` | 首次启用服务端 Watchlist 时导入旧版浏览器列表 |
| `POST /api/watchlist/symbols`、`DELETE /api/watchlist/symbols/:symbol` | 添加或移除交易对 |
| `PUT /api/watchlist/order` | 更新共享列表顺序 |
| `GET /api/watchlist/prices?symbols=BTCUSDT,ETHUSDT` | 获取报价快照 |
| `GET /api/watchlist/stream?symbols=BTCUSDT,ETHUSDT` | 通过 SSE 推送 Watchlist 报价和状态 |

K 线周期：`1m`、`3m`、`5m`、`15m`、`30m`、`1h`、`2h`、`4h`、`6h`、`8h`、`12h`、`1d`、`3d`、`1w`、`1M`。OI 数据受 Binance 上游接口的周期和时间范围限制。

## 项目结构

```text
backend/
  cmd/server/             Go 服务入口与路由
  internal/handler/       HTTP、SSE 输入和响应
  internal/service/        行情、Watchlist 与分析业务
  internal/repository/     Binance 与 PostgreSQL 数据访问
  internal/database/      PostgreSQL 初始化和迁移
  internal/indicator/     技术指标
frontend/src/
  views/Dashboard.vue     主页面与 Watchlist
  components/SimpleChart.vue  KLineCharts Pro 图表
  stores/analysis.ts       前端行情和 Watchlist 状态
  services/               API、图表数据源和本地偏好
```

## 技术栈与开发检查

- 后端：Go、Gin、PostgreSQL（pgx）、Binance REST/WebSocket
- 前端：Vue 3、TypeScript、Pinia、Vite、Element Plus、KLineCharts Pro

运行后端测试：

```bash
go test -count=1 ./backend/...
```

运行前端测试和类型检查/生产构建：

```bash
cd frontend
npm test
npm run build
```
