## MODIFIED Requirements

### Requirement: 管理自选交易对

系统 SHALL 在 K 线图左侧的 Watchlist 中展示当前可用的预设及自定义交易对，并 SHALL 将选择、添加和删除操作放在 Watchlist 中。系统 SHALL 将自选列表保存在当前浏览器，并在重新打开页面后恢复。Header SHALL 不再显示独立的交易对下拉选择器。

#### Scenario: 添加并恢复自选交易对
- **WHEN** 用户在 Watchlist 中添加一个有效且受支持的 Binance 合约交易对
- **THEN** 该交易对出现在 Watchlist 中，且页面重新加载后仍可选择

#### Scenario: 删除自选交易对
- **WHEN** 用户从 Watchlist 中删除一个交易对
- **THEN** 该交易对从 Watchlist 和本地保存的数据中移除，其他预设及自选交易对仍可选择

#### Scenario: 不能删除最后一个交易对
- **WHEN** Watchlist 中只剩一个交易对
- **THEN** 系统不允许删除该交易对

#### Scenario: 重复添加
- **WHEN** 用户输入的交易对与预设或已有自选交易对仅大小写或首尾空白不同
- **THEN** 系统 SHALL 不创建重复项，并告知用户该交易对已存在

### Requirement: 所选交易对一致生效

系统 SHALL 将 Watchlist 中当前选中的交易对用于 K 线、当前交易对分析和实时行情请求。切换交易对时，系统 SHALL 停止旧交易对的实时订阅，并避免旧请求或推送覆盖新交易对的数据。

#### Scenario: 切换至自选交易对
- **WHEN** 用户选择一个已添加的预设或自选交易对
- **THEN** 图表、当前交易对分析和实时行情均针对该交易对更新

#### Scenario: 快速切换交易对
- **WHEN** 用户在旧交易对请求完成前切换到另一个交易对
- **THEN** 旧交易对的响应或推送不会显示为当前交易对的数据

#### Scenario: 已保存交易对失效
- **WHEN** 一个先前保存的交易对不再受数据源支持
- **THEN** 系统明确提示该交易对不可用，并允许用户从 Watchlist 中移除或改选其他交易对
