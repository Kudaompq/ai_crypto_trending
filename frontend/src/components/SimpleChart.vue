<template>
  <div class="simple-chart-container">
    <div class="chart-header">
      <h3>📈 ETH/USDT K线图</h3>
      <div class="current-price" v-if="candles && candles.length > 0">
        <span class="label">当前价格:</span>
        <span class="price" :class="priceClass">${{ currentPrice.toFixed(2) }}</span>
        <span class="change" :class="changeClass">{{ priceChange >= 0 ? '+' : '' }}{{ priceChange.toFixed(2) }}%</span>
      </div>
    </div>
    
    <div class="price-levels" v-if="resistance || support">
      <div class="levels-row">
        <div class="level-group resistance-group">
          <span class="level-title">压力位:</span>
          <span v-for="(level, i) in topResistance" :key="'r-'+i" class="level-badge resistance">
            ${{ level.price.toFixed(2) }}
          </span>
        </div>
        <div class="level-group support-group">
          <span class="level-title">支撑位:</span>
          <span v-for="(level, i) in topSupport" :key="'s-'+i" class="level-badge support">
            ${{ level.price.toFixed(2) }}
          </span>
        </div>
      </div>
    </div>

    <div class="simple-chart">
      <div class="chart-info">
        <div class="info-item">
          <span class="label">最高:</span>
          <span class="value">${{ highPrice.toFixed(2) }}</span>
        </div>
        <div class="info-item">
          <span class="label">最低:</span>
          <span class="value">${{ lowPrice.toFixed(2) }}</span>
        </div>
        <div class="info-item">
          <span class="label">成交量:</span>
          <span class="value">{{ totalVolume.toFixed(0) }}</span>
        </div>
      </div>
      
      <div class="candles-display">
        <div v-for="(candle, index) in displayCandles" :key="index" class="candle-bar">
          <div 
            class="bar" 
            :class="candle.close >= candle.open ? 'bullish' : 'bearish'"
            :style="getCandleStyle(candle)"
          >
            <div class="tooltip">
              <div>开: ${{ candle.open.toFixed(2) }}</div>
              <div>高: ${{ candle.high.toFixed(2) }}</div>
              <div>低: ${{ candle.low.toFixed(2) }}</div>
              <div>收: ${{ candle.close.toFixed(2) }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Candle, SRLevel } from '../services/api'

const props = defineProps<{
  candles: Candle[]
  resistance?: SRLevel[]
  support?: SRLevel[]
}>()

const displayCandles = computed(() => {
  if (!props.candles || props.candles.length === 0) return []
  // Show last 50 candles
  return props.candles.slice(-50)
})

const currentPrice = computed(() => {
  if (!props.candles || props.candles.length === 0) return 0
  return props.candles[props.candles.length - 1].close
})

const priceChange = computed(() => {
  if (!props.candles || props.candles.length < 2) return 0
  const current = props.candles[props.candles.length - 1].close
  const previous = props.candles[props.candles.length - 2].close
  return ((current - previous) / previous) * 100
})

const priceClass = computed(() => {
  return priceChange.value >= 0 ? 'positive' : 'negative'
})

const changeClass = computed(() => {
  return priceChange.value >= 0 ? 'positive' : 'negative'
})

const highPrice = computed(() => {
  if (!displayCandles.value || displayCandles.value.length === 0) return 0
  return Math.max(...displayCandles.value.map(c => c.high))
})

const lowPrice = computed(() => {
  if (!displayCandles.value || displayCandles.value.length === 0) return 0
  return Math.min(...displayCandles.value.map(c => c.low))
})

const totalVolume = computed(() => {
  if (!displayCandles.value || displayCandles.value.length === 0) return 0
  return displayCandles.value.reduce((sum, c) => sum + c.volume, 0)
})

const topResistance = computed(() => {
  return props.resistance?.slice(0, 3) || []
})

const topSupport = computed(() => {
  return props.support?.slice(0, 3) || []
})

function getCandleStyle(candle: Candle) {
  const range = highPrice.value - lowPrice.value
  if (range === 0) return { height: '50%' }
  
  const bodyHeight = Math.abs(candle.close - candle.open) / range * 100
  const bottom = (candle.low - lowPrice.value) / range * 100
  
  return {
    height: `${Math.max(bodyHeight, 2)}%`,
    bottom: `${bottom}%`
  }
}
</script>

<style scoped>
.simple-chart-container {
  background: linear-gradient(135deg, #1e1e1e 0%, #2a2a2a 100%);
  border-radius: 12px;
  padding: 20px;
  border: 1px solid #3a3a3a;
}

.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 15px;
  border-bottom: 2px solid #3a3a3a;
}

.chart-header h3 {
  margin: 0;
  color: #fff;
  font-size: 20px;
}

.current-price {
  display: flex;
  align-items: center;
  gap: 12px;
}

.current-price .label {
  color: #999;
  font-size: 14px;
}

.current-price .price {
  font-size: 28px;
  font-weight: 700;
}

.current-price .price.positive {
  color: #26a69a;
}

.current-price .price.negative {
  color: #ef5350;
}

.current-price .change {
  font-size: 16px;
  font-weight: 600;
  padding: 4px 12px;
  border-radius: 6px;
}

.current-price .change.positive {
  color: #26a69a;
  background: rgba(38, 166, 154, 0.1);
}

.current-price .change.negative {
  color: #ef5350;
  background: rgba(239, 83, 80, 0.1);
}

.price-levels {
  margin-bottom: 20px;
}

.levels-row {
  display: flex;
  gap: 20px;
  flex-wrap: wrap;
}

.level-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.level-title {
  color: #999;
  font-size: 14px;
  font-weight: 600;
}

.level-badge {
  padding: 4px 12px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
}

.level-badge.resistance {
  background: rgba(239, 83, 80, 0.2);
  color: #ef5350;
  border: 1px solid rgba(239, 83, 80, 0.4);
}

.level-badge.support {
  background: rgba(38, 166, 154, 0.2);
  color: #26a69a;
  border: 1px solid rgba(38, 166, 154, 0.4);
}

.simple-chart {
  background: #1a1a1a;
  border-radius: 8px;
  padding: 20px;
}

.chart-info {
  display: flex;
  gap: 30px;
  margin-bottom: 20px;
  padding-bottom: 15px;
  border-bottom: 1px solid #2a2a2a;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.info-item .label {
  color: #999;
  font-size: 13px;
}

.info-item .value {
  color: #fff;
  font-size: 15px;
  font-weight: 600;
}

.candles-display {
  display: flex;
  align-items: flex-end;
  gap: 2px;
  height: 300px;
  padding: 10px 0;
}

.candle-bar {
  flex: 1;
  position: relative;
  height: 100%;
  display: flex;
  align-items: flex-end;
}

.bar {
  width: 100%;
  position: relative;
  border-radius: 2px;
  cursor: pointer;
  transition: opacity 0.2s;
}

.bar:hover {
  opacity: 0.8;
}

.bar.bullish {
  background: #26a69a;
}

.bar.bearish {
  background: #ef5350;
}

.tooltip {
  display: none;
  position: absolute;
  bottom: 100%;
  left: 50%;
  transform: translateX(-50%);
  background: rgba(0, 0, 0, 0.9);
  color: #fff;
  padding: 8px 12px;
  border-radius: 6px;
  font-size: 12px;
  white-space: nowrap;
  z-index: 10;
  margin-bottom: 5px;
}

.bar:hover .tooltip {
  display: block;
}

.tooltip div {
  margin: 2px 0;
}
</style>
