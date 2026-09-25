## Why

目前界面只提供写死的 8 个交易对，用户无法选择其他由 Binance 合约数据源支持的交易对。实时行情连接建立后，即使上游 WebSocket 没有送来价格，页面仍显示“已连接”；价格停止更新时用户难以判断数据是否过期，也无法及时恢复。

## What Changes

- 保留现有常用交易对，允许用户手动添加、选择和移除自选交易对，并在浏览器中保留自选列表。
- 对自定义交易对进行格式及数据源可用性校验，让 K 线、分析、交易机会和实时行情使用一致的交易对规则；不支持的交易对给出明确反馈。
- 只有收到对应交易对与周期的实时价格事件后，界面才显示实时连接正常；显示最近一次行情更新时间，并在长时间无事件时提示数据过期。
- 上游连接失败或中断时自动重试，并以定期获取行情作为备用更新路径；恢复实时推送后回到实时状态，避免页面静默停在旧价格。
- 本次以手动维护自选交易对为范围，不包含交易所全量交易对搜索。

## Capabilities

### New Capabilities

- `custom-trading-pairs`: 用户管理自选 Binance 合约交易对，并在所有行情与分析入口一致地使用和校验所选交易对。
- `reliable-realtime-market-data`: 页面准确呈现实时报价连接和数据新鲜度，在推送中断时重试并提供备用更新。

### Modified Capabilities

无；当前 `openspec/specs/` 中没有既有能力规范。

## Impact

- 前端交易对状态、选择控件、行情展示与 API 调用：`frontend/src/stores/analysis.ts`、`frontend/src/views/Dashboard.vue`、`frontend/src/services/api.ts`。
- 后端交易对校验、SSE 接口与 Binance WebSocket 订阅：`backend/internal/handler/`、`backend/internal/service/market_stream_service.go`、`backend/internal/repository/binance.go`。
- 开发与 Docker 环境中的 API 代理、上游网络连通性及相关错误提示需要纳入验证；预计不新增数据库表或第三方服务。
