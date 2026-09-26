# Market Kline Chart Specification

## Purpose

让用户查看当前 Watchlist 交易对的交互式 K 线，并通过周期选择和向左翻页浏览行情历史。图表须与所选交易对和周期的实时行情保持一致，并遵守页面移除分析面板后的展示范围。

## Requirements

### Requirement: 显示当前交易对的交互式 K 线

系统 SHALL 显示当前 Watchlist 交易对及所选周期的 OHLCV K 线，并 SHALL 支持拖动和缩放图表查看不同时间范围。图表 SHALL 保留当前 K 线区域已有的价格、涨跌幅、高低价、成交量和 ATR 信息，且 SHALL 不渲染已移除的分析面板或交易机会内容。

#### Scenario: 加载当前交易对
- **WHEN** 用户打开页面或选中 Watchlist 中的交易对
- **THEN** 图表显示该交易对当前周期的历史 K 线和行情信息

#### Scenario: 查看图表其他时间范围
- **WHEN** 用户拖动或缩放图表
- **THEN** K 线、时间刻度和价格刻度对应更新后的可视范围

### Requirement: 向前分页加载更早的 K 线

系统 SHALL 在用户将图表拖动到当前历史数据左边界且仍有更早数据时，按时间边界请求下一页历史 K 线。分页结果 SHALL 按时间升序合并，不得重复已显示的时间点；分页响应 SHALL 明确报告是否还有更早数据，到达最早可用数据后 SHALL 停止该方向的请求。分页失败时 SHALL 保留已加载数据并提供可恢复的失败反馈。

#### Scenario: 加载更早历史
- **WHEN** 用户滚动到已加载 K 线的最左侧且该方向仍有数据
- **THEN** 系统请求游标之前的一页 K 线，并将其追加到图表历史数据前方

#### Scenario: 历史页边界无重复
- **WHEN** 相邻两页在时间边界相邻或上游返回包含边界时间点
- **THEN** 图表按时间升序显示合并后的 K 线，且同一开盘时间最多显示一次

#### Scenario: 已无更早数据
- **WHEN** 历史接口报告没有更早数据
- **THEN** 图表将该方向标记为已到达历史边界，继续向左拖动时不重复请求

#### Scenario: 历史分页请求失败
- **WHEN** 请求更早 K 线失败
- **THEN** 已显示 K 线保持不变，系统允许用户再次触发加载

### Requirement: 在图表上方选择 Binance 原生 K 线周期

系统 SHALL 将周期选择控件显示在 K 线绘图区上方，并允许用户从 Binance USDⓈ-M 原生周期中选择。周期选项 SHALL 包含 `1m`、`3m`、`5m`、`15m`、`30m`、`1h`、`2h`、`4h`、`6h`、`8h`、`12h`、`1d`、`3d`、`1w` 和 `1M`。系统 SHALL 将各选项映射到 K 线图表支持的周期类型与跨度，不提供任意周期输入或 K 线聚合。首次进入时 SHALL 沿用当前默认周期。

#### Scenario: 周期控件位置
- **WHEN** 页面显示 K 线图
- **THEN** 周期选择控件位于 K 线绘图区上方

#### Scenario: 选择 Binance 原生周期
- **WHEN** 用户选择一个 Binance 原生周期
- **THEN** 图表加载当前交易对该周期的历史 K 线，并使用该周期的实时行情

#### Scenario: 请求未支持周期
- **WHEN** 客户端请求 Binance USDⓈ-M 不支持的周期
- **THEN** 服务端返回可识别的参数错误，页面保留当前有效周期并提示该周期不可用

### Requirement: 交易对或周期切换时隔离图表数据

系统 SHALL 在交易对或周期变化时停止旧订阅，并为新的交易对和周期重新加载历史及实时 K 线。旧请求或旧订阅在切换后返回的数据 SHALL NOT 覆盖当前图表。

