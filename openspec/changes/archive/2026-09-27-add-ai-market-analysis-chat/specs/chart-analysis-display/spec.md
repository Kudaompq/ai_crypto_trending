## MODIFIED Requirements

### Requirement: 主页面只保留行情展示及必要交互

主页面 SHALL 显示 K 线图、交易对和周期选择及行情状态。桌面布局 SHALL 将 K 线图放在左侧，并将 Watchlist 与 AI 行情分析放在右侧同一面板容器的两个 Tab 中；窄屏布局 SHALL 将 K 线图排在该面板之前。Watchlist SHALL 保留交易对选择、实时报价及已有的列表管理交互。AI 行情分析 SHALL 只在用户主动发送消息后生成对话回复，并遵循 `ai-market-analysis-chat` 能力。主页面 SHALL 不渲染交易建议、趋势、支撑/压力位、形态或市场结构等旧的静态独立分析面板，也 SHALL 不提供交易机会入口、弹窗或后台轮询。用户选取并附着于 K 线图的技术指标副图属于图表内容，不属于已移除的静态分析面板。

#### Scenario: 主页面加载完成
- **WHEN** 当前交易对行情加载完成且用户使用桌面端
- **THEN** 页面左侧显示当前交易对及周期的 K 线图，右侧同一容器提供 Watchlist 和 AI 行情分析两个 Tab，且默认显示 Watchlist

#### Scenario: 切换右侧 Tab
- **WHEN** 用户在 Watchlist 与 AI 行情分析之间切换
- **THEN** 两个 Tab 在同一右侧容器中显示各自内容，Watchlist 行情订阅及 AI 会话内容保持可用，切换 Tab 本身不发起模型请求

#### Scenario: 窄屏页面加载完成
- **WHEN** 用户在窄屏设备打开主页面
- **THEN** 页面先显示 K 线图，随后显示包含 Watchlist 和 AI 行情分析 Tab 的面板

#### Scenario: 主页面显示用户选取的图表指标
- **WHEN** 用户为当前交易对启用了 K 线图技术指标副图
- **THEN** 指标副图附着于 K 线图显示，旧的静态分析面板和交易机会界面仍不显示

#### Scenario: 后台机会监控已移除
- **WHEN** 主页面运行并持续接收行情
- **THEN** 浏览器不显示交易机会入口或弹窗，也不定期请求交易机会数据
