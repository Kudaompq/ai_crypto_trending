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
        <div class="info-item" v-if="atr">
          <span class="label">ATR({{ atr.period }}):</span>
          <span class="value atr-value">${{ atr.value.toFixed(2) }}</span>
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

        <!-- Chart Area -->
        <div class="chart-area">
          <!-- Candles Display -->
          <div class="candles-display">
            <div v-for="(candle, index) in displayCandles" :key="index" class="candle-bar">
              <!-- Upper wick (from body top to high) -->
              <div class="wick upper-wick" :style="getUpperWickStyle(candle)"></div>

              <!-- Candle body (from open to close) -->
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

              <!-- Lower wick (from body bottom to low) -->
              <div class="wick lower-wick" :style="getLowerWickStyle(candle)"></div>
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
import type { Candle, ATRIndicator } from '../services/api'

const props = defineProps<{
  candles: Candle[]
  symbol?: string
  atr?: ATRIndicator
}>()

const symbolLabel = computed(() => {
  if (!props.symbol) return 'ETH/USDT'
  // Convert BTCUSDT to BTC/USDT format
  return props.symbol.replace('USDT', '/USDT')
})

const displayCandles = computed(() => {
  if (!props.candles || props.candles.length === 0) return []
  // Show last 100 candles
  return props.candles.slice(-100)
})

const currentPrice = computed(() => {
  if (!props.candles || props.candles.length === 0) return 0
  const current = props.candles[props.candles.length - 1]
  return current?.close ?? 0
})

const priceChange = computed(() => {
  if (!props.candles || props.candles.length < 2) return 0
  const current = props.candles[props.candles.length - 1]
  const previous = props.candles[props.candles.length - 2]
  if (!current || !previous || previous.close === 0) return 0
  return ((current.close - previous.close) / previous.close) * 100
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
    const candle = candles[index]
    if (candle) labels.push(formatTime(candle.timestamp))
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
    const first = props.candles[0]
    const last = props.candles[props.candles.length - 1]
    if (!first || !last) return `${month}/${day}`
    const timeDiff = last.timestamp - first.timestamp
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
  if (range === 0) return { height: '50%', bottom: '50%' }

  // Body height is the difference between open and close
  const bodyHeight = Math.abs(candle.close - candle.open) / range * 100
  // Body bottom is positioned at the lower of open/close
  const bodyBottom = (Math.min(candle.open, candle.close) - lowPrice.value) / range * 100

  return {
    height: `${Math.max(bodyHeight, 1)}%`,
    bottom: `${bodyBottom}%`
  }
}

function getUpperWickStyle(candle: Candle) {
  const range = highPrice.value - lowPrice.value
  if (range === 0) return { height: '0%', bottom: '50%' }

  // Upper wick goes from top of body to high
  const bodyTop = Math.max(candle.open, candle.close)
  const wickHeight = (candle.high - bodyTop) / range * 100
  const wickBottom = (bodyTop - lowPrice.value) / range * 100

  return {
    height: `${wickHeight}%`,
    bottom: `${wickBottom}%`
  }
}

function getLowerWickStyle(candle: Candle) {
  const range = highPrice.value - lowPrice.value
  if (range === 0) return { height: '0%', bottom: '50%' }

  // Lower wick goes from low to bottom of body
  const bodyBottom = Math.min(candle.open, candle.close)
  const wickHeight = (bodyBottom - candle.low) / range * 100
  const wickBottom = (candle.low - lowPrice.value) / range * 100

  return {
    height: `${wickHeight}%`,
    bottom: `${wickBottom}%`
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
  /* Keeps candle tooltips above the candle bodies. */
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

.wick {
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
  width: 2px;
  pointer-events: none;
}

.upper-wick,
.lower-wick {
  background: #999;
}

/* Wick colors based on parent candle */
.candle-bar:has(.bar.bullish) .wick {
  background: #26a69a;
}

.candle-bar:has(.bar.bearish) .wick {
  background: #ef5350;
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

/* ATR value styling */
.atr-value {
  color: #00D9FF !important;
  font-weight: 700;
}

</style>
