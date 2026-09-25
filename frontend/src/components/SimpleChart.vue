<template>
  <section class="chart-card">
    <div class="chart-header">
      <div>
        <h3>{{ symbolLabel }} K线图</h3>
        <span class="chart-interval">{{ interval }}</span>
      </div>
      <div v-if="candles.length" class="current-price">
        <span class="label">当前价格:</span>
        <strong class="price" :class="priceChange >= 0 ? 'positive' : 'negative'">${{ currentPrice.toFixed(2) }}</strong>
        <span class="change" :class="priceChange >= 0 ? 'positive' : 'negative'">
          {{ priceChange >= 0 ? '+' : '' }}{{ priceChange.toFixed(2) }}%
        </span>
      </div>
    </div>

    <div class="chart-summary">
      <div class="info-item"><span class="label">最高:</span><strong>${{ highPrice.toFixed(2) }}</strong></div>
      <div class="info-item"><span class="label">最低:</span><strong>${{ lowPrice.toFixed(2) }}</strong></div>
      <div class="info-item"><span class="label">成交量:</span><strong>{{ totalVolume.toFixed(0) }}</strong></div>
      <div v-if="atr" class="info-item"><span class="label">ATR({{ atr.period }}):</span><strong>${{ atr.value.toFixed(2) }}</strong></div>
    </div>

    <div ref="chartHost" class="lightweight-chart" role="img" :aria-label="`${symbolLabel} ${interval} K线图`"></div>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  CandlestickSeries,
  ColorType,
  createChart,
  type CandlestickData,
  type IChartApi,
  type ISeriesApi,
  type Time,
  type UTCTimestamp
} from 'lightweight-charts'
import type { ATRIndicator, Candle } from '../services/api'

const props = defineProps<{
  candles: Candle[]
  symbol?: string
  interval?: string
  atr?: ATRIndicator
}>()

const chartHost = ref<HTMLElement | null>(null)
const symbolLabel = computed(() => props.symbol ? props.symbol.replace('USDT', '/USDT') : 'ETH/USDT')
const displayCandles = computed(() => props.candles.slice(-100))
const currentCandle = computed(() => displayCandles.value[displayCandles.value.length - 1])
const previousCandle = computed(() => displayCandles.value[displayCandles.value.length - 2])
const currentPrice = computed(() => currentCandle.value?.close ?? 0)
const priceChange = computed(() => {
  const previous = previousCandle.value?.close ?? 0
  return previous ? ((currentPrice.value - previous) / previous) * 100 : 0
})
const highPrice = computed(() => displayCandles.value.length ? Math.max(...displayCandles.value.map(candle => candle.high)) : 0)
const lowPrice = computed(() => displayCandles.value.length ? Math.min(...displayCandles.value.map(candle => candle.low)) : 0)
const totalVolume = computed(() => displayCandles.value.reduce((sum, candle) => sum + candle.volume, 0))

let chart: IChartApi | null = null
let candleSeries: ISeriesApi<'Candlestick', Time> | null = null
let resizeObserver: ResizeObserver | null = null
let lastChartCandles: Candle[] = []

function toChartData(candles: Candle[]): CandlestickData<Time>[] {
  return candles.map(candle => ({
    time: Math.floor(candle.timestamp / 1000) as UTCTimestamp,
    open: candle.open,
    high: candle.high,
    low: candle.low,
    close: candle.close
  }))
}

function sameCandle(left: Candle, right: Candle): boolean {
  return left.timestamp === right.timestamp && left.open === right.open && left.high === right.high &&
    left.low === right.low && left.close === right.close && left.volume === right.volume
}

function setChartData(candles: Candle[]) {
  if (!candleSeries) return
  candleSeries.setData(toChartData(candles))
  lastChartCandles = candles.slice()
}

function updateChartData(candles: Candle[]) {
  if (!candleSeries) return
  const next = candles.slice(-100)
  if (next.length === 0) {
    setChartData([])
    return
  }
  const last = next[next.length - 1]!
  const oldLast = lastChartCandles[lastChartCandles.length - 1]
  const prefixLength = Math.min(lastChartCandles.length, next.length) - 1
  const sameHistory = prefixLength >= 0 && lastChartCandles.slice(0, prefixLength)
    .every((candle, index) => next[index] && sameCandle(candle, next[index]!))
  if (!oldLast || !sameHistory || last.timestamp < oldLast.timestamp) {
    setChartData(next)
    return
  }
  candleSeries.update(toChartData([last])[0]!)
  lastChartCandles = next
}

onMounted(() => {
  const container = chartHost.value
  if (!container) return
  chart = createChart(container, {
    width: container.clientWidth || 640,
    height: 480,
    layout: { background: { type: ColorType.Solid, color: '#171717' }, textColor: '#c6c9d1' },
    grid: { vertLines: { color: '#252525' }, horzLines: { color: '#252525' } },
    handleScroll: true,
    handleScale: true,
    rightPriceScale: { borderColor: '#393939' },
    timeScale: { borderColor: '#393939', timeVisible: true, secondsVisible: false },
    crosshair: { vertLine: { color: '#727272' }, horzLine: { color: '#727272' } }
  })
  candleSeries = chart.addSeries(CandlestickSeries, {
    upColor: '#26a69a',
    downColor: '#ef5350',
    borderVisible: false,
    wickUpColor: '#26a69a',
    wickDownColor: '#ef5350'
  })
  setChartData(props.candles.slice(-100))
  chart.timeScale().fitContent()
  if (typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(entries => {
      const entry = entries[0]
      if (entry && chart) chart.resize(Math.floor(entry.contentRect.width), Math.floor(entry.contentRect.height))
    })
    resizeObserver.observe(container)
  }
})

watch(() => props.candles, candles => updateChartData(candles), { deep: true })
watch(() => [props.symbol, props.interval], () => {
  setChartData(props.candles.slice(-100))
  chart?.timeScale().fitContent()
})

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
  resizeObserver = null
  chart?.remove()
  chart = null
  candleSeries = null
  lastChartCandles = []
})
</script>

<style scoped>
.chart-card {
  overflow: hidden;
  padding: 18px;
  border: 1px solid #333;
  border-radius: 12px;
  background: #171717;
  color: #eee;
}

.chart-header { display: flex; justify-content: space-between; align-items: center; gap: 16px; margin-bottom: 14px; }
.chart-header h3 { display: inline; margin: 0; font-size: 18px; }
.chart-interval { margin-left: 10px; color: #888; font-size: 12px; }
.current-price { display: flex; align-items: baseline; gap: 8px; }
.label { color: #999; font-size: 12px; }
.price, .change { font-size: 14px; }
.positive { color: #26a69a; }
.negative { color: #ef5350; }
.chart-summary { display: flex; gap: 22px; flex-wrap: wrap; margin: 12px 0; }
.info-item { display: flex; gap: 6px; align-items: baseline; font-size: 12px; }
.info-item strong { color: #ddd; font-weight: 500; }
.lightweight-chart { width: 100%; height: 480px; min-width: 0; }

@media (max-width: 600px) {
  .chart-card { padding: 12px; }
  .chart-header { align-items: flex-start; flex-direction: column; }
  .chart-summary { gap: 10px 16px; }
  .lightweight-chart { height: 380px; }
}
</style>
