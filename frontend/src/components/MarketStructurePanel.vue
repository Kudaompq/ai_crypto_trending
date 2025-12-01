<template>
  <el-card class="market-structure-panel" shadow="hover">
    <template #header>
      <div class="card-header">
        <span class="title">🏗️ 市场结构分析</span>
        <div class="grade-badge" :class="getScoreClass()">
          {{ structure?.market_quality?.grade || '未知' }} 级
        </div>
      </div>
    </template>

    <div v-if="structure" class="structure-content">
      <!-- Market Quality Score -->
      <div class="quality-section">
        <div class="section-title">📊 市场质量评分</div>
        <div class="quality-score">
          <div class="score-circle" :class="getScoreClass()">
            <div class="score-value">{{ Math.round(structure.market_quality?.overall_score || 0) }}</div>
            <div class="score-label">总分</div>
          </div>
          <div class="quality-details">
            <div class="quality-item">
              <span class="label">评级:</span>
              <span class="badge" :class="getGradeClass()">{{ structure.market_quality?.grade }}</span>
            </div>
            <div class="quality-item">
              <span class="label">交易条件:</span>
              <span class="value">{{ translateCondition(structure.market_quality?.trading_condition) }}</span>
            </div>
            <div class="quality-item recommendation">
              <span class="label">建议:</span>
              <span class="value">{{ structure.market_quality?.recommendation }}</span>
            </div>
          </div>
        </div>

        <!-- Score Breakdown -->
        <div class="score-breakdown">
          <div v-for="(score, key) in structure.market_quality?.score_breakdown" :key="key" class="breakdown-item">
            <div class="breakdown-label">{{ translateScoreKey(key) }}</div>
            <el-progress 
              :percentage="Math.round(score)" 
              :color="getProgressColor(score)"
              :stroke-width="8"
            />
          </div>
        </div>
      </div>

      <!-- Trend Confirmation -->
      <div class="trend-section">
        <div class="section-title">📈 趋势确认</div>
        <div class="trend-grid">
          <div class="trend-item">
            <div class="trend-label">EMA排列</div>
            <span class="badge" :class="getAlignmentClass(structure.trend_confirmation?.ema_alignment)">
              {{ translateAlignment(structure.trend_confirmation?.ema_alignment) }}
            </span>
          </div>
          <div class="trend-item">
            <div class="trend-label">MACD信号</div>
            <span class="badge" :class="getSignalClass(structure.trend_confirmation?.macd_signal)">
              {{ translateSignal(structure.trend_confirmation?.macd_signal) }}
            </span>
          </div>
          <div class="trend-item">
            <div class="trend-label">价格位置</div>
            <span class="trend-value">{{ translatePriceVsEMA(structure.trend_confirmation?.price_vs_ema) }}</span>
          </div>
          <div class="trend-item">
            <div class="trend-label">确认强度</div>
            <span class="badge" :class="getStrengthClass(structure.trend_confirmation?.strength)">
              {{ structure.trend_confirmation?.strength }}
            </span>
          </div>
        </div>
        <div class="confirmation-score">
          <span class="score-text">确认得分: {{ Math.round(structure.trend_confirmation?.confirmation_score || 0) }}/100</span>
          <el-progress 
            :percentage="Math.round(structure.trend_confirmation?.confirmation_score || 0)" 
            :color="getProgressColor(structure.trend_confirmation?.confirmation_score || 0)"
          />
        </div>
      </div>

      <!-- Volatility Profile -->
      <div class="volatility-section">
        <div class="section-title">📊 波动性分析</div>
        <div class="volatility-grid">
          <div class="volatility-item">
            <div class="volatility-label">ATR</div>
            <div class="volatility-value">{{ structure.volatility_profile?.current_atr?.toFixed(2) }}</div>
          </div>
          <div class="volatility-item">
            <div class="volatility-label">ATR百分比</div>
            <div class="volatility-value">{{ structure.volatility_profile?.atr_percentage?.toFixed(2) }}%</div>
          </div>
          <div class="volatility-item">
            <div class="volatility-label">波动水平</div>
            <span class="badge" :class="getVolatilityClass(structure.volatility_profile?.volatility_level)">
              {{ translateVolatility(structure.volatility_profile?.volatility_level) }}
            </span>
          </div>
          <div class="volatility-item">
            <div class="volatility-label">趋势</div>
            <span class="volatility-value">
              {{ structure.volatility_profile?.is_expanding ? '扩张中 📈' : '收缩中 📉' }}
            </span>
          </div>
        </div>
        <div v-if="structure.volatility_profile?.risk_adjustment" class="risk-adjustment">
          <div class="custom-alert">
            {{ translateRiskAdjustment(structure.volatility_profile.risk_adjustment) }}
          </div>
        </div>
      </div>

      <!-- Key Level Confluence -->
      <div class="confluence-section">
        <div class="section-title">🎯 关键位汇聚</div>
        
        <div class="nearest-levels">
          <div v-if="structure.key_level_confluence?.nearest_resistance" class="level-card resistance">
            <div class="level-header">
              <span class="level-type">阻力位</span>
              <span class="level-price">{{ structure.key_level_confluence.nearest_resistance.price.toFixed(2) }}</span>
            </div>
            <div class="level-distance">
              距离: {{ structure.key_level_confluence.nearest_resistance.distance.toFixed(2) }}%
            </div>
            <div class="level-strength">
              强度: {{ Math.round(structure.key_level_confluence.nearest_resistance.strength) }}/100
            </div>
            <div class="level-factors">
              <span 
                v-for="(factor, idx) in structure.key_level_confluence.nearest_resistance.factors" 
                :key="idx"
                class="factor-badge"
              >
                {{ factor }}
              </span>
            </div>
          </div>

          <div v-if="structure.key_level_confluence?.nearest_support" class="level-card support">
            <div class="level-header">
              <span class="level-type">支撑位</span>
              <span class="level-price">{{ structure.key_level_confluence.nearest_support.price.toFixed(2) }}</span>
            </div>
            <div class="level-distance">
              距离: {{ structure.key_level_confluence.nearest_support.distance.toFixed(2) }}%
            </div>
            <div class="level-strength">
              强度: {{ Math.round(structure.key_level_confluence.nearest_support.strength) }}/100
            </div>
            <div class="level-factors">
              <span 
                v-for="(factor, idx) in structure.key_level_confluence.nearest_support.factors" 
                :key="idx"
                class="factor-badge"
              >
                {{ factor }}
              </span>
            </div>
          </div>
        </div>

        <!-- Confluence Zones -->
        <div v-if="structure.key_level_confluence?.confluence_zones?.length" class="confluence-zones">
          <div class="zones-title">汇聚区域</div>
          <div 
            v-for="(zone, idx) in structure.key_level_confluence.confluence_zones.slice(0, 3)" 
            :key="idx"
            class="zone-item"
            :class="zone.significance.toLowerCase()"
          >
            <div class="zone-header">
              <span class="badge" :class="getSignificanceClass(zone.significance)">
                {{ translateSignificance(zone.significance) }}
              </span>
              <span class="zone-strength">强度: {{ Math.round(zone.strength) }}</span>
            </div>
            <div class="zone-range">
              {{ zone.price_range[0].toFixed(2) }} - {{ zone.price_range[1].toFixed(2) }}
            </div>
            <div class="zone-factors">
              <span v-for="(factor, fidx) in zone.factors" :key="fidx" class="zone-factor">
                {{ factor }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- Pattern Signals -->
      <div class="pattern-section">
        <div class="section-title">🕯️ 形态信号</div>
        <div class="pattern-summary">
          <div class="pattern-count">
            <div class="count-item bullish">
              <span class="count-label">看涨</span>
              <span class="count-value">{{ structure.pattern_signals?.bullish_count || 0 }}</span>
            </div>
            <div class="count-item bearish">
              <span class="count-label">看跌</span>
              <span class="count-value">{{ structure.pattern_signals?.bearish_count || 0 }}</span>
            </div>
            <div class="count-item dominant">
              <span class="count-label">主导信号</span>
              <span class="badge" :class="getDominantClass(structure.pattern_signals?.dominant_signal)">
                {{ translateDominant(structure.pattern_signals?.dominant_signal) }}
              </span>
            </div>
          </div>
        </div>
        <div v-if="structure.pattern_signals?.recent_patterns?.length" class="recent-patterns">
          <div class="patterns-title">最近形态</div>
          <div class="patterns-list">
            <span 
              v-for="(pattern, idx) in structure.pattern_signals.recent_patterns" 
              :key="idx"
              class="pattern-badge"
            >
              {{ pattern }}
            </span>
          </div>
        </div>
        <div class="pattern-reliability">
          <span class="reliability-text">可靠性: {{ Math.round(structure.pattern_signals?.pattern_reliability || 0) }}%</span>
          <el-progress 
            :percentage="Math.round(structure.pattern_signals?.pattern_reliability || 0)" 
            :color="getProgressColor(structure.pattern_signals?.pattern_reliability || 0)"
          />
        </div>
      </div>

      <!-- Basic Structure (Legacy) -->
      <div class="basic-structure">
        <div class="section-title">🔍 基础结构</div>
        <div class="structure-grid">
          <div class="structure-item">
            <div class="icon">{{ structure.higher_high ? '✅' : '❌' }}</div>
            <div class="info">
              <div class="label">Higher High</div>
              <div class="status">{{ structure.higher_high ? '是' : '否' }}</div>
            </div>
          </div>

          <div class="structure-item">
            <div class="icon">{{ structure.higher_low ? '✅' : '❌' }}</div>
            <div class="info">
              <div class="label">Higher Low</div>
              <div class="status">{{ structure.higher_low ? '是' : '否' }}</div>
            </div>
          </div>

          <div class="structure-item">
            <div class="icon">{{ structure.structure_break ? '⚠️' : '✅' }}</div>
            <div class="info">
              <div class="label">结构破位</div>
              <div class="status" :class="structure.structure_break ? 'warning' : 'safe'">
                {{ structure.structure_break ? '是' : '否' }}
              </div>
            </div>
          </div>

          <div class="structure-item">
            <div class="icon">{{ getRiskIcon() }}</div>
            <div class="info">
              <div class="label">风险等级</div>
              <span class="badge" :class="getRiskClass()">{{ structure.risk_level }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Strengths & Weaknesses -->
      <div class="insights-section">
        <div class="insights-grid">
          <div v-if="structure.market_quality?.strengths?.length" class="insights-card strengths">
            <div class="insights-title">✅ 优势</div>
            <ul class="insights-list">
              <li v-for="(strength, idx) in structure.market_quality.strengths" :key="idx">
                {{ strength }}
              </li>
            </ul>
          </div>

          <div v-if="structure.market_quality?.weaknesses?.length" class="insights-card weaknesses">
            <div class="insights-title">⚠️ 劣势</div>
            <ul class="insights-list">
              <li v-for="(weakness, idx) in structure.market_quality.weaknesses" :key="idx">
                {{ weakness }}
              </li>
            </ul>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="loading">
      <el-skeleton :rows="5" animated />
    </div>
  </el-card>
</template>

<script setup lang="ts">
import type { MarketStructure } from '../services/api'

const props = defineProps<{
  structure?: MarketStructure
}>()

// Helper functions
function getScoreClass(): string {
  const score = props.structure?.market_quality?.overall_score || 0
  if (score >= 85) return 'excellent'
  if (score >= 70) return 'good'
  if (score >= 55) return 'fair'
  return 'poor'
}

function getProgressColor(score: number): string {
  if (score >= 80) return '#26a69a'
  if (score >= 60) return '#ffa726'
  return '#ef5350'
}

function translateCondition(condition?: string): string {
  const map: Record<string, string> = {
    'EXCELLENT': '优秀',
    'GOOD': '良好',
    'FAIR': '一般',
    'POOR': '较差'
  }
  return map[condition || ''] || condition || '未知'
}

function translateScoreKey(key: string): string {
  const map: Record<string, string> = {
    'trend_confirmation': '趋势确认',
    'volatility': '波动性',
    'key_levels': '关键位',
    'patterns': '形态',
    'structure_integrity': '结构完整性'
  }
  return map[key] || key
}

function translateAlignment(alignment?: string): string {
  const map: Record<string, string> = {
    'BULLISH': '多头排列',
    'BEARISH': '空头排列',
    'NEUTRAL': '中性'
  }
  return map[alignment || ''] || alignment || '未知'
}

function translateSignal(signal?: string): string {
  const map: Record<string, string> = {
    'BULLISH': '看涨',
    'BEARISH': '看跌',
    'NEUTRAL': '中性'
  }
  return map[signal || ''] || signal || '未知'
}

function translatePriceVsEMA(position?: string): string {
  const map: Record<string, string> = {
    'ABOVE_KEY_EMAS': '关键均线上方',
    'BELOW_KEY_EMAS': '关键均线下方',
    'ABOVE_MEDIUM_EMA': '中期均线上方',
    'BELOW_MEDIUM_EMA': '中期均线下方',
    'NEUTRAL': '中性'
  }
  return map[position || ''] || position || '未知'
}

function translateVolatility(level?: string): string {
  const map: Record<string, string> = {
    'HIGH': '高',
    'NORMAL': '正常',
    'LOW': '低'
  }
  return map[level || ''] || level || '未知'
}

function translateRiskAdjustment(adjustment?: string): string {
  const map: Record<string, string> = {
    'REDUCE_POSITION_SIZE': '建议减小仓位',
    'CONSIDER_WIDER_STOPS': '考虑放宽止损',
    'NORMAL': '正常仓位管理'
  }
  return map[adjustment || ''] || adjustment || ''
}

function translateSignificance(significance?: string): string {
  const map: Record<string, string> = {
    'CRITICAL': '关键',
    'IMPORTANT': '重要',
    'MODERATE': '一般'
  }
  return map[significance || ''] || significance || '未知'
}

function translateDominant(signal?: string): string {
  const map: Record<string, string> = {
    'BULLISH': '看涨',
    'BEARISH': '看跌',
    'NEUTRAL': '中性'
  }
  return map[signal || ''] || signal || '未知'
}

function getRiskIcon(): string {
  const level = props.structure?.risk_level
  if (level === '低') return '🟢'
  if (level === '中') return '🟡'
  return '🔴'
}

// CSS class functions for dark theme badges
function getGradeClass(): string {
  const grade = props.structure?.market_quality?.grade
  if (grade === 'A') return 'badge-success'
  if (grade === 'B') return 'badge-info'
  if (grade === 'C') return 'badge-warning'
  return 'badge-danger'
}

function getAlignmentClass(alignment?: string): string {
  if (alignment === 'BULLISH') return 'badge-success'
  if (alignment === 'BEARISH') return 'badge-danger'
  return 'badge-info'
}

function getSignalClass(signal?: string): string {
  if (signal === 'BULLISH') return 'badge-success'
  if (signal === 'BEARISH') return 'badge-danger'
  return 'badge-info'
}

function getStrengthClass(strength?: string): string {
  if (strength === 'STRONG') return 'badge-success'
  if (strength === 'MODERATE') return 'badge-warning'
  return 'badge-info'
}

function getVolatilityClass(level?: string): string {
  if (level === 'NORMAL') return 'badge-success'
  if (level === 'LOW') return 'badge-info'
  return 'badge-warning'
}

function getSignificanceClass(significance?: string): string {
  if (significance === 'CRITICAL') return 'badge-danger'
  if (significance === 'IMPORTANT') return 'badge-warning'
  return 'badge-info'
}

function getDominantClass(signal?: string): string {
  if (signal === 'BULLISH') return 'badge-success'
  if (signal === 'BEARISH') return 'badge-danger'
  return 'badge-info'
}

function getRiskClass(): string {
  const level = props.structure?.risk_level
  if (level === '低') return 'badge-success'
  if (level === '中') return 'badge-warning'
  return 'badge-danger'
}
</script>

<style scoped>
.market-structure-panel {
  background: linear-gradient(135deg, #1e1e1e 0%, #2a2a2a 100%);
  border: 1px solid #3a3a3a;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.title {
  font-size: 18px;
  font-weight: 600;
  color: #fff;
}

.structure-content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: #fff;
  margin-bottom: 16px;
  padding-bottom: 8px;
  border-bottom: 2px solid #3a3a3a;
}

/* Quality Section */
.quality-section {
  background: rgba(255, 255, 255, 0.03);
  padding: 20px;
  border-radius: 12px;
}

.quality-score {
  display: flex;
  gap: 24px;
  align-items: center;
  margin-bottom: 20px;
}

.score-circle {
  width: 120px;
  height: 120px;
  border-radius: 50%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  border: 4px solid;
  transition: all 0.3s ease;
}

.score-circle.excellent {
  border-color: #26a69a;
  background: rgba(38, 166, 154, 0.1);
}

.score-circle.good {
  border-color: #66bb6a;
  background: rgba(102, 187, 106, 0.1);
}

.score-circle.fair {
  border-color: #ffa726;
  background: rgba(255, 167, 38, 0.1);
}

.score-circle.poor {
  border-color: #ef5350;
  background: rgba(239, 83, 80, 0.1);
}

.score-value {
  font-size: 36px;
  font-weight: 700;
  color: #fff;
}

.score-label {
  font-size: 12px;
  color: #999;
  margin-top: 4px;
}

.quality-details {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.quality-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.quality-item .label {
  color: #999;
  font-size: 14px;
}

.quality-item .value {
  color: #fff;
  font-size: 14px;
}

.quality-item.recommendation {
  padding: 12px;
  background: rgba(255, 167, 38, 0.1);
  border-left: 3px solid #ffa726;
  border-radius: 4px;
}

.quality-item.recommendation .value {
  color: #ffa726;
  font-weight: 500;
}

.score-breakdown {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.breakdown-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.breakdown-label {
  font-size: 13px;
  color: #bbb;
}

/* Trend Section */
.trend-section {
  background: rgba(255, 255, 255, 0.03);
  padding: 20px;
  border-radius: 12px;
}

.trend-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}

.trend-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px;
  background: rgba(255, 255, 255, 0.05);
  border-radius: 8px;
}

.trend-label {
  font-size: 12px;
  color: #999;
}

.trend-value {
  font-size: 14px;
  color: #fff;
}

.confirmation-score {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.score-text {
  font-size: 14px;
  color: #bbb;
}

/* Volatility Section */
.volatility-section {
  background: rgba(255, 255, 255, 0.03);
  padding: 20px;
  border-radius: 12px;
}

.volatility-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}

.volatility-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px;
  background: rgba(255, 255, 255, 0.05);
  border-radius: 8px;
}

.volatility-label {
  font-size: 12px;
  color: #999;
}

.volatility-value {
  font-size: 16px;
  font-weight: 600;
  color: #fff;
}

.risk-adjustment {
  margin-top: 12px;
}

/* Confluence Section */
.confluence-section {
  background: rgba(255, 255, 255, 0.03);
  padding: 20px;
  border-radius: 12px;
}

.nearest-levels {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 16px;
  margin-bottom: 20px;
}

.level-card {
  padding: 16px;
  border-radius: 8px;
  border-left: 4px solid;
}

.level-card.resistance {
  background: rgba(239, 83, 80, 0.1);
  border-color: #ef5350;
}

.level-card.support {
  background: rgba(38, 166, 154, 0.1);
  border-color: #26a69a;
}

.level-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.level-type {
  font-size: 14px;
  font-weight: 600;
  color: #fff;
}

.level-price {
  font-size: 18px;
  font-weight: 700;
  color: #fff;
}

.level-distance,
.level-strength {
  font-size: 12px;
  color: #bbb;
  margin-bottom: 4px;
}

.level-factors {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
}

.factor-tag {
  font-size: 11px;
}

.confluence-zones {
  margin-top: 16px;
}

.zones-title {
  font-size: 14px;
  font-weight: 600;
  color: #fff;
  margin-bottom: 12px;
}

.zone-item {
  padding: 12px;
  margin-bottom: 12px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.05);
  border-left: 3px solid;
}

