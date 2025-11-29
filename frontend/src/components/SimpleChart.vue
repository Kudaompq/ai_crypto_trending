<template>
  <div class="simple-chart-container">
    <div class="chart-header">
      <h3>📈 {{ symbolLabel }} K线图</h3>
      <div class="current-price" v-if="candles && candles.length > 0">
        <span class="label">当前价格:</span>
        <span class="price" :class="priceClass">${{ currentPrice.toFixed(2) }}</span>
        <span class="change" :class="changeClass">{{ priceChange >= 0 ? '+' : '' }}{{ priceChange.toFixed(2) }}%</span>
      </div>
    </div>

    <div class="price-levels">
      <div class="levels-row">
        <div class="level-group resistance-group">
          <span class="level-title">压力位:</span>
          <span v-if="topResistance.length === 0" class="no-data">暂无</span>
          <span v-for="(level, i) in topResistance" :key="'r-' + i" class="level-badge resistance">
            ${{ level.price.toFixed(2) }}
          </span>
        </div>
        <div class="level-group support-group">
          <span class="level-title">支撑位:</span>
          <span v-if="topSupport.length === 0" class="no-data">暂无</span>
          <span v-for="(level, i) in topSupport" :key="'s-' + i" class="level-badge support">
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

      <div class="chart-container">
        <!-- Y-axis (Price) -->
        <div class="y-axis">
          <div class="y-label" v-for="(price, i) in yAxisLabels" :key="'y-' + i"
            :style="{ bottom: `${(i / (yAxisLabels.length - 1)) * 100}%` }">
            ${{ price.toFixed(0) }}
          </div>
        </div>

        <!-- Chart Area with SR Lines -->
        <div class="chart-area">
          <!-- SR Level Lines (no labels) -->
          <div class="sr-lines">
            <!-- Resistance Lines -->
            <div v-for="(level, i) in topResistance" :key="'r-line-' + i" class="sr-line resistance-line"
              :style="getSRLineStyle(level.price)">
              <span class="sr-price-label">{{ level.price.toFixed(2) }}</span>
            </div>

            <!-- Support Lines -->
            <div v-for="(level, i) in topSupport" :key="'s-line-' + i" class="sr-line support-line"
              :style="getSRLineStyle(level.price)">
              <span class="sr-price-label">{{ level.price.toFixed(2) }}</span>
            </div>
          </div>

          <!-- Candles Display -->
          <div class="candles-display">
            <div v-for="(candle, index) in displayCandles" :key="index" class="candle-bar">
              <div class="bar" :class="candle.close >= candle.open ? 'bullish' : 'bearish'"
                :style="getCandleStyle(candle)">
                <div class="tooltip">
                  <div>时间: {{ formatTime(candle.timestamp) }}</div>
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

      <!-- X-axis (Time) -->
      <div class="x-axis">
        <div class="x-label" v-for="(label, i) in xAxisLabels" :key="'x-' + i">
          {{ label }}
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
  symbol?: string
}>()

const symbolLabel = computed(() => {
  if (!props.symbol) return 'ETH/USDT'
  // Convert BTCUSDT to BTC/USDT format
  return props.symbol.replace('USDT', '/USDT')
})

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
  if (!props.resistance || props.resistance.length === 0) return []

  // Filter resistance levels that are within the displayed candles' price range
  const high = highPrice.value
  const low = lowPrice.value

  return props.resistance
    .filter(level => level.price >= low && level.price <= high)
    .slice(0, 5)
})

const topSupport = computed(() => {
  if (!props.support || props.support.length === 0) return []

  // Filter support levels that are within the displayed candles' price range
  const high = highPrice.value
  const low = lowPrice.value

  return props.support
    .filter(level => level.price >= low && level.price <= high)
    .slice(0, 5)
})

const yAxisLabels = computed(() => {
  if (!displayCandles.value || displayCandles.value.length === 0) return []
  const high = highPrice.value
  const low = lowPrice.value
  const step = (high - low) / 4
  return [
    low,
    low + step,
    low + step * 2,
    low + step * 3,
    high
  ]
})

