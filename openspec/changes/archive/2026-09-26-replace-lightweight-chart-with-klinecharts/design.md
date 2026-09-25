## Context

参见 [proposal.md](./proposal.md) 与 [market-kline-chart spec](./specs/market-kline-chart/spec.md)。当前 Dashboard 管理选中交易对、周期、K 线 SSE、备用请求和新鲜度状态；K 线 REST 接口支持 `limit`，后端最多接受 500 条，但没有时间游标。K 线 store 默认一次拉取 100 条，当前图表只显示最近 100 根。实时流的周期校验目前仅覆盖六个 Binance 原生周期。

KLineChart 的数据加载器提供历史 `getBars`、实时 `subscribeBar` 和清理 `unsubscribeBar`；`setPeriod` 接受周期跨度与单位。本变更只映射 Binance 原生周期，不使用 KLineChart 周期模型生成 Binance 未提供的数据。[KLineChart 数据集成](https://klinecharts.com/guide/data-integration)；[KLineChart setPeriod](https://klinecharts.com/api/instance/setPeriod)；[Binance USDⓈ-M K 线 REST API](https://developers.binance.com/en/docs/catalog/core-trading-derivatives-trading-usd-s-m-futures/api/rest-api/market-data)

## Goals / Non-Goals

**Goals:**

- 让图表组件通过分页加载并保留已加载历史，同时保持当前选中交易对的实时更新。
- 让前端控件和后端 REST/SSE 共用 Binance 原生周期列表，并映射为 KLineChart 的周期跨度与类型。
- 将周期控件放在 K 线绘图区上方，沿用 store 当前的 `1d` 默认周期。
- 替换前端图表依赖，正确释放图表实例、观察器、分页请求和行情订阅。

**Non-Goals:**

- 不提供任意周期输入，也不合成 Binance 未原生提供的周期。
- 不增加画线工具、技术指标面板、支撑/压力位或交易机会功能。
- 不改变交易对 Watchlist、Binance USDⓈ-M 数据源、SSE 协议和现有价格新鲜度语义。

## Decisions

1. **以 KLineChart 作为绘制引擎，通过 Vue 组件封装实例生命周期。** 组件挂载时初始化图表和数据加载器，卸载时销毁图表与监听器；将现有 `Candle` 的毫秒时间戳及 OHLCV 数值映射到图表数据。价格、涨跌幅、高低价、成交量和 ATR 继续由 Vue 展示层计算。图表依赖使用锁定版本并更新 npm lockfile。
   - 备选方案：继续维护 Lightweight Charts 或手写蜡烛图。二者都不符合用户选择 KLineChart 的方向，也会保留旧依赖或自绘维护成本。

2. **将历史分页收敛到图表数据加载器和现有 K 线 REST 接口。** 图表每页默认请求 100 根，后端仍限制单次 `limit` 最大为 500。向左加载时以前端最早 K 线开盘时间减 1 毫秒作为排他 `endTime` 游标；API 层将可选游标传递到 handler、service 和 Binance repository。历史结果按开盘时间升序返回。响应增加 `has_more_before`，图表据此设置是否还能向左分页；当最后一页恰好满页时，额外一次空页请求可以确认历史结束。未带 `endTime` 的现有调用维持最新数据语义。
   - 备选方案：只在浏览器缓存现有 100 根数据。该方式不能满足拖动到左边界继续加载更早历史的要求。
   - Binance K 线接口按开盘时间识别 K 线，并提供 `endTime` 与 `limit` 参数；分页使用时间边界而不重复请求已显示范围。[Binance K 线接口](https://developers.binance.com/en/docs/catalog/core-trading-derivatives-trading-usd-s-m-futures/api/rest-api/market-data)

3. **前后端共享 Binance 原生周期映射。** 使用 `1m`、`3m`、`5m`、`15m`、`30m`、`1h`、`2h`、`4h`、`6h`、`8h`、`12h`、`1d`、`3d`、`1w`、`1M` 作为有效周期。REST 与 SSE 使用同一校验来源。前端映射到 KLineChart 的 `{ span, type }`：分钟、小时、日、周和月分别映射到对应跨度和类型；月周期 `1M` 必须保留大写 M，避免与分钟 `1m` 混淆。未支持周期返回明确参数错误。
   - 备选方案：仅在前端显示周期并让 Binance 上游自行拒绝。这样错误信息不一致，也会为非法周期建立无效实时订阅。
   - KLineChart 周期映射依据：[setPeriod API](https://klinecharts.com/api/instance/setPeriod)；有效周期依据：[Binance USDⓈ-M K 线接口](https://developers.binance.com/en/docs/catalog/core-trading-derivatives-trading-usd-s-m-futures/api/rest-api/market-data)。

4. **由图表适配器连接现有唯一 K 线 SSE，并向 Dashboard 暴露状态。** 适配器实现 KLineChart 的历史读取及 `subscribeBar`/`unsubscribeBar`，复用 Dashboard 当前的新鲜度检测和备用行情行为；同一交易对/周期只保留一条浏览器 SSE。交易对或周期变化时先关闭旧订阅，再加载新历史并订阅新目标。备用 REST 更新通过适配器更新图表，但状态仍显示为非实时。
   - 备选方案：保留 Dashboard SSE 并让图表再开启一条 SSE。重复连接会浪费资源，并可能令图表和页面状态不同步。
   - KLineChart 官方数据接入生命周期见[数据加载器文档](https://klinecharts.com/guide/data-integration)。

5. **将原生周期控件移入图表卡片上沿。** 从 Dashboard header 移出周期按钮；在绘图区上方显示原生周期快捷选择，更新现有 store 周期并调用 KLineChart 对应的周期设置。沿用 `1d` 默认值，不新增周期输入框或收藏周期存储。窄屏下控件可换行或横向滚动。
   - 备选方案：保留 header 控件并在图表卡片重复显示。重复入口容易产生状态不同步，也不符合指定位置。

6. **移除旧库相关 UI 与依赖，并保留 KLineChart 的许可信息。** 删除 `lightweight-charts` 依赖、图表专属类型与旧图表导入；审查并移除旧的 TradingView 专属归属文字，按 KLineChart Apache-2.0 分发内容保留所需 LICENSE/NOTICE 信息。[KLineChart 仓库与许可](https://github.com/klinecharts/KLineChart)
   - 备选方案：留下 TradingView 归属链接。换库后该文字会错误指向新图表的来源。

7. **先以行为测试验证失败，再实现前后端改动。** 前端覆盖初始化、向左翻页及游标、排序去重、无更多数据、错误重试、周期映射、实时事件、旧请求隔离、卸载时关闭订阅；后端覆盖 `endTime` 校验和透传、默认查询兼容、500 条上限及全部原生周期。测试使用本地假数据源或可注入客户端，不依赖 Binance 网络。
   - 备选方案：仅验证构建或检查图表渲染。它们不能证明翻页边界、实时订阅释放或错误恢复正确。

## Risks / Trade-offs

- [锁定的 Go SDK 对时间游标或完整周期集的支持与预期不同] → 先检查当前 `go.mod` 版本，用请求构造测试确认参数；不要无依据升级 SDK。
- [相邻页游标处理错误会造成重复或漏 K 线] → 用固定时间戳覆盖边界前后数据，验证严格早于游标、升序、去重及 `has_more_before`。
- [图表快速切换时旧请求或 SSE 晚到] → 以 symbol/interval 请求代次隔离结果，并验证取消订阅与旧回调丢弃。
- [KLineChart 周期映射和 Binance interval 大小写不一致] → 将每个原生 interval 到 `{ span, type }` 的映射独立测试，尤其覆盖 `1m` 与 `1M`。
- [新库的尺寸、自适应或数据加载 API 与现有测试环境不兼容] → 为 Vue 组件创建可控的图表依赖边界，使用真实浏览器验收缩放、向左翻页及容器 resize。
- [新依赖带来不同许可证义务] → 随锁文件升级一并审阅包内 LICENSE、NOTICE 和生产构建产物。

## Migration Plan

1. 先部署支持可选 `endTime`、`has_more_before` 和统一原生周期校验的后端；不带游标的现有请求继续工作。
2. 部署 KLineChart 前端适配、原生周期控件和更新后的依赖锁文件；保持当前自选交易对存储格式及默认 `1d` 周期。
3. 在端到端验收通过后删除 Lightweight Charts 实现、包依赖及误导性的旧 TradingView 归属文字。
4. 回滚时恢复旧前端即可继续调用不带游标的 K 线接口；后端新增的可选游标不要求数据库迁移。
