## Context

动机见 `proposal.md`。当前 Watchlist 的顺序由预设列表和自选列表拼接得到；`SimpleChart.vue` 通过 KLineCharts core 10.0.3 展示 Binance K 线，但没有研究指标或绘图状态。服务端已有 Binance SDK 与统一的 `/api` 路由，可复用现有行情接入边界。

KLineCharts 当前版本提供 MA、EMA、BOLL、MACD 指标与交互式绘图覆盖层。参考项目 `klinecharts/preview` 使用 Pro 包、Polygon 数据源和 9.x 版本；本项目沿用其图表工具布局思路，继续使用当前 core 版本与 Binance 数据源。

Binance USDⓈ-M 历史 OI 使用 `GET /futures/data/openInterestHist`，周期为 5m、15m、30m、1h、2h、4h、6h、12h、1d，最多 500 条且只提供最近 1 个月。1m、3m 不属于该接口支持周期，实测也返回空数组。参见 [Binance USDⓈ-M Market Data](https://developers.binance.info/en/docs/catalog/core-trading-derivatives-trading-usd-s-m-futures/api/rest-api/market-data) 和 [KLineCharts 指标](https://klinecharts.com/en-US/guide/indicator)、[绘图覆盖层](https://klinecharts.com/en-US/guide/overlay)。

## Goals / Non-Goals

**Goals:**

- 让 Watchlist 全列表可以拖动排序，并在浏览器刷新后恢复。
- 在图表中配置价格叠加指标、MACD/OI 副图和常见绘图对象。
- 以交易对为隔离单位持久化研究指标、参数和绘图对象；同一交易对切换周期时复用。
- OI 只呈现 Binance 为所选周期和时间范围实际返回的数据；缺少数据时留空。

**Non-Goals:**

- 增加新的行情供应商、OI 推算/插值、账户级同步或服务端用户配置存储。
- 恢复已退役的独立分析卡片、自动支撑/压力位或交易机会功能。
- 改变 K 线周期、实时 K 线订阅和历史 K 线分页行为。

## Decisions

1. **前端维护 Watchlist 顺序，使用独立的浏览器存储项。**
   在 `analysis` store 中保存完整 symbol 顺序，并在可见交易对集合改变时规范化：保留仍有效且不重复的已保存项，将新交易对追加到末尾。拖动只更新顺序，不触发 `setSymbol`。使用原生 Pointer Events 支持鼠标和触摸，不增加拖拽依赖。备选方案是保持预设与自选分组排序；它无法表达用户要求的全列表位置。

2. **图表偏好按 symbol 存储，不以 interval 建索引。**
   使用单独版本化的 localStorage 文档，包含每个 symbol 的启用指标、指标参数和绘图对象。首次访问读取默认设置；损坏或旧版本数据按默认设置恢复，且不影响行情加载。切换 symbol 时只加载目标 symbol 的配置。周期切换时不改配置，只重载 K 线与研究数据。备选方案是按 symbol+interval 存储，会违背用户已确定的跨周期共用要求。

3. **沿用锁定的 KLineCharts core 10.0.3，而不迁入 preview 的 Pro/Polygon 技术栈。**
   使用库内 MA、EMA、BOLL、MACD 指标和其内置水平线、趋势线、射线、平行通道、斐波那契覆盖层；由 Vue 控件管理可见性和参数，序列化 indicator/overlay 描述并在图表重建后恢复。研究状态单独于 OHLCV 与实时 SSE 生命周期，避免图表切换导致偏好串到别的 symbol。

4. **通过现有后端 Binance Repository 提供 OI 查询。**
   新增只读 `/api/open-interest` 路由及 service/repository 路径，使用项目锁定的 `github.com/adshao/go-binance/v2 v2.8.9` 的 `NewOpenInterestStatisticsService`，不升级 SDK。handler 复用交易对校验，接受所选周期及 K 线时间范围；仅将受 Binance 支持的周期传给上游。未知/无样本周期返回空数据，不构成错误；格式错误、无效交易对和上游请求失败继续使用可区分的错误响应。

5. **将 OI 返回样本作为离散数据映射到自己的副图。**
   OI study 使用 KLineCharts 自定义副图指标，把上游样本按其时间戳映射至对应 K 线时间位置，只为实际样本输出值；缺项为 null，不延续前值。默认画 `sumOpenInterestValue` 名义价值，并按交易对的计价资产标注单位；本次不增加数量/名义价值切换控件。前端只在 OI 开启时、对当前 symbol/interval/已加载时间范围请求数据；对 symbol 或 interval 变化取消旧请求并校验响应版本。新样本随启用的 OI pane 定时刷新，刷新节奏不快于每分钟一次。Binance 仅保存最近一个月且分页单次最多 500 条，超过可用范围的副图保持空白。

6. **保存绘图时机使用图表事件而非定时全量写入。**
   overlay 完成创建、移动、编辑或删除时读取当前 symbol 的 overlays 并做短延迟合并写入；图表销毁前同步最终状态。恢复时只重建已完成的对象，避免把正在绘制的临时锚点写入存储。系统自动分析标记继续关闭，手动画出的相同价位线是独立 overlay。

## Risks / Trade-offs

- Binance OI 历史仅覆盖最近 1 个月、最多 500 条一页，且没有 1m/3m 样本 → 按实际接口边界留空，不伪造连续性；对超出数据范围的时间段不画线。
- OI 是 REST 历史统计而非当前 K 线 WebSocket 事件 → 仅在 study 激活时按分钟级节奏刷新最新时间片，UI 不把它标为逐笔实时数据。
- 用户可绘制对象和参数增加浏览器存储用量，且浏览器存储可能被清除 → 控制在每个 symbol 的图表配置内；读写失败时保留当前会话可用性并显示存储提示。
- 指标/绘图恢复与 K 线重新初始化存在竞态 → 在图表和目标 symbol/interval 就绪后恢复，并以请求代次阻止旧 OI 响应覆盖新状态。

## Migration Plan

- 不变更 SQLite schema，也不要求清理现有浏览器存储。首次启动时没有 Watchlist 顺序或图表偏好时沿用当前列表顺序和空图表配置。
- 新增存储项均带版本和容错解析；回滚代码不影响已有自选交易对数据。
- 后端新增只读路由，无数据库迁移；部署回滚时旧前端不调用该路由。

## References

- `klinecharts/preview` 示例：[app.ts](https://github.com/klinecharts/preview/blob/main/src/app.ts)、[package.json](https://github.com/klinecharts/preview/blob/main/package.json)
- 本项目已锁定 `klinecharts` 10.0.3 与 Binance Go SDK 2.8.9；实现时需以 lockfile/go.mod 为准。