.zone-item.critical {
  border-color: #ef5350;
}

.zone-item.important {
  border-color: #ffa726;
}

.zone-item.moderate {
  border-color: #42a5f5;
}

.zone-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.zone-strength {
  font-size: 12px;
  color: #bbb;
}

.zone-range {
  font-size: 14px;
  font-weight: 600;
  color: #fff;
  margin-bottom: 8px;
}

.zone-factors {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.zone-factor {
  font-size: 11px;
  color: #999;
  padding: 4px 8px;
  background: rgba(255, 255, 255, 0.05);
  border-radius: 4px;
}

/* Pattern Section */
.pattern-section {
  background: rgba(255, 255, 255, 0.03);
  padding: 20px;
  border-radius: 12px;
}

.pattern-summary {
  margin-bottom: 16px;
}

.pattern-count {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 12px;
}

.count-item {
  padding: 12px;
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  align-items: center;
}

.count-item.bullish {
  background: rgba(38, 166, 154, 0.1);
}

.count-item.bearish {
  background: rgba(239, 83, 80, 0.1);
}

.count-item.dominant {
  background: rgba(66, 165, 245, 0.1);
}

.count-label {
  font-size: 12px;
  color: #999;
}

.count-value {
  font-size: 24px;
  font-weight: 700;
  color: #fff;
}

.recent-patterns {
  margin-bottom: 16px;
}

.patterns-title {
  font-size: 13px;
  color: #bbb;
  margin-bottom: 8px;
}

.patterns-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.pattern-tag {
  font-size: 12px;
}

.pattern-reliability {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.reliability-text {
  font-size: 14px;
  color: #bbb;
}

/* Basic Structure */
.basic-structure {
  background: rgba(255, 255, 255, 0.03);
  padding: 20px;
  border-radius: 12px;
}

.structure-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 16px;
}

.structure-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: rgba(255, 255, 255, 0.05);
  border-radius: 8px;
}

