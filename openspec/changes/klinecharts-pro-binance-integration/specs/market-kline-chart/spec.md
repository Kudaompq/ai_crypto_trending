## MODIFIED Requirements

### Requirement: 显示当前交易对的交互式 K 线

系统 SHALL 显示当前 Watchlist 交易对及所选周期的 OHLCV K 线，并 SHALL 支持拖动和缩放图表查看不同时间范围。仅现有右侧图表区域 SHALL 替换为集成图表控件，以承载周期、指标和绘图入口。左侧 Watchlist、页面主 header、全局行情状态及右侧图表区域以外的 header SHALL 保持原有布局和行为。右侧图表区域 SHALL NOT 渲染应用自定义的图表标题及摘要栏，也 SHALL NOT 在这些栏位展示当前价、相邻 K 线涨跌幅、高低价、加载区间成交量或 ATR。Watchlist 报价 SHALL 保留。图表 SHALL 不渲染已移除的分析面板或交易机会内容。

#### Scenario: 加载当前交易对
- **WHEN** 用户打开页面或选中 Watchlist 中的交易对
- **THEN** 图表显示该交易对当前周期的历史 OHLCV K 线和集成图表控件

#### Scenario: 查看图表其他时间范围
- **WHEN** 用户拖动或缩放图表
- **THEN** K 线、时间刻度和价格刻度对应更新后的可视范围

#### Scenario: 移除应用自定义行情摘要
- **WHEN** 页面显示 K 线图
- **THEN** 图表不显示独立的自定义标题或行情摘要栏，Watchlist 报价和全局行情状态仍可见

#### Scenario: 保留图表区域以外的页面外壳
- **WHEN** 集成图表控件显示在右侧图表区域
- **THEN** 左侧 Watchlist、页面主 header、全局行情状态及其他区域的 header 保持原有布局和行为

### Requirement: 在图表上方选择 Binance 原生 K 线周期

系统 SHALL 在图表上方的集成周期栏中允许用户从 Binance USDⓈ-M 原生周期中选择。周期选项 SHALL 包含 `1m`、`3m`、`5m`、`15m`、`30m`、`1h`、`2h`、`4h`、`6h`、`8h`、`12h`、`1d`、`3d`、`1w` 和 `1M`。系统 SHALL 将各选项映射到 K 线图表支持的周期类型与跨度，不提供任意周期输入或 K 线聚合。首次进入时 SHALL 沿用当前默认周期。

#### Scenario: 周期控件位置
- **WHEN** 页面显示 K 线图
- **THEN** 集成周期栏位于 K 线绘图区上方并显示当前周期

#### Scenario: 选择 Binance 原生周期
- **WHEN** 用户在集成周期栏中选择一个 Binance 原生周期
- **THEN** 图表加载当前交易对该周期的历史 K 线，并使用该周期的实时行情

#### Scenario: 请求未支持周期
- **WHEN** 客户端请求 Binance USDⓈ-M 不支持的周期
- **THEN** 服务端返回可识别的参数错误，页面保留当前有效周期并提示该周期不可用

### Requirement: Create and manage common chart drawings

系统 SHALL 在图表集成控件中提供指标入口和分类绘图工具。指标入口 SHALL 允许用户启用、关闭并配置可用指标，且 SHALL 遵守图表研究指标的共享规则。绘图入口 SHALL 提供直线/射线/线段、通道、形状、斐波那契和波浪类别，并至少包含水平线、趋势线、射线、平行通道和斐波那契回撤。用户选择具体绘图工具后 SHALL 能在图表上放置、调整、选择、删除、隐藏或锁定对象；吸附选项 SHALL 控制绘图锚点对齐行为。绘图对象 SHALL 按交易对分别保存，并跨周期恢复到相同价格和时间坐标。

#### Scenario: Draw and adjust an object
- **WHEN** the user selects a drawing tool, places an object on the chart, and adjusts one of its anchors
- **THEN** the chart displays the object at the selected coordinates

#### Scenario: Select a tool from a drawing category
- **WHEN** 用户展开绘图类别并选择一个具体工具
- **THEN** 图表进入该工具的放置交互，用户完成所需锚点后图表显示绘图对象

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

#### Scenario: Manage a selected drawing
- **WHEN** the user selects an existing drawing and applies adjust, hide, lock, or delete
- **THEN** the chart applies that action, and a deleted drawing does not reappear when the current pair is restored
