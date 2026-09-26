# custom-trading-pairs Specification

## Purpose

允许用户在常用交易对之外维护自己的 Binance 合约交易对列表，并确保所选交易对在历史行情、分析、交易机会和实时行情中具有一致且可解释的行为。

## Requirements

### Requirement: 管理自选交易对

系统 SHALL 保留预设交易对，并允许用户输入、添加、选择和移除由数据源支持的自选交易对。系统 SHALL 将自选列表保存在当前浏览器，并在重新打开页面后恢复。

#### Scenario: 添加并恢复自选交易对
- **WHEN** 用户输入一个有效且受支持的 Binance 合约交易对并添加它
- **THEN** 该交易对出现在选择列表中，且页面重新加载后仍可选择

#### Scenario: 删除自选交易对
- **WHEN** 用户删除一个自选交易对
- **THEN** 该交易对从自选列表和本地保存的数据中移除，预设交易对仍可选择

#### Scenario: 重复添加
- **WHEN** 用户输入的交易对与预设或已有自选交易对仅大小写或首尾空白不同
- **THEN** 系统 SHALL 不创建重复项，并告知用户该交易对已存在

### Requirement: Reorder and restore the watchlist
The system SHALL let users drag a visible watchlist symbol to a new position. The resulting order SHALL be saved in the current browser and restored after reload. Reordering SHALL not change the selected trading pair. Newly added symbols SHALL be appended to the saved order, and removed symbols SHALL no longer appear in it.

#### Scenario: Move a symbol in the watchlist
- **WHEN** the user drags a symbol before or after another visible symbol
- **THEN** the watchlist displays the new order without changing the selected symbol

#### Scenario: Restore the chosen order
- **WHEN** the user reloads the page after reordering symbols
- **THEN** the watchlist restores the same order

#### Scenario: Add or remove a symbol after reordering
- **WHEN** the user adds a symbol or removes a visible symbol
- **THEN** the new symbol is appended to the watchlist order and a removed symbol is discarded from that order

### Requirement: 验证交易对

系统 SHALL 将输入规范化为交易所使用的交易对标识，校验格式和 Binance 合约数据源中的可交易状态，并将“格式错误”“不受支持”与“数据源暂不可用”区分反馈。只有校验通过的交易对 SHALL 被加入自选列表。

#### Scenario: 无效格式
- **WHEN** 用户输入包含不允许字符或为空的交易对
- **THEN** 系统拒绝添加，并显示格式错误

#### Scenario: 不受支持的交易对
- **WHEN** 交易对格式有效，但 Binance 合约数据源不支持或不再提供该交易对
- **THEN** 系统拒绝添加，并显示不受支持的反馈

#### Scenario: 校验服务不可用
- **WHEN** 交易对校验所需的数据源暂时无法访问
- **THEN** 系统提示稍后重试，且不将其误报为不受支持

### Requirement: 所选交易对一致生效

系统 SHALL 将当前选中的交易对用于 K 线、分析、交易机会和实时行情请求。切换交易对时，系统 SHALL 停止旧交易对的实时订阅，并避免旧请求结果覆盖新交易对的数据。

#### Scenario: 切换至自选交易对
- **WHEN** 用户选择一个已添加的自选交易对
- **THEN** 图表、分析、交易机会和实时行情均针对该交易对更新

#### Scenario: 快速切换交易对
- **WHEN** 用户在旧交易对请求完成前切换到另一个交易对
- **THEN** 旧交易对的响应或推送不会显示为当前交易对的数据

#### Scenario: 已保存交易对失效
- **WHEN** 一个先前保存的交易对不再受数据源支持
- **THEN** 系统明确提示该交易对不可用，并允许用户移除或改选其他交易对
