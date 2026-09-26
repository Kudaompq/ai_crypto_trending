## Why

当前指标启用状态和参数按交易对分别保存。切换币种后需要重复设置指标，无法在自选交易对间保持一致的图表观察方式。

## What Changes

- 将 MA、EMA、BOLL、MACD 和 OI 的启用状态与参数改为跨交易对共用，并继续跨周期保存。
- 图表绘图对象仍按交易对分别保存，并继续跨周期保留。
- OI 数据仍按当前交易对和周期加载；共享的只是 OI 指标显示设置。

## Capabilities

### New Capabilities
- 无

### Modified Capabilities
- `market-kline-chart`: 指标启用状态和参数跨交易对共享；绘图对象按交易对隔离。

## Impact

- 前端本地存储服务 `chartPreferences.ts` 和图表组件 `SimpleChart.vue`。
- 前端存储及图表组件测试；不改变 API、后端、指标计算或数据来源。
