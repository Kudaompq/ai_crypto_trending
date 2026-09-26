## MODIFIED Requirements

### Requirement: 管理自选交易对

系统 SHALL 保留预设交易对，并允许用户输入、添加、选择和移除由数据源支持的自选交易对。系统 SHALL 将单个有序 Watchlist 持久化到服务端数据库，并让访问该服务的所有客户端读取和修改同一份列表。数据库初始化时 SHALL 仅从首个完成迁移的客户端导入一次旧版浏览器 Watchlist；后续客户端 SHALL 以数据库列表为准。保存失败时 SHALL 明确提示未能同步，且不得将未保存的修改当作已持久化。

#### Scenario: 添加并恢复自选交易对
- **WHEN** 用户输入一个有效且受支持的 Binance 合约交易对并添加它
- **THEN** 该交易对出现在共享 Watchlist 中，且页面重新加载或服务重启后仍可选择

#### Scenario: 删除自选交易对
- **WHEN** 用户删除一个自选交易对
- **THEN** 该交易对从共享 Watchlist 移除，预设交易对仍可重新添加

#### Scenario: 数据库首次启用时迁移本地列表
- **WHEN** 新数据库收到旧版浏览器 Watchlist 的首次迁移请求
- **THEN** 数据库保存该客户端当前可见的有序列表；之后的迁移请求不覆盖共享列表

#### Scenario: 数据库写入失败
- **WHEN** 添加或删除操作未能保存到服务端数据库
- **THEN** 系统告知用户列表未能同步，且未保存的操作不被显示为已提交

#### Scenario: 重复添加
- **WHEN** 用户输入的交易对与共享 Watchlist 中现有交易对仅大小写或首尾空白不同
- **THEN** 系统 SHALL 不创建重复项，并告知用户该交易对已存在

### Requirement: Reorder and restore the watchlist

The system SHALL let users drag a visible watchlist symbol to a new position. The resulting order SHALL be saved in the shared server database and restored for all clients. Reordering SHALL not change the selected trading pair. Newly added symbols SHALL be appended to the saved order, and removed symbols SHALL no longer appear in it.

#### Scenario: Move a symbol in the watchlist
- **WHEN** the user drags a symbol before or after another visible symbol
- **THEN** the watchlist displays the new order without changing the selected symbol

#### Scenario: Restore the chosen order
- **WHEN** the user reloads the page after reordering symbols
- **THEN** the watchlist restores the same database-backed order

#### Scenario: Add or remove a symbol after reordering
- **WHEN** the user adds a symbol or removes a visible symbol
- **THEN** the new symbol is appended to the shared watchlist order and a removed symbol is discarded from that order
