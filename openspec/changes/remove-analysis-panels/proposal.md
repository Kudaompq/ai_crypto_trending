## Why

当前页面展示支撑位和压力位标记、整组分析面板及交易机会模块，包含额外的后台轮询、服务端计算和持久化。用户希望移除这些不再需要的功能及其数据，只保留行情图表和必要的行情交互。

## What Changes

- 移除 K 线图上的支撑位和压力位线、价格标签及摘要标记。
- 移除整个分析面板区域，包括交易建议、趋势、指标、支撑/压力位、形态和市场结构面板。
- **BREAKING**：移除交易机会按钮、弹窗、后台轮询、前端 API 与类型，以及后端 `/api/opportunities` 路由和对应的 handler、检测 service、repository、模型与数据库初始化逻辑。
- 删除专用于交易机会的 `backend/data/opportunities.db` 文件及其中已有记录；新版启动时不得重新创建该数据库。
- 保留 K 线图、交易对与周期选择、行情连接状态。后端仍被其他分析功能使用的计算不在本次移除范围内。

## Capabilities

### New Capabilities
- `chart-analysis-display`: 定义主页面展示边界，以及已移除分析面板和交易机会功能后的行为。

### Modified Capabilities

## Impact

- 前端：`Dashboard.vue`、`SimpleChart.vue`、交易机会组件及 `services/api.ts`。
- 后端：`/api/opportunities` 路由、handler、机会检测与持久化代码、机会专用数据模型及 SQLite 启动逻辑。
- 现有 `/api/opportunities` 调用方将收到 404；机会历史数据随专用数据库删除且不可恢复。
- 不改变实时行情订阅、K 线数据、交易对/周期选择及其他仍在使用的分析能力。
