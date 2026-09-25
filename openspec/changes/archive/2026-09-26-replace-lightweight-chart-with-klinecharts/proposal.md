## Why

当前图表使用 Lightweight Charts，而用户希望切换到 KLineChart，并能在拖动到左侧边界时继续查看更早的 K 线。现有后端历史接口没有时间游标；周期控件也在页面 header 中且只有固定的四个选项，无法满足图表上方的周期选择体验。

## What Changes

- **BREAKING（前端依赖）**：移除 Lightweight Charts，改用 KLineChart 绘制当前 Watchlist 交易对的实时 K 线，继续保留行情状态及页面现存的 K 线价格信息。
- 将周期选择控件移到 K 线图上方，并按 KLineChart 的周期模型适配 Binance USDⓈ-M 原生周期；不提供任意周期输入或自定义 K 线聚合。
- 扩展历史 K 线请求，使图表可按时间边界向前分页加载更早数据；保留现有默认查询行为及原生周期的实时 K 线订阅。
- 页面继续遵守移除分析面板后的展示范围，只保留行情图表和必要的交易对、周期及行情状态控件。

## Capabilities

### New Capabilities

- `market-kline-chart`: 定义交互式 K 线展示、历史数据向前分页、原生周期选择、实时更新及切换交易对/周期时的数据隔离行为。

### Modified Capabilities

无。

## Impact

- 前端：K 线图表组件、Dashboard 周期控件、K 线数据加载与订阅适配、相关依赖和测试。
- 后端：K 线 handler、repository 和行情流服务，增加时间游标并统一校验 Binance 原生周期。
- API：K 线历史接口增加可选 `endTime` 游标和 `has_more_before` 分页状态；未提供游标时保持现有最新数据查询语义。
- 验收：遵循项目 TDD 约定，覆盖图表历史分页、周期切换、实时更新、过期请求隔离和旧图表依赖移除；执行前后端相关回归。
