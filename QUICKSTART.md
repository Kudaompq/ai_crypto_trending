# Originx 快速开始

Originx 展示 Binance USDⓈ-M 合约 K 线、实时 Watchlist 报价和可定制图表。Watchlist 存在 PostgreSQL 中，因此本地开发和完整 Docker 部署都需要 PostgreSQL。

## 环境要求

本地开发需要：

- Go 1.23 或更高版本
- Node.js 20.19+ 或 22.12+，以及 npm
- Docker Compose v2（用于启动 PostgreSQL）
- 能访问 Binance 行情 API 和 WebSocket 的网络

只运行完整 Docker 部署时，无需在主机上安装 Go 或 Node.js。

## 本地开发

### 1. 配置并启动 PostgreSQL

在仓库根目录运行：

```bash
cp .env.example .env
```

编辑 `.env`，为 `POSTGRES_PASSWORD` 设置本地密码，并在 `WATCHLIST_DATABASE_URL` 中填入同一个密码。为避免 URL 解析问题，请使用 URL 安全字符组成的密码。

启动数据库并确认其状态为 `healthy`：

```bash
docker compose up -d postgres
docker compose ps
```

### 2. 启动前后端

仍在仓库根目录运行：

```bash
./start.sh
```

脚本会读取 `.env`、检查前端依赖、编译 Go 后端并启动前后端。打开 `http://localhost:5173`。默认后端地址为 `http://localhost:8080`；检查健康状态：

```bash
curl --noproxy '*' http://127.0.0.1:8080/api/health
```

如果 8080 端口被占用：

```bash
PORT=18080 ./start.sh
```

此时后端健康检查地址为 `http://127.0.0.1:18080/api/health`，Vite 会自动将 `/api` 请求代理到该端口。按 Ctrl+C 停止前后端；PostgreSQL 和数据卷仍会保留。需要停止数据库时运行 `docker compose stop postgres`。

## 完整 Docker 部署

先按上面的步骤创建并配置 `.env`，然后在仓库根目录运行：

```bash
docker compose up -d --build
docker compose ps
```

访问地址：

- 前端：`http://localhost:9999`
- 后端健康检查：`http://localhost:8888/api/health`
- PostgreSQL：主机回环地址 `127.0.0.1:5432`（默认）

查看服务日志：

```bash
docker compose logs -f postgres backend frontend
```

停止完整服务：

```bash
docker compose down
```

`docker compose down` 会保留 `postgres-data` 卷。不要加 `-v`，否则会删除保存 Watchlist 的数据库卷。卷不等同于备份；备份和恢复命令见 [README.md](README.md#数据持久化与备份)。

## 使用页面

- 在左侧 Watchlist 选择交易对；点击 `+` 添加，拖动手柄调整顺序，点击 `×` 移除。
- 在 K 线图表工具栏选择周期、添加 MA、EMA、BOLL、MACD 或 OI 指标，并使用绘图工具。
- 图表指标设置和绘图保存在当前浏览器；Watchlist 及顺序保存在服务端数据库，同一部署的访问者共用一份列表。
- 行情状态说明：`正在连接实时行情` 表示正在等待行情；`实时更新` 表示已收到实时事件；`备用更新（非实时）` 表示当前使用 REST 备用行情；`行情不可用或已过期` 表示近期没有有效行情。

如果 Watchlist 数据库同步失败，页面会显示本地缓存并禁止修改，直到数据库恢复同步。若 Binance 实时流不可用，状态会明确显示为备用更新或行情不可用，不会把 REST 行情标成实时。
