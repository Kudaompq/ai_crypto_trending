## Why

当前交易对和添加/删除操作都集中在 header，币种较多时不便于浏览和切换。当前 K 线由自定义 DOM 图形绘制；以 Lightweight Charts 替换后，可以沿用项目自有 Binance 行情，同时获得成熟的缩放、拖动和时间轴交互。

## What Changes

- 在 K 线图左侧新增 Watchlist，展示所有当前可用的预设和自定义交易对及各自实时价格；选择一项后，该交易对成为当前图表和行情请求的目标。
- 移除 header 中的交易对下拉框，并将添加、删除交易对的入口放入 Watchlist；继续使用现有本地持久化和 Binance 合约校验行为。
- 将手工绘制的 K 线组件替换为 Lightweight Charts 蜡烛图；保留现有周期选项、Binance 历史数据和当前选中交易对的实时更新及行情状态。
- 为 Watchlist 中的所有交易对提供实时最新价，并保留现有图表区域内的当前价格、涨跌、高低价、成交量和 ATR 信息。
- 本次不提供完整的 TradingView 绘图工具栏；Lightweight Charts 的自定义绘图工具作为后续独立需求。

## Capabilities

### New Capabilities

- `lightweight-kline-chart`: 使用 Lightweight Charts 显示并持续更新 Binance USDⓈ-M 合约 K 线，支持现有周期和图表交互。
- `market-watchlist-quotes`: 在 Watchlist 中展示每个自选交易对的当前实时价格，且只向页面转发其自选列表中的行情。

### Modified Capabilities

- `custom-trading-pairs`: 将交易对浏览、选择、添加和删除入口收敛到左侧 Watchlist，并移除交易机会相关的过时行为描述。

## Impact

- 前端：`Dashboard.vue`、现有 `SimpleChart.vue`、交易对 store、行情 API、样式和相应 Vitest 用例；新增 `lightweight-charts` 运行依赖。
- 行情接口：图表继续使用现有 REST K 线和 SSE 实时流；Watchlist 增加批量价格快照及多交易对实时价格 SSE。
- 多币种最新价使用 Binance USDⓈ-M 全市场 mini ticker 行情流，经后端筛选后只推送当前页面 Watchlist 中的交易对，避免浏览器为每个币种单独建立行情连接。
- 图表数据适配：将现有毫秒时间戳和 OHLCV 数据转换为 Lightweight Charts 的蜡烛图数据，并将当前交易对和周期的更新送入图表。
- 工作流：所有行为修改遵循项目要求的 TDD；实现时先添加并运行失败的前后端行为测试，再完成适配、实时流验证和浏览器验证。
