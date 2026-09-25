# ETH K线量化分析系统

完整的 ETH K线分析系统，包含后端 API 和前端可视化界面。

## 功能特性

### 技术指标
- **MACD**: DIF, DEA, Histogram
- **KDJ**: K, D, J 值
- **RSI**: 6周期 & 14周期

### 趋势分析
- 综合多指标判断趋势方向（上升/下降/盘整）
- 趋势强度评分（0-1）
- 趋势反转概率计算

### 蜡烛图形态识别
基于《日本蜡烛图技术》，识别18+种经典形态：
- **单根形态**: 锤子线、上吊线、倒锤子、射击之星、十字星等
- **双根形态**: 看涨/看跌吞没、刺透形态、乌云盖顶、孕线等
- **三根形态**: 启明星、黄昏星、三只乌鸦、三个白兵等

### 支撑/压力位
- 基于价格聚类算法
- 成交量加权
- 强度评分

### 市场结构分析
- 识别前高前低（Higher High, Higher Low）
- 结构破位检测
- 风险等级评估

## 项目结构

```
ai_trending/
├── backend/                    # Golang 后端
│   ├── cmd/
│   │   └── server/
│   │       └── main.go         # 服务器入口
│   ├── internal/
│   │   ├── handler/            # HTTP 处理器
│   │   ├── service/            # 业务逻辑
│   │   ├── repository/         # 数据访问
│   │   ├── indicator/          # 技术指标
│   │   └── model/              # 数据模型
│   └── go.mod
└── frontend/                   # Vue3 前端
    ├── src/
    │   ├── views/              # 页面组件
    │   ├── components/         # 通用组件
    │   ├── stores/             # 状态管理
    │   └── api/                # API 接口
    └── package.json
```

## 快速开始

在项目根目录运行本地开发脚本（需要 Go 1.23+、Node.js 和 npm）：

```bash
./start.sh
```

首次运行会安装前端依赖。访问 `http://localhost:5173`，按 Ctrl+C 停止前后端服务。
如果 8080 端口已被占用，可以运行 `PORT=18080 ./start.sh`。

也可以分别启动：

### 1. 启动后端

```bash
cd backend
go mod download
go run cmd/server/main.go
```

服务器将在 `http://localhost:8080` 启动

### 2. 启动前端

```bash
cd frontend
npm install
npm run dev
```

前端将在 `http://localhost:5173` 启动

### 3. 访问应用

在浏览器中打开: `http://localhost:5173`

### 自选交易对与行情状态

点击交易对下拉框旁的 `+`，输入 Binance 合约交易对（例如 `AVAXUSDT`）并添加。
输入会自动转为大写，服务端会检查该合约是否可交易。自选列表保存在当前浏览器；
选中自选交易对后可点击 `−` 删除。

页面将行情标为“正在连接实时行情”“实时更新”“备用更新（非实时）”或
“行情不可用或已过期”。收到第一条价格事件后才显示实时更新。实时事件中断约
15 秒后，页面每约 15 秒通过 REST 获取备用行情；恢复推送后自动回到实时更新。
如果价格不动，先看状态和“最近行情”时间：备用更新说明 Binance WebSocket
可能被当前网络或代理阻断；行情不可用说明备用 REST 请求也未成功。

## API 端点

#### 校验交易对
```bash
GET /api/symbols/validate?symbol=AVAXUSDT
```

成功时返回规范化后的交易对；格式错误返回 400，不可交易返回 404，
交易所校验暂不可用返回 503。

#### 1. 健康检查
```bash
GET /api/health
```

#### 2. 获取K线数据
```bash
GET /api/kline?symbol=ETHUSDT&interval=1d&limit=100
```

参数:
- `symbol`: 交易对（默认: ETHUSDT）
- `interval`: 时间周期（1m, 5m, 15m, 1h, 4h, 1d）
- `limit`: 数据条数（默认: 100, 最大: 500）

#### 3. 获取综合分析
```bash
GET /api/analysis?symbol=ETHUSDT&interval=1d&limit=100
```

返回完整的分析结果，包括：
- 趋势分析
- 技术指标（MACD, KDJ, RSI）
- 支撑/压力位
- 蜡烛图形态
- 市场结构

#### 4. 实时 K 线流
```bash
GET /api/stream?symbol=ETHUSDT&interval=1m
```

该接口使用 Server-Sent Events 持续推送 Binance 合约 K 线。连接建立后先发送
`ready` 事件；收到上游价格后发送 `kline` 事件，上游连接失败或重试时发送
`status` 事件。`ready` 只表示浏览器已连到后端，不表示上游价格已开始推送。
同一交易对和周期的浏览器连接共享一条 Binance WebSocket，并在上游断开后自动重连。

### 示例响应

```json
{
  "symbol": "ETHUSDT",
  "interval": "1d",
  "timestamp": 1700000000000,
  "trend": {
    "direction": "上升",
    "strength": 0.75,
    "change_probability": 0.25
  },
  "indicators": {
    "macd": {
      "dif": 12.5,
      "dea": 10.2,
      "histogram": 4.6
    },
    "kdj": {
      "k": 75.3,
      "d": 68.9,
      "j": 88.1
    },
    "rsi": {
      "rsi6": 68.5,
      "rsi14": 62.3
    }
  },
  "sr_levels": {
    "resistance": [
      {"price": 2100, "strength": 0.9}
    ],
    "support": [
      {"price": 2000, "strength": 0.85}
    ]
  },
  "candlestick_patterns": [
    {
      "pattern": "看涨吞没",
      "type": "反转",
      "direction": "看涨",
      "position": -1,
      "reliability": 0.9,
      "description": "大阳线吞没前阴线，强烈看涨信号"
    }
  ],
  "market_structure": {
    "higher_high": true,
    "higher_low": true,
    "structure_break": false,
    "risk_level": "低"
  }
}
```

## 技术栈

### 后端
- **语言**: Go 1.21+
- **框架**: Gin
- **数据源**: Binance API
- **技术指标**: TA-Lib

### 前端
- **框架**: Vue 3 + TypeScript
- **构建工具**: Vite
- **图表**: TradingView Lightweight Charts
- **UI**: Element Plus

## 开发计划

- [x] Phase 0: 学习日本蜡烛图技术
- [x] Phase 1: 后端基础架构
- [x] Phase 2: 前端开发
- [ ] Phase 3: 优化与测试

## 许可证

MIT
