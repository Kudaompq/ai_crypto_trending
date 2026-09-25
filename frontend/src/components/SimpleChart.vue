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

    <div ref="chartHost" class="kline-chart" role="img" :aria-label="`${symbolLabel} ${interval} K线图`"></div>
    <p v-if="historyError" class="history-error" role="alert">{{ historyError }}</p>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { dispose, init, type Chart, type DataLoader, type KLineData } from 'klinecharts'
import { api, type ATRIndicator, type Candle } from '../services/api'
import { KlineHistory, toKlineChartPeriod } from '../services/klineHistory'

const props = withDefaults(defineProps<{
  candles: Candle[]
  symbol?: string
  interval?: string
  hasMoreBefore?: boolean
  atr?: ATRIndicator
}>(), {
  symbol: 'ETHUSDT',
  interval: '1d',
  hasMoreBefore: true
})

const chartHost = ref<HTMLElement | null>(null)
const historyError = ref('')
const symbolLabel = computed(() => props.symbol.replace('USDT', '/USDT'))
const latestCandle = computed(() => props.candles[props.candles.length - 1])
const previousCandle = computed(() => props.candles[props.candles.length - 2])
const currentPrice = computed(() => latestCandle.value?.close ?? 0)
const priceChange = computed(() => {
  const previous = previousCandle.value?.close ?? 0
  return previous ? ((currentPrice.value - previous) / previous) * 100 : 0
})
const highPrice = computed(() => props.candles.length ? Math.max(...props.candles.map(candle => candle.high)) : 0)
const lowPrice = computed(() => props.candles.length ? Math.min(...props.candles.map(candle => candle.low)) : 0)
const totalVolume = computed(() => props.candles.reduce((sum, candle) => sum + candle.volume, 0))

const history = new KlineHistory((request, signal) =>
  api.getKlineData(request.symbol, request.interval, request.limit, request.endTime, signal)
)
let chart: Chart | null = null
let resizeObserver: ResizeObserver | null = null
let liveBarCallback: ((bar: KLineData) => void) | null = null
let lastPublishedCandle: Candle | undefined

function toKlineData(candle: Candle): KLineData {
  return {
    timestamp: candle.timestamp,
    open: candle.open,
    high: candle.high,
    low: candle.low,
    close: candle.close,
    volume: candle.volume
  }
}

function sameCandle(left: Candle | undefined, right: Candle | undefined): boolean {
  return left?.timestamp === right?.timestamp && left?.open === right?.open && left?.high === right?.high &&
    left?.low === right?.low && left?.close === right?.close && left?.volume === right?.volume
}

function isAbortError(error: unknown): boolean {
  return error instanceof DOMException && error.name === 'AbortError' ||
    typeof error === 'object' && error !== null &&
    ('name' in error) && ((error as { name?: unknown }).name === 'CanceledError')
}

const dataLoader: DataLoader = {
  getBars: async ({ type, symbol, callback }) => {
    if (type === 'init') {
      history.seed(symbol.ticker, props.interval, props.candles, props.hasMoreBefore)
      historyError.value = ''
      callback(history.candles.map(toKlineData), { forward: history.hasMoreBefore, backward: false })
      return
    }
    if (type === 'forward') {
      try {
        const candles = await history.loadOlder(symbol.ticker, props.interval, 100)
        historyError.value = ''
        callback(candles.map(toKlineData), { forward: history.hasMoreBefore, backward: false })
      } catch (error: unknown) {
        if (isAbortError(error)) return
        historyError.value = api.errorMessage(error, '加载更早 K 线失败，可继续拖动重试')
        // Resolve the chart's in-flight request without advancing the cursor.
        // KLineChart unlocks its loader and a subsequent drag can retry.
        callback([], { forward: history.hasMoreBefore, backward: false })
      }
      return
    }
    callback(history.candles.map(toKlineData), { forward: history.hasMoreBefore, backward: false })
  },
  subscribeBar: ({ callback }) => {
    liveBarCallback = callback
    lastPublishedCandle = props.candles[props.candles.length - 1]
  },
  unsubscribeBar: () => {
    liveBarCallback = null
  }
}

watch(() => props.candles, candles => {
  const latest = candles[candles.length - 1]
  if (latest && liveBarCallback && !sameCandle(lastPublishedCandle, latest)) {
    liveBarCallback(toKlineData(latest))
  }
  lastPublishedCandle = latest
}, { deep: true })

onMounted(() => {
  const container = chartHost.value
  if (!container) return

  history.seed(props.symbol, props.interval, props.candles, props.hasMoreBefore)
  chart = init(container, { styles: 'dark', timezone: 'Asia/Shanghai' })
  if (!chart) return
  chart.setSymbol({ ticker: props.symbol, pricePrecision: 8, volumePrecision: 2 })
  chart.setPeriod(toKlineChartPeriod(props.interval))
  chart.setScrollEnabled(true)
  chart.setZoomEnabled(true)
  chart.setDataLoader(dataLoader)

  if (typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(() => chart?.resize())
    resizeObserver.observe(container)
  }
})

onBeforeUnmount(() => {
  liveBarCallback = null
  resizeObserver?.disconnect()
  resizeObserver = null
  history.reset(props.symbol, props.interval)
  if (chart) dispose(chart)
  chart = null
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
.kline-chart { width: 100%; height: 480px; min-width: 0; }
.history-error { margin: 8px 0 0; color: #ff8a80; font-size: 12px; }

@media (max-width: 600px) {
  .chart-card { padding: 12px; }
  .chart-header { align-items: flex-start; flex-direction: column; }
  .chart-summary { gap: 10px 16px; }
  .kline-chart { height: 380px; }
}
</style>
