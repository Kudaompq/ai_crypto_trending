# ETH K线量化分析系统 - 快速开始

## 🚀 启动指南

### 1. 启动后端服务器

打开终端 1:
```bash
cd /Users/kudaompq/Desktop/crypto/ai_trending/backend
go run cmd/server/main.go
```

✅ 后端将运行在: `http://localhost:8080`

### 2. 启动前端开发服务器

打开终端 2:
```bash
cd /Users/kudaompq/Desktop/crypto/ai_trending/frontend
npm run dev
```

✅ 前端将运行在: `http://localhost:5173`

### 3. 访问应用

在浏览器中打开: `http://localhost:5173`

---

## 📊 功能说明

### 主要功能
1. **K线图表**: 实时显示 ETH 价格走势
2. **趋势分析**: 判断市场趋势方向和强度
3. **技术指标**: MACD, KDJ, RSI 多维度分析
4. **支撑/压力位**: 自动识别关键价格区间
5. **蜡烛图形态**: 识别18+种经典形态
6. **市场结构**: 分析前高前低和风险等级

### 操作步骤
1. 选择时间周期 (1h / 4h / 1d)
2. 点击"刷新数据"按钮
3. 查看各项分析结果

---

## 🛠️ 技术栈

**后端**: Go + Gin + TA-Lib + Binance API
**前端**: Vue3 + TypeScript + TradingView Charts + Element Plus

---

## 📝 API 端点

- `GET /api/health` - 健康检查
- `GET /api/kline` - 获取K线数据
- `GET /api/analysis` - 获取综合分析

---

## ✅ 已完成功能

- [x] 后端 API 服务
- [x] 技术指标计算 (MACD, KDJ, RSI)
- [x] 蜡烛图形态识别 (18+种)
- [x] 支撑/压力位检测
- [x] 趋势分析引擎
- [x] 市场结构分析
- [x] Vue3 前端界面
- [x] TradingView 图表集成
- [x] 响应式暗黑主题

---

## 📖 详细文档

查看完整文档:
- 后端实现: `walkthrough.md`
- 前端实现: `frontend_walkthrough.md`
- 蜡烛图形态: `candlestick_patterns_knowledge.md`
- 实现计划: `implementation_plan.md`