.icon {
  font-size: 32px;
}

.info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.label {
  font-size: 12px;
  color: #999;
}

.status {
  font-size: 16px;
  font-weight: 600;
  color: #fff;
}

.status.warning {
  color: #ef5350;
}

.status.safe {
  color: #26a69a;
}

/* Insights Section */
.insights-section {
  background: rgba(255, 255, 255, 0.03);
  padding: 20px;
  border-radius: 12px;
}

.insights-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 16px;
}

.insights-card {
  padding: 16px;
  border-radius: 8px;
  border-left: 4px solid;
}

.insights-card.strengths {
  background: rgba(38, 166, 154, 0.1);
  border-color: #26a69a;
}

.insights-card.weaknesses {
  background: rgba(255, 167, 38, 0.1);
  border-color: #ffa726;
}

.insights-title {
  font-size: 14px;
  font-weight: 600;
  color: #fff;
  margin-bottom: 12px;
}

.insights-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.insights-list li {
  font-size: 13px;
  color: #ddd;
  padding-left: 16px;
  position: relative;
}

.insights-list li::before {
  content: '•';
  position: absolute;
  left: 0;
  color: #999;
}

/* Dark Theme Badges */
.badge {
  display: inline-block;
  padding: 4px 12px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
  text-align: center;
  transition: all 0.3s ease;
}

