<template>
  <el-card class="indicator-panel" shadow="hover">
    <template #header>
      <div class="card-header">
        <span class="title">📊 技术指标</span>
      </div>
    </template>

    <div v-if="indicators" class="indicators-content">
      <!-- MACD -->
      <div class="indicator-section">
        <div class="indicator-title">MACD</div>
        <div class="indicator-values">
          <div class="indicator-item">
            <span class="key">DIF:</span>
            <span class="value" :class="indicators.macd.dif > 0 ? 'positive' : 'negative'">
              {{ indicators.macd.dif.toFixed(2) }}
            </span>
          </div>
          <div class="indicator-item">
            <span class="key">DEA:</span>
            <span class="value" :class="indicators.macd.dea > 0 ? 'positive' : 'negative'">
              {{ indicators.macd.dea.toFixed(2) }}
            </span>
          </div>
          <div class="indicator-item">
            <span class="key">Histogram:</span>
            <span class="value" :class="indicators.macd.histogram > 0 ? 'positive' : 'negative'">
              {{ indicators.macd.histogram.toFixed(2) }}
            </span>
          </div>
        </div>
      </div>

      <!-- KDJ -->
      <div class="indicator-section">
        <div class="indicator-title">KDJ</div>
        <div class="indicator-values">
          <div class="indicator-item">
            <span class="key">K:</span>
            <span class="value">{{ indicators.kdj.k.toFixed(2) }}</span>
          </div>
          <div class="indicator-item">
            <span class="key">D:</span>
            <span class="value">{{ indicators.kdj.d.toFixed(2) }}</span>
          </div>
          <div class="indicator-item">
            <span class="key">J:</span>
            <span class="value" :class="getKDJClass(indicators.kdj.j)">
              {{ indicators.kdj.j.toFixed(2) }}
            </span>
          </div>
        </div>
      </div>

      <!-- RSI -->
      <div class="indicator-section">
        <div class="indicator-title">RSI</div>
        <div class="indicator-values">
          <div class="indicator-item">
            <span class="key">RSI(6):</span>
            <span class="value" :class="getRSIClass(indicators.rsi.rsi6)">
              {{ indicators.rsi.rsi6.toFixed(2) }}
            </span>
          </div>
          <div class="indicator-item">
            <span class="key">RSI(14):</span>
            <span class="value" :class="getRSIClass(indicators.rsi.rsi14)">
              {{ indicators.rsi.rsi14.toFixed(2) }}
            </span>
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
import type { Indicators } from '../services/api'

defineProps<{
  indicators?: Indicators
}>()

function getKDJClass(value: number): string {
  if (value > 80) return 'overbought'
  if (value < 20) return 'oversold'
  return ''
}

function getRSIClass(value: number): string {
  if (value > 70) return 'overbought'
  if (value < 30) return 'oversold'
  return ''
}
</script>

<style scoped>
.indicator-panel {
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

.indicators-content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.indicator-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.indicator-title {
  font-size: 16px;
  font-weight: 600;
  color: #ffa726;
  border-bottom: 2px solid #ffa726;
  padding-bottom: 4px;
}

.indicator-values {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.indicator-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 4px 0;
}

.key {
  font-size: 14px;
  color: #999;
  font-weight: 500;
}

.value {
  font-size: 16px;
  font-weight: 600;
  color: #fff;
}

.positive {
  color: #26a69a !important;
}

.negative {
  color: #ef5350 !important;
}

.overbought {
  color: #ef5350 !important;
}

.oversold {
  color: #26a69a !important;
}

.loading {
  padding: 20px 0;
}
</style>
