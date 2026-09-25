## Context

参见 [proposal.md](./proposal.md) 与本变更的 specs。当前交易对列表由 Pinia store 管理并保存在浏览器；`Dashboard.vue` 维护当前选中币种和一个 K 线 SSE。历史 K 线由 REST 提供，`MarketStreamService` 按交易对和周期共享 Binance K 线 WebSocket。`SimpleChart.vue` 是基于 DOM 的手工蜡烛绘制。

Watchlist 需要同时展示多个币种的价格，但现有实时流仅服务当前选中交易对。为了避免浏览器为每个币种单独建立连接，价格流需由后端汇总并按各页面的自选集合筛选；当前图表的 K 线流则继续独立运行。

## Goals / Non-Goals

**Goals:**

- 让 Watchlist 一次性初始化所有交易对的最新价，并持续接收后续价格变化。
- 后端复用单一 USDⓈ-M 全市场价格上游流，对每个浏览器订阅只转发其请求的币种。
- 在实时流断开期间保留已知价并标示为过期；恢复后继续推送。
- 用 Lightweight Charts 绘制当前选中交易对，同时沿用当前 REST、K 线 SSE 和周期控件。
- 保留当前交易对选择、验证、增删和本地存储行为，并将这些操作移进 Watchlist。

**Non-Goals:**

- 不提供 Advanced Charts 的绘图工具栏，也不新增趋势线或其他绘图编辑器。
- 不为 Watchlist 行增加 24 小时涨跌幅或成交量；只显示实时最新价。
- 不改变 Binance 合约校验规则、现有 K 线 SSE 协议或分析计算。

## Decisions

1. **用 Lightweight Charts 仅替换 K 线绘制层。** `Dashboard.vue` 继续负责周期和行情协调；新图表组件在历史数据到达时设置蜡烛数据，在实时 SSE 到达时更新当前蜡烛或追加新蜡烛。将毫秒时间戳转换为图表需要的时间单位，并响应容器尺寸变化。保留现有价格、涨跌、高低价、成交量和 ATR 摘要。
   - 备选方案：保留手写 DOM 图表。该方案继续维护价格坐标、拖动和缩放等底层交互，无法满足选择 Lightweight Charts 的目标。
   - 周期按钮继续由应用显示；Lightweight Charts 本身不提供 Advanced Charts 的周期工具栏。

2. **在应用内实现 Watchlist，替换 header 币种下拉。** 列表从现有 `availableSymbols` 读取；选中行调用现有 symbol 切换路径。将添加表单和移除操作放在列表中，继续调用现有 Binance 校验和 store 本地持久化。选中项删除时沿用 store 选择另一个可用交易对的行为。窄屏下列表改为图表上方布局。
   - 备选方案：同时保留 header 下拉。两个入口需要持续同步，并造成重复控件；用户已明确要求移除 header 下拉。