.badge-success {
  background: rgba(38, 166, 154, 0.2);
  color: #26a69a;
  border: 1px solid rgba(38, 166, 154, 0.3);
}

.badge-danger {
  background: rgba(239, 83, 80, 0.2);
  color: #ef5350;
  border: 1px solid rgba(239, 83, 80, 0.3);
}

.badge-warning {
  background: rgba(255, 167, 38, 0.2);
  color: #ffa726;
  border: 1px solid rgba(255, 167, 38, 0.3);
}

.badge-info {
  background: rgba(66, 165, 245, 0.2);
  color: #42a5f5;
  border: 1px solid rgba(66, 165, 245, 0.3);
}

/* Grade Badge (Header) */
.grade-badge {
  padding: 8px 16px;
  border-radius: 6px;
  font-size: 16px;
  font-weight: 600;
  text-align: center;
  transition: all 0.3s ease;
}

.grade-badge.excellent {
  background: rgba(38, 166, 154, 0.2);
  color: #26a69a;
  border: 2px solid #26a69a;
}

.grade-badge.good {
  background: rgba(102, 187, 106, 0.2);
  color: #66bb6a;
  border: 2px solid #66bb6a;
}

.grade-badge.fair {
  background: rgba(255, 167, 38, 0.2);
  color: #ffa726;
  border: 2px solid #ffa726;
}

.grade-badge.poor {
  background: rgba(239, 83, 80, 0.2);
  color: #ef5350;
  border: 2px solid #ef5350;
}

/* Custom Alert */
.custom-alert {
  padding: 12px 16px;
  background: rgba(66, 165, 245, 0.1);
  border-left: 4px solid #42a5f5;
  border-radius: 4px;
  color: #42a5f5;
  font-size: 14px;
}

/* Factor Badges */
.factor-badge {
  display: inline-block;
  padding: 4px 8px;
  margin: 2px;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 4px;
  font-size: 11px;
  color: #ddd;
}

/* Pattern Badges */
.pattern-badge {
  display: inline-block;
  padding: 4px 10px;
  margin: 2px;
  background: rgba(66, 165, 245, 0.15);
  border: 1px solid rgba(66, 165, 245, 0.3);
  border-radius: 4px;
  font-size: 12px;
  color: #42a5f5;
}

.loading {
  padding: 20px 0;
}
</style>
