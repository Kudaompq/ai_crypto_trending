## Why

当前 Dashboard 将 Watchlist 放在左侧、K 线放在右侧，没有与当前交易对行情直接关联的对话入口。把图表与 Watchlist / AI 行情分析并列展示，并提供基于当前标的和周期的只读对话，可以让用户在查看行情时追问指标和走势，而不恢复已退役的静态分析面板。

## What Changes

- 调整主页面布局：K 线图位于左侧，右侧使用同一个面板容器，通过 Watchlist 和 AI 行情分析两个 Tab 切换。
- 新增只读的多轮 AI 行情分析对话。每轮对话以当前 USDⓈ-M 交易对、周期、现有指标分析和 K 线数据作为上下文；切换交易对后使用该币种自己的会话，切换周期后继续对话并使用新周期的行情上下文。
- 对话历史仅保存在当前浏览器会话内；不自动发起模型请求，不提供交易或其他写操作。
- 后端新增可配置的 OpenAI-compatible 模型接入，模型凭据只由后端读取，不向浏览器暴露。模型未配置、超时或调用失败时，在对话区域显示可恢复的错误状态。
- 保留 Watchlist 的实时价格、增删和排序，以及已有 K 线、周期切换和行情状态行为；窄屏布局将 K 线排在双 Tab 面板之前。

## Capabilities

### New Capabilities

- `ai-market-analysis-chat`: 为当前交易对和周期提供有行情依据、按币种隔离的只读 AI 多轮分析对话。

### Modified Capabilities

- `chart-analysis-display`: 调整主页面的图表与侧栏布局，并允许用户主动打开 AI 行情对话；原有静态分析面板和交易机会仍保持退役状态。

## Impact

- 前端 `Dashboard.vue`、行情 store、API 封装及 Dashboard 行为测试；图表和 Watchlist 的既有选择、行情流与列表操作需在新布局中保持同步。
- 后端新增对话 handler、行情上下文服务和模型 provider 配置，复用现有交易对校验、K 线及指标分析能力。
- 新增一个对话 API；模型通过后端配置的 OpenAI-compatible HTTP 接口访问，密钥不进入前端配置或响应。
- 新增后端 provider、输入校验、会话隔离及页面布局的行为测试；不新增数据库存储。