3. **通过 Binance USDⓈ-M 全市场 mini ticker 作为 Watchlist 上游价格流。** 后端以一条 `/market/ws/!miniTicker@arr` 连接接收全市场有变化的 mini ticker 事件，从 `c` 取最新价，并按事件中的合约类型标识过滤为 USDⓈ-M（`st = 1`）。Binance 文档说明该流每秒更新且只包含已变化的交易对；因此后端需将数组中的每项独立解析，不能假设每条消息只包含一个币种。
   - 备选方案：为每个币种创建一个 K 线/mini ticker WebSocket。连接数会随用户自选数量和并发页面数增长，且 K 线周期流不适合作为统一 Watchlist 报价源。
   - Binance USDⓈ-M 当前市场流说明：[`!miniTicker@arr` 数据结构、频率和路由](https://developers.binance.info/en/docs/catalog/core-trading-derivatives-trading-usd-s-m-futures/api/ws-streams/market)、[市场流连接和 `/market` 路由](https://developers.binance.com/en/docs/products/derivatives-trading-usds-futures/websocket-market-streams/Connect)。

4. **为价格初始值增加批量快照接口，实时变化走 SSE。** 页面将当前 Watchlist 交易对作为集合请求初始价格快照，并打开多币种 SSE；SSE 服务只向该连接转发集合内的价格。价格快照使用 USDⓈ-M 最新价格 REST 数据，避免等待某交易对下一次发生变化后才显示价格。事件携带交易对与时间戳，前端仅接受比该交易对当前缓存更新的价格，以免较晚返回的快照覆盖实时推送。增删交易对时更新该页面的 SSE 关注集合；页面卸载时关闭连接并释放服务端订阅。
   - 备选方案：让每个浏览器直接连接 Binance 或为每行开一个 EventSource。前者将上游与行情协议暴露在前端且绕过现有后端验证；后者产生随币种数线性增长的浏览器连接。
   - 最新价快照使用 Binance USDⓈ-M [`GET /fapi/v1/ticker/price`](https://developers.binance.info/en/docs/catalog/core-trading-derivatives-trading-usd-s-m-futures/api/rest-api/market-data)。

5. **把多币种报价连接状态与当前图表的 K 线状态分开。** Watchlist 上方提供价格流状态；价格流重连时保留缓存价并标示延迟，K 线图继续使用既有的实时/备用/不可用状态。收到 SSE `ready` 不等于价格流已收到有效市场数据；服务端应在收到可解析的 USDⓈ-M mini ticker 后才宣布价格源进入实时状态。因为 Binance 全市场 mini ticker 只推送发生变化的交易对，不能仅凭单个交易对长时间没有变化将其价格判为过期。
   - 备选方案：将 Watchlist 行情并入当前选中交易对的状态。未选中交易对的断流将无法被检测，且用户会看到误导性的整页状态。

6. **遵循 TDD 验证前后端行为。** 前端测试覆盖 Watchlist 显示全部币种、选中切换图表、增删操作、价格按交易对更新、断线标示与恢复；后端测试覆盖快照过滤、上游数组解析、USDⓈ-M 类型过滤、按客户端自选集合转发、无效交易对拒绝、断开后清理和重连。先看到相应行为测试失败，再实现功能。
   - 备选方案：只验证构建和页面可见。它们不能发现币种错配、掉订阅或价格快照覆盖新推送等问题。

## Risks / Trade-offs

- 全市场流可能包含大量更新 → 由单一后端连接接入，在后端过滤到各浏览器声明的 Watchlist 后再写入 SSE；不把全市场消息广播到浏览器。
- 全市场流在 2026 年合约市场迁移后可能包含 USDⓈ-M 与 COIN-M 标记 → 按 Binance 的 `st` 字段仅接受 USDⓈ-M，并用测试覆盖未知标记与异常帧。
- mini ticker 数组只列出发生变化的币种 → 使用 REST 快照填充初值；后续只更新对应币种，不能用“本轮未收到该币种”覆盖其现价。
- 快照与 SSE 可能乱序到达 → 每个价格带交易对及事件时间；较旧数据不得覆盖更新的缓存值。
- SSE 连接可能打开但行情上游静默或中断 → 服务端基于有效上游消息检测静默并重连，推送 reconnecting 状态；页面保留价格但明确标示非实时。
- Lightweight Charts 依照项目许可证要求需显示 TradingView 归属信息 → 在图表区域保留可见的归属链接。
- 图表容器在侧栏布局调整时可能尺寸变化 → 使用 ResizeObserver 或等效方式调用图表尺寸更新，并在组件卸载时清理图表和观察器。

## Migration Plan

1. 后端先部署价格快照与 SSE 接口；新增接口不影响当前版本前端。
2. 前端部署 Watchlist、价格订阅和 Lightweight Charts 适配，并保留现有浏览器交易对存储格式。
3. 在确认新图表及所有 Watchlist 报价通过验收后移除旧 `SimpleChart` 实现和对应测试。
4. 回滚时先部署不请求新价格接口的前端版本，再回滚后端；无需数据库迁移或修改已保存的自选列表。