const xAxisLabels = computed(() => {
  if (!displayCandles.value || displayCandles.value.length === 0) return []
  const candles = displayCandles.value
  const step = Math.floor(candles.length / 5)
  const labels = []

  for (let i = 0; i < 6; i++) {
    const index = Math.min(i * step, candles.length - 1)
    labels.push(formatTime(candles[index].timestamp))
  }

  return labels
})

function formatTime(timestamp: number): string {
  const date = new Date(timestamp)
  const month = (date.getMonth() + 1).toString().padStart(2, '0')
  const day = date.getDate().toString().padStart(2, '0')
  const hour = date.getHours().toString().padStart(2, '0')
  const minute = date.getMinutes().toString().padStart(2, '0')

  // Show date + time for intraday, just date for daily
  if (props.candles && props.candles.length > 0) {
    const timeDiff = props.candles[props.candles.length - 1].timestamp - props.candles[0].timestamp
    const hoursDiff = timeDiff / (1000 * 60 * 60)

    if (hoursDiff < 48) {
      // Intraday - show time
      return `${hour}:${minute}`
    } else if (hoursDiff < 720) {
      // Less than 30 days - show month/day
      return `${month}/${day}`
    }
  }

  return `${month}/${day}`
}

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

function getSRLineStyle(price: number) {
  const range = highPrice.value - lowPrice.value
  if (range === 0) return { bottom: '50%' }

  const bottom = ((price - lowPrice.value) / range) * 100

  return {
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

.no-data {
  color: #666;
  font-size: 13px;
  font-style: italic;
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

.chart-container {
  position: relative;
  display: flex;
  gap: 10px;
}

.y-axis {
  position: relative;
  width: 60px;
  height: 300px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}

.y-label {
  position: absolute;
  right: 5px;
  transform: translateY(50%);
  font-size: 11px;
  color: #999;
  font-weight: 500;
  background: #1a1a1a;
  padding: 2px 4px;
  border-radius: 3px;
}

.chart-area {
  flex: 1;
  position: relative;
  height: 300px;
  overflow: visible;
  /* Allow labels to overflow */
  padding-right: 60px;
  /* Space for labels */
}

.sr-lines {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  /* Cover full candles area */
  bottom: 0;
  pointer-events: none;
  z-index: 15;
  /* Above candles, below tooltips */
  overflow: visible;
  /* Allow labels to show outside */
}

.sr-line {
  position: absolute;
  left: 0;
  right: 0;
  /* Full width */
  height: 0;
  border-top: 1px solid;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding-right: 10px;
  /* Space before label */
}

.resistance-line {
  border-color: rgba(239, 83, 80, 0.6);
}

.support-line {
  border-color: rgba(38, 166, 154, 0.6);
}

.sr-price-label {
  font-size: 10px;
  font-weight: 700;
  padding: 0;
  background: transparent;
  color: #fff;
  transform: translateY(-50%);
  white-space: nowrap;
  pointer-events: none;
  line-height: 1;
  text-shadow: 0 1px 3px rgba(0, 0, 0, 0.9);
  position: relative;
  z-index: 1;
}

.resistance-line .sr-price-label {
  color: rgba(239, 83, 80, 1);
}

.support-line .sr-price-label {
  color: rgba(38, 166, 154, 1);
}

.candles-display {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: flex-end;
  gap: 2px;
  padding: 10px 0;
  background: linear-gradient(to top, #1a1a1a 0%, #1a1a1a 25%, transparent 25%, transparent 50%, #1a1a1a 50%, #1a1a1a 75%, transparent 75%);
  background-size: 100% 25%;
  z-index: 10;
  /* Above SR lines, allows tooltips to show on top */
}

.x-axis {
  display: flex;
  justify-content: space-between;
  padding: 8px 60px 0 70px;
  margin-top: 5px;
}

.x-label {
  font-size: 11px;
  color: #999;
  font-weight: 500;
}

.candle-bar {
  flex: 1;
  position: relative;
  height: 100%;
  display: flex;
  align-items: flex-end;
  z-index: 20;
  /* Create stacking context for tooltip to escape above SR lines */
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
  z-index: 100;
  /* Highest layer to always show above everything */
  margin-bottom: 5px;
}

.bar:hover .tooltip {
  display: block;
}

.tooltip div {
  margin: 2px 0;
}
</style>
