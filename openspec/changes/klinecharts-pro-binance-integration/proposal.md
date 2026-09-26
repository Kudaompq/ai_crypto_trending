## Why

当前图表把周期选择、行情摘要、指标设置和绘图工具分散在自定义栏位中，交互与 `klinecharts/preview` 的一体化图表体验不同。接入 KLineChart Pro 并通过 Binance 自定义数据适配，可以统一周期、指标和绘图入口，同时保留项目的 USDⓈ-M 行情边界；此外，当前 OI 副图绘制的是名义价值，用户需要的是未平仓持仓量。

## What Changes

- **BREAKING（右侧图表界面）**：仅在现有右侧图表区域接入 KLineChart Pro，用 Pro 的周期栏、指标入口和绘图栏替换图表自定义的 `interval-buttons`、`chart-header`、`chart-summary` 和 `chart-toolbar`。移除图表自定义摘要中的当前价/相邻 K 线涨跌幅、高低价、加载区间成交量和 ATR。左侧 Watchlist（布局、标题、报价和交互）、页面主 header、全局行情状态及其他区域的 header 均保持不变。
- 实现 Binance USDⓈ-M 自定义 Datafeed，提供受支持交易对搜索、15 个 Binance 原生周期映射、历史 K 线获取，以及与现有实时行情链路协调的订阅和取消订阅。不得使用 Pro 默认的 Polygon Datafeed。
- 保留历史向前分页、交易对/周期切换隔离、实时与备用更新状态、研究指标设置和图表偏好；Pro 绘图对象继续按交易对保存并跨周期恢复。
- 将 OI 副图从 `sumOpenInterestValue` 改为 `sumOpenInterest` 数量字段；无 Binance 样本或不支持的周期继续留空，不插值或用持仓价值替代。
- 验证 Pro 与项目现有 KLineCharts 版本的兼容性，并处理 Core 绘图偏好到 Pro 绘图状态的恢复兼容。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `market-kline-chart`：以 Pro 栏位承载周期、指标和绘图交互；移除图表自定义标题和摘要；保持 Binance 周期、历史分页、实时状态、研究偏好及绘图持久化行为。
- `open-interest-chart`：明确副图展示 Binance `sumOpenInterest` 未平仓持仓量，而非 `sumOpenInterestValue` 名义价值。

## Impact

- 前端 `Dashboard.vue`、`SimpleChart.vue`、Binance K 线历史与 SSE 适配、图表偏好和 OI 映射。
- 前端依赖与 lockfile：`@klinecharts/pro` 及其要求的 `klinecharts` 版本组合，需要通过兼容性验证确定。
- 不新增行情供应商或后端数据 API；现有 Binance K 线、实时行情及 OI API 作为数据来源。
