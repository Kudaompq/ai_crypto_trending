<template>
  <div class="trading-recommendation">
    <div class="recommendation-header">
      <h3>📋 交易建议</h3>
    </div>

    <div class="recommendation-content">
      <!-- Main Recommendation -->
      <div class="main-recommendation" :class="recommendationClass">
        <div class="recommendation-badge">
          <span class="recommendation-icon">{{ recommendationIcon }}</span>
          <span class="recommendation-text">{{ recommendation }}</span>
        </div>
        <div class="confidence-level">
          <span class="confidence-label">信心指数:</span>
          <div class="confidence-bar">
            <div 
              class="confidence-fill" 
              :style="{ width: `${Math.abs(confidenceScore)}%` }"
              :class="confidenceClass"
            ></div>
          </div>
          <span class="confidence-value">{{ Math.abs(confidenceScore).toFixed(0) }}%</span>
        </div>
      </div>

      <!-- Analysis Details -->
      <div class="analysis-details">
        <h4>📊 分析依据</h4>
        <ul class="reason-list">
          <li v-for="(reason, index) in reasons" :key="index" class="reason-item">
            <span class="reason-icon" :class="reason.type">{{ reason.icon }}</span>
            <span class="reason-text">{{ reason.text }}</span>
          </li>
        </ul>
      </div>

      <!-- Risk Warning -->
      <div class="risk-warning" v-if="riskWarnings.length > 0">
        <h4>⚠️ 风险提示</h4>
        <ul class="warning-list">
          <li v-for="(warning, index) in riskWarnings" :key="index">
            {{ warning }}
          </li>
        </ul>
      </div>

      <!-- Disclaimer -->
      <div class="disclaimer">
        <p>⚠️ 本建议仅供参考，不构成投资建议。请根据自身风险承受能力谨慎决策。</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AnalysisResult } from '../services/api'

const props = defineProps<{
  analysis: AnalysisResult
}>()

// Calculate recommendation score based on all indicators
const analysisScore = computed(() => {
  let score = 0
  const reasons: Array<{ text: string; icon: string; type: string }> = []
  const warnings: string[] = []

  // 1. Trend Direction (30% weight)
  if (props.analysis.trend.direction === '上升') {
    score += 30
    reasons.push({ text: `趋势方向: ${props.analysis.trend.direction} (强度: ${props.analysis.trend.strength.toFixed(1)})`, icon: '📈', type: 'bullish' })
  } else if (props.analysis.trend.direction === '下降') {
    score -= 30
    reasons.push({ text: `趋势方向: ${props.analysis.trend.direction} (强度: ${props.analysis.trend.strength.toFixed(1)})`, icon: '📉', type: 'bearish' })
  } else {
    reasons.push({ text: `趋势方向: ${props.analysis.trend.direction}`, icon: '➡️', type: 'neutral' })
  }

  // 2. MACD (20% weight)
  const macd = props.analysis.indicators.macd
  if (macd.dif > macd.dea && macd.dif > 0) {
    score += 20
    reasons.push({ text: 'MACD金叉且为正值，多头信号', icon: '✅', type: 'bullish' })
  } else if (macd.dif < macd.dea && macd.dif < 0) {
    score -= 20
    reasons.push({ text: 'MACD死叉且为负值，空头信号', icon: '❌', type: 'bearish' })
  }

  // 3. KDJ (15% weight)
  const kdj = props.analysis.indicators.kdj
  if (kdj.k > kdj.d && kdj.k < 20) {
    score += 15
    reasons.push({ text: `KDJ超卖区域 (K=${kdj.k.toFixed(1)})，可能反弹`, icon: '🔵', type: 'bullish' })
  } else if (kdj.k < kdj.d && kdj.k > 80) {
    score -= 15
    reasons.push({ text: `KDJ超买区域 (K=${kdj.k.toFixed(1)})，可能回调`, icon: '🔴', type: 'bearish' })
    warnings.push('KDJ处于超买区域，注意回调风险')
  }

  // 4. RSI (15% weight)
  const rsi = props.analysis.indicators.rsi.rsi14
  if (rsi < 30) {
    score += 15
    reasons.push({ text: `RSI超卖 (${rsi.toFixed(1)})，可能反弹`, icon: '🔵', type: 'bullish' })
  } else if (rsi > 70) {
    score -= 15
    reasons.push({ text: `RSI超买 (${rsi.toFixed(1)})，可能回调`, icon: '🔴', type: 'bearish' })
    warnings.push('RSI处于超买区域，注意回调风险')
  } else {
    reasons.push({ text: `RSI中性 (${rsi.toFixed(1)})`, icon: '⚪', type: 'neutral' })
  }

  // 5. Candlestick Patterns (10% weight)
  const patterns = props.analysis.candlestick_patterns
  const bullishPatterns = patterns.filter(p => p.type === 'bullish').length
  const bearishPatterns = patterns.filter(p => p.type === 'bearish').length
  
  if (bullishPatterns > bearishPatterns) {
    score += 10
    reasons.push({ text: `检测到${bullishPatterns}个看涨形态`, icon: '🕯️', type: 'bullish' })
  } else if (bearishPatterns > bullishPatterns) {
    score -= 10
    reasons.push({ text: `检测到${bearishPatterns}个看跌形态`, icon: '🕯️', type: 'bearish' })
  }

  // 6. Market Structure (10% weight)
  const structure = props.analysis.market_structure
  if (structure.risk_level === '低风险') {
    score += 5
  } else if (structure.risk_level === '高风险') {
    score -= 5
    warnings.push(`市场结构显示${structure.risk_level}`)
  }

  return { score, reasons, warnings }
})