#### Scenario: 切换交易对或周期
- **WHEN** 用户更改 Watchlist 交易对或图表周期
- **THEN** 图表显示新选择对应的数据，且旧实时订阅停止

#### Scenario: 切换后收到过期结果
- **WHEN** 旧交易对或周期的历史响应或实时事件在切换后到达
- **THEN** 图表忽略该结果且不将其显示为当前行情

### Requirement: 实时与备用行情更新图表

系统 SHALL 将与当前交易对和周期匹配的有效实时 K 线更新到图表，更新当前蜡烛或追加新蜡烛。实时推送不可用时，系统 SHALL 遵守既有行情状态和备用更新行为，且 SHALL 不将旧数据标记为实时。

#### Scenario: 收到当前周期的实时 K 线
- **WHEN** 系统收到当前交易对及周期的有效实时 K 线
- **THEN** 图表更新对应蜡烛并记录最近行情时间

#### Scenario: 实时推送中断
- **WHEN** 当前周期的实时推送中断
- **THEN** 图表沿用既有备用行情行为，并明确显示非实时状态

### Requirement: Configure chart studies per trading pair
The system SHALL let users enable and disable moving average (MA), exponential moving average (EMA), and Bollinger Band studies on the price pane, and MACD on a secondary chart pane. Users SHALL be able to set each study's applicable parameters. The enabled studies and their parameters SHALL be shared by all trading pairs in the current browser and SHALL be reused across all chart intervals.

#### Scenario: Select and configure price-pane studies
- **WHEN** the user enables MA, EMA, or Bollinger Bands and sets valid parameters
- **THEN** the selected studies are drawn on the price pane using those parameters

#### Scenario: Select and configure a secondary study
- **WHEN** the user enables MACD and configures its periods
- **THEN** MACD appears in a secondary pane below the price chart

#### Scenario: Share studies across trading pairs
- **WHEN** the user changes the selected studies or parameters for one trading pair and switches to another pair
- **THEN** the same studies and parameters remain enabled for the other pair

#### Scenario: Keep study settings across intervals
- **WHEN** the user changes the chart interval
- **THEN** the same studies and parameters remain enabled

#### Scenario: Migrate existing study settings
- **WHEN** the browser contains saved per-pair study settings from the previous version
- **THEN** the settings for the currently selected pair are used as the shared settings when present, otherwise the first saved study settings are used, while drawings remain associated with their original trading pairs

#### Scenario: Reject invalid study parameters
- **WHEN** the user enters an invalid parameter combination, such as a non-positive period or a MACD fast period that is not less than its slow period
- **THEN** the system SHALL reject the invalid settings and SHALL not apply them to the chart

### Requirement: Create and manage common chart drawings
The system SHALL provide common drawing tools including a horizontal line, trend line, ray, parallel channel, and Fibonacci retracement. Users SHALL be able to create, adjust, and remove drawings. Drawings SHALL be saved separately for each trading pair in the current browser and SHALL remain associated with their price and time coordinates across chart interval changes.

#### Scenario: Draw and adjust an object
- **WHEN** the user selects a drawing tool, places an object on the chart, and adjusts one of its anchors
- **THEN** the chart displays the object at the selected coordinates

#### Scenario: Restore drawings after reload
- **WHEN** the user reloads the page with a symbol selected
- **THEN** the chart restores that symbol's saved drawings at their original price and time coordinates

#### Scenario: Keep drawings isolated by symbol
- **WHEN** the user switches to a different trading pair
- **THEN** drawings belonging to the previous pair are not displayed on the current pair's chart

#### Scenario: Keep drawings across intervals
- **WHEN** the user changes the chart interval and then returns to a time range containing a saved drawing anchor
- **THEN** the drawing remains attached to the same price and time coordinates

#### Scenario: Remove a drawing
- **WHEN** the user removes a saved drawing from the chart
- **THEN** that drawing is removed from the current pair's saved drawings and stays absent after reload