const recommendation = computed(() => {
  const score = analysisScore.value.score
  if (score > 50) return '强烈做多'
  if (score > 20) return '做多'
  if (score > -20) return '观望'
  if (score > -50) return '做空'
  return '强烈做空'
})

const recommendationIcon = computed(() => {
  const score = analysisScore.value.score
  if (score > 50) return '🚀'
  if (score > 20) return '📈'
  if (score > -20) return '⏸️'
  if (score > -50) return '📉'
  return '⚠️'
})

const recommendationClass = computed(() => {
  const score = analysisScore.value.score
  if (score > 20) return 'bullish'
  if (score < -20) return 'bearish'
  return 'neutral'
})

const confidenceScore = computed(() => {
  return Math.min(Math.abs(analysisScore.value.score), 100)
})

const confidenceClass = computed(() => {
  const score = analysisScore.value.score
  if (score > 20) return 'bullish'
  if (score < -20) return 'bearish'
  return 'neutral'
})

const reasons = computed(() => analysisScore.value.reasons)
const riskWarnings = computed(() => analysisScore.value.warnings)
</script>

<style scoped>
.trading-recommendation {
  background: linear-gradient(135deg, #1e1e1e 0%, #2a2a2a 100%);
  border-radius: 12px;
  padding: 20px;
  border: 1px solid #3a3a3a;
}

.recommendation-header {
  margin-bottom: 20px;
  padding-bottom: 15px;
  border-bottom: 2px solid #3a3a3a;
}

.recommendation-header h3 {
  margin: 0;
  color: #fff;
  font-size: 18px;
}

.recommendation-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.main-recommendation {
  padding: 20px;
  border-radius: 10px;
  border: 2px solid;
}

.main-recommendation.bullish {
  background: rgba(38, 166, 154, 0.1);
  border-color: #26a69a;
}

.main-recommendation.bearish {
  background: rgba(239, 83, 80, 0.1);
  border-color: #ef5350;
}

.main-recommendation.neutral {
  background: rgba(255, 167, 38, 0.1);
  border-color: #ffa726;
}

.recommendation-badge {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 15px;
}

.recommendation-icon {
  font-size: 32px;
}

.recommendation-text {
  font-size: 28px;
  font-weight: 700;
  color: #fff;
}

.confidence-level {
  display: flex;
  align-items: center;
  gap: 10px;
}

.confidence-label {
  color: #999;
  font-size: 14px;
  min-width: 70px;
}

.confidence-bar {
  flex: 1;
  height: 8px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 4px;
  overflow: hidden;
}

.confidence-fill {
  height: 100%;
  transition: width 0.3s ease;
}

.confidence-fill.bullish {
  background: linear-gradient(90deg, #26a69a 0%, #4db6ac 100%);
}

.confidence-fill.bearish {
  background: linear-gradient(90deg, #ef5350 0%, #f44336 100%);
}

.confidence-fill.neutral {
  background: linear-gradient(90deg, #ffa726 0%, #ffb74d 100%);
}

.confidence-value {
  color: #fff;
  font-weight: 600;
  font-size: 14px;
  min-width: 40px;
  text-align: right;
}

.analysis-details h4,
.risk-warning h4 {
  color: #fff;
  font-size: 16px;
  margin: 0 0 12px 0;
}

.reason-list,
.warning-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.reason-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: rgba(255, 255, 255, 0.05);
  border-radius: 6px;
}

.reason-icon {
  font-size: 16px;
}

.reason-text {
  color: #ddd;
  font-size: 14px;
}

.risk-warning {
  background: rgba(255, 152, 0, 0.1);
  border: 1px solid rgba(255, 152, 0, 0.3);
  border-radius: 8px;
  padding: 15px;
}

.warning-list li {
  color: #ffb74d;
  font-size: 14px;
  padding: 4px 0;
}

.disclaimer {
  background: rgba(255, 255, 255, 0.05);
  border-radius: 6px;
  padding: 12px;
}

.disclaimer p {
  margin: 0;
  color: #999;
  font-size: 12px;
  line-height: 1.5;
}
</style>
