<template>
  <div class="dashboard">
    <!-- Header -->
    <div class="header">
      <div class="header-content">
        <h1 class="title">
          <svg width="40" height="40" viewBox="0 0 64 64" fill="none" xmlns="http://www.w3.org/2000/svg"
            class="logo-icon">
            <rect width="64" height="64" rx="16" fill="url(#paint0_linear)" />
            <path d="M12 44L24 32L32 40L52 20" stroke="white" stroke-width="4" stroke-linecap="round"
              stroke-linejoin="round" />
            <path d="M52 20V30" stroke="white" stroke-width="4" stroke-linecap="round" />
            <path d="M52 20H42" stroke="white" stroke-width="4" stroke-linecap="round" />
            <defs>
              <linearGradient id="paint0_linear" x1="0" y1="0" x2="64" y2="64" gradientUnits="userSpaceOnUse">
                <stop stop-color="#4F46E5" />
                <stop offset="1" stop-color="#7C3AED" />
              </linearGradient>
            </defs>
          </svg>
          加密货币趋势分析系统
        </h1>

        <div class="controls">
          <!-- GitHub Link -->
          <a href="https://github.com/kudaompq" target="_blank" rel="noopener noreferrer" class="github-link"
            title="GitHub">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
              <path
                d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
            </svg>
          </a>
        </div>
      </div>

      <div v-if="store.storageWarning" class="symbol-error" role="alert">{{ store.storageWarning }}</div>

      <div class="last-update">
        <span class="stream-status" :class="streamState">
          {{ streamStatusText }}
        </span>
        <span v-if="store.lastUpdate">最近行情: {{ formatTime(store.lastUpdate) }}</span>
        <span v-if="streamNote" class="stream-note">{{ streamNote }}</span>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="store.loading && !store.analysisResult" class="loading-overlay">
      <el-icon class="is-loading" :size="60">
        <Loading />
      </el-icon>
      <div class="loading-text">正在获取数据...</div>
    </div>

    <!-- Error State -->
    <el-alert v-if="store.error" :title="store.error" type="error" show-icon closable @close="store.error = null"
      style="margin-bottom: 20px" />

    <!-- Main Content -->
    <div class="market-layout">
      <aside class="watchlist" aria-label="Watchlist">
        <div class="watchlist-heading">
          <div>
            <h2>Watchlist</h2>
            <span class="watchlist-stream-status" :class="store.priceStreamState">{{ priceStreamStatusText }}</span>
          </div>
          <button class="symbol-action" type="button" @click="showSymbolEditor = !showSymbolEditor"
            title="添加自选交易对" aria-label="添加自选交易对">+</button>
        </div>
        <form v-if="showSymbolEditor" class="symbol-editor" @submit.prevent="addSymbol">
          <input v-model="newSymbol" class="symbol-input" placeholder="如 AVAXUSDT"
            aria-label="自选交易对" :disabled="symbolValidating" />
          <button class="symbol-action" type="submit" :disabled="symbolValidating">
            {{ symbolValidating ? '校验中…' : '添加' }}
          </button>
          <span v-if="symbolError" class="symbol-error" role="alert">{{ symbolError }}</span>
        </form>
        <div class="watchlist-items">
          <div v-for="item in store.availableSymbols" :key="item.value" class="watchlist-item"
            :class="{ active: store.symbol === item.value }" :data-symbol="item.value">
            <button class="watchlist-select" type="button" :aria-label="`选择 ${item.label}`"
              @click="selectWatchlistSymbol(item.value)">
              <div class="watchlist-symbol-info">
                <strong>{{ item.label }}</strong>
                <span>{{ item.value }}</span>
              </div>
              <div class="watchlist-price-info">
                <strong>{{ store.unavailableSymbols.includes(item.value)
                  ? '暂不支持' : formatPrice(store.pricesBySymbol[item.value]?.price) }}</strong>
                <span v-if="store.pricesBySymbol[item.value]?.change24hPercent !== undefined" class="watchlist-change"
                  :class="(store.pricesBySymbol[item.value]?.change24hPercent ?? 0) >= 0 ? 'positive' : 'negative'">
                  {{ formatChangePercent(store.pricesBySymbol[item.value]?.change24hPercent) }}
                </span>
              </div>
            </button>
            <button v-if="store.availableSymbols.length > 1" class="watchlist-delete" type="button"
              title="删除交易对" :aria-label="`删除 ${item.label}`" @click="removeSymbol(item.value)">×</button>
          </div>
        </div>
      </aside>

      <div class="chart-section">
        <div class="interval-buttons" role="group" aria-label="K线周期">
          <button v-for="option in intervalOptions" :key="option.value" class="interval-btn"
            :data-interval="option.value" :aria-pressed="store.interval === option.value"
            :class="{ active: store.interval === option.value }" @click="changeInterval(option.value)">
            {{ option.label }}
          </button>
        </div>
        <SimpleChart v-if="store.klineData" :candles="store.klineData.data"
          :symbol="store.symbol" :interval="store.interval" :has-more-before="store.klineData.has_more_before"
          :atr="store.analysisResult?.indicators.atr" />
        <div v-else class="chart-placeholder">
          {{ store.loading ? '正在加载 K 线…' : '选择交易对后显示 K 线图' }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useAnalysisStore } from '../stores/analysis'
import { api } from '../services/api'
import type { MarketEvent } from '../services/api'
import SimpleChart from '../components/SimpleChart.vue'
import { BINANCE_KLINE_INTERVALS } from '../services/klineHistory'

const store = useAnalysisStore()
const showSymbolEditor = ref(false)
const newSymbol = ref('')
const symbolError = ref<string | null>(null)
const symbolValidating = ref(false)
let refreshTimer: number | null = null
let watchdogTimer: number | null = null
let marketStream: EventSource | null = null
let lastAnalysisRefresh = 0
const streamState = ref<'connecting' | 'live' | 'fallback' | 'unavailable'>('connecting')
const streamNote = ref('')
const streamStatusText = computed(() => ({
  connecting: '正在连接实时行情',
  live: '实时更新',
  fallback: '备用更新（非实时）',
  unavailable: '行情不可用或已过期'
})[streamState.value])
const priceStreamStatusText = computed(() => ({
  connecting: '报价连接中',
  live: '实时价格',
  reconnecting: '价格重连中'
})[store.priceStreamState])
let streamStartedAt = 0
let lastLiveEventAt = 0
let lastFallbackAttempt = 0
let fallbackGeneration: number | null = null
let validatingStreamError = false
let marketGeneration = 0

const intervalLabels: Record<(typeof BINANCE_KLINE_INTERVALS)[number], string> = {
  '1m': '1分', '3m': '3分', '5m': '5分', '15m': '15分', '30m': '30分',
  '1h': '1小时', '2h': '2小时', '4h': '4小时', '6h': '6小时', '8h': '8小时', '12h': '12小时',
  '1d': '1天', '3d': '3天', '1w': '1周', '1M': '1月'
}
const intervalOptions = BINANCE_KLINE_INTERVALS.map(value => ({ value, label: intervalLabels[value] }))

onMounted(() => {
  void reloadMarket()
  watchdogTimer = window.setInterval(checkMarketFreshness, 1000)

  // Periodic REST fallback also refreshes the calculated indicators.
  refreshTimer = window.setInterval(() => {
    void handleRefresh()
  }, 60 * 1000)

})

watch(() => store.availableSymbols.map(item => item.value).join(','), () => {
  void store.startWatchlistPriceStream()
}, { immediate: true })

onUnmounted(() => {
  marketGeneration++
  if (watchdogTimer) clearInterval(watchdogTimer)
  if (refreshTimer) {
    clearInterval(refreshTimer)
  }

  marketStream?.close()
  store.stopWatchlistPriceStream()
})

function changeInterval(interval: string) {
  store.setInterval(interval)
  void reloadMarket()
}

function selectWatchlistSymbol(symbol: string) {
  if (store.symbol === symbol) return
  store.setSymbol(symbol)
  void reloadMarket()
}

async function addSymbol() {
  symbolValidating.value = true
  symbolError.value = null
  try {
    const selected = await store.addCustomSymbol(newSymbol.value)
    newSymbol.value = ''
    showSymbolEditor.value = false
    store.setSymbol(selected)
    await reloadMarket()
  } catch (err: unknown) {
    symbolError.value = api.errorMessage(err, '添加交易对失败')
  } finally {
    symbolValidating.value = false
  }
}

function removeSymbol(symbol: string) {
  store.removeSymbol(symbol)
  void reloadMarket()
}

function formatPrice(price?: number): string {
  if (price === undefined || !Number.isFinite(price)) return '获取中…'
  return new Intl.NumberFormat('en-US', { maximumFractionDigits: 8 }).format(price)
}

function formatChangePercent(change?: number): string {
  if (change === undefined || !Number.isFinite(change)) return '—'
  return `${change > 0 ? '+' : ''}${change.toFixed(2)}%`
}

async function handleRefresh() {
  const generation = marketGeneration
  await store.fetchAnalysis()
  if (generation === marketGeneration) lastAnalysisRefresh = Date.now()
}

async function reloadMarket() {
  const generation = ++marketGeneration
  marketStream?.close()
  marketStream = null
  streamState.value = 'connecting'
  streamNote.value = ''
  streamStartedAt = Date.now()
  lastLiveEventAt = 0
  lastFallbackAttempt = 0
  validatingStreamError = false
  await handleRefresh()
  if (generation !== marketGeneration) return
  if (store.invalidSymbol) {
    streamState.value = 'unavailable'
    streamNote.value = store.error || '该交易对不可用，请选择其他交易对'
    return
  }
  streamStartedAt = Date.now()
  lastFallbackAttempt = streamStartedAt
  if (store.lastUpdate && !store.error) {
    streamState.value = 'fallback'
    streamNote.value = '等待实时行情，当前使用备用更新'
  } else {
    streamState.value = 'unavailable'
    streamNote.value = store.error || '暂时无法获取行情数据'
  }
  connectMarketStream()
}

function checkMarketFreshness() {
  if (store.invalidSymbol) return
  const lastEvent = lastLiveEventAt || streamStartedAt
  if (Date.now() - lastEvent < 15_000) return
  if (Date.now() - lastFallbackAttempt >= 15_000) void runFallback()
}

async function runFallback() {
  if (fallbackGeneration === marketGeneration || store.invalidSymbol) return
  const generation = marketGeneration
  const liveAtStart = lastLiveEventAt
  fallbackGeneration = generation
  lastFallbackAttempt = Date.now()
  streamState.value = 'fallback'
  const success = await store.fetchFallbackKline()
  if (fallbackGeneration === generation) fallbackGeneration = null
  if (generation !== marketGeneration || store.invalidSymbol || lastLiveEventAt > liveAtStart) return
  streamState.value = success ? 'fallback' : 'unavailable'
  if (!success && !streamNote.value) streamNote.value = '备用行情获取失败'
}

function connectMarketStream() {
  marketStream?.close()
  const expectedSymbol = store.symbol
  const expectedInterval = store.interval
  const generation = marketGeneration

  marketStream = api.createMarketStream(expectedSymbol, expectedInterval, (event: MarketEvent) => {
    if (generation !== marketGeneration || event.symbol !== store.symbol || event.interval !== store.interval) return

    store.updateRealtimeCandle(event.candle)
    lastLiveEventAt = Date.now()
    streamState.value = 'live'
    streamNote.value = ''

    // Price and the active candle update on every event; heavier indicator
    // calculations run at most once every 15 seconds or when a candle closes.
    const analysisIsStale = Date.now() - lastAnalysisRefresh >= 15_000
    if (event.is_final || analysisIsStale) {
      void handleRefresh()
    }
  }, (status) => {
    if (generation !== marketGeneration || status.symbol !== store.symbol || status.interval !== store.interval) return
    if (status.state === 'reconnecting') {
      streamNote.value = status.message
      void runFallback()
    }
  })

  marketStream.onerror = () => {
    if (generation !== marketGeneration) return
    streamNote.value = '行情连接中断，正在重试'
    void runFallback()
    if (validatingStreamError) return
    validatingStreamError = true
    void api.validateSymbol(expectedSymbol).catch((err: unknown) => {
      if (generation !== marketGeneration || !api.invalidSelection(err)) return
      marketStream?.close()
      store.invalidSymbol = true
      store.error = api.errorMessage(err, '该交易对不可用')
      streamNote.value = store.error
      streamState.value = 'unavailable'
    }).finally(() => { validatingStreamError = false })
  }
}

function formatTime(date: Date): string {
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}
</script>

<style scoped>
.dashboard {
  min-height: 100vh;
  background: linear-gradient(135deg, #0a0a0a 0%, #1a1a1a 100%);
  padding: 20px;
}

.header {
  background: linear-gradient(135deg, #1e1e1e 0%, #2a2a2a 100%);
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 24px;
  border: 1px solid #3a3a3a;
}

.stream-status {
  display: inline-flex;
  align-items: center;
  margin-right: 12px;
  color: #ffa726;
}

.stream-status::before {
  content: '';
  width: 8px;
  height: 8px;
  margin-right: 6px;
  border-radius: 50%;
  background: currentColor;
}

.stream-status.live {
  color: #26a69a;
}

.stream-status.fallback { color: #ffa726; }
.stream-status.unavailable { color: #ef5350; }
.stream-note { margin-left: 12px; color: #aaa; }

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.title {
  font-size: 32px;
  font-weight: 700;
  color: #fff;
  margin: 0;
  display: flex;
  align-items: center;
  gap: 12px;
}



.controls {
  display: flex;
  gap: 12px;
  align-items: center;
}

.symbol-action {
  padding: 8px 12px;
  border: 1px solid rgba(102, 126, 234, 0.5);
  border-radius: 8px;
  background: rgba(102, 126, 234, 0.15);
  color: #fff;
  cursor: pointer;
}

.symbol-action:disabled { opacity: 0.6; cursor: wait; }

.symbol-editor {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin: 8px 0;
}

.symbol-input {
  padding: 9px 12px;
  min-width: 250px;
  border: 1px solid #555;
  border-radius: 8px;
  background: #242424;
  color: #fff;
}

.symbol-error { color: #ff8a80; font-size: 13px; }

.interval-buttons {
  display: flex;
  flex-wrap: nowrap;
  gap: 8px;
  max-width: 100%;
  margin-bottom: 12px;
  overflow-x: auto;
  scrollbar-width: thin;
  background: rgba(255, 255, 255, 0.05);
  padding: 4px;
  border-radius: 10px;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.interval-btn {
  flex: 0 0 auto;
  padding: 8px 12px;
  border: none;
  background: transparent;
  color: #999;
  font-size: 14px;
  font-weight: 600;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.3s ease;
  position: relative;
  overflow: hidden;
}

.interval-btn::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  opacity: 0;
  transition: opacity 0.3s ease;
  z-index: -1;
}

.interval-btn:hover {
  color: #fff;
  transform: translateY(-2px);
}

.interval-btn:hover::before {
  opacity: 0.3;
}

.interval-btn.active {
  color: #fff;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  box-shadow: 0 4px 15px rgba(102, 126, 234, 0.4);
}

.interval-btn.active::before {
  opacity: 1;
}

.github-link {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: #fff;
  transition: all 0.3s ease;
  text-decoration: none;
}

.github-link:hover {
  background: rgba(255, 255, 255, 0.1);
  border-color: rgba(102, 126, 234, 0.5);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.3);
}

.github-link svg {
  transition: transform 0.3s ease;
}

.github-link:hover svg {
  transform: scale(1.1);
}

.last-update {
  font-size: 14px;
  color: #999;
}

.loading-overlay {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 100px 0;
  gap: 20px;
}

.loading-text {
  font-size: 18px;
  color: #999;
}

.content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.market-layout {
  display: flex;
  align-items: stretch;
  gap: 16px;
  min-height: 520px;
}

.watchlist {
  display: flex;
  flex: 0 0 250px;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
  border: 1px solid #333;
  border-radius: 12px;
  background: #171717;
}

.watchlist-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.watchlist-heading h2 {
  margin: 0 0 5px;
  color: #eee;
  font-size: 18px;
}

.watchlist-stream-status {
  color: #999;
  font-size: 12px;
}

.watchlist-stream-status.live { color: #26a69a; }
.watchlist-stream-status.reconnecting { color: #ffa726; }

.watchlist-items {
  display: flex;
  flex-direction: column;
  gap: 6px;
  overflow-y: auto;
}

.watchlist-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: 56px;
  padding: 8px 10px;
  border: 1px solid transparent;
  border-radius: 8px;
  color: #ddd;
  cursor: pointer;
}

.watchlist-item:hover { background: #242424; }
.watchlist-item.active { border-color: #667eea; background: rgba(102, 126, 234, .14); }
.watchlist-select { display: flex; flex: 1; align-items: center; justify-content: space-between; gap: 8px; min-width: 0; padding: 0; border: 0; background: transparent; color: inherit; text-align: left; cursor: pointer; }
.watchlist-symbol-info, .watchlist-price-info { display: flex; flex-direction: column; gap: 4px; }
.watchlist-symbol-info strong, .watchlist-price-info strong { font-size: 13px; }
.watchlist-symbol-info span { color: #888; font-size: 11px; }
.watchlist-price-info { align-items: flex-end; }
.watchlist-change { font-size: 12px; font-variant-numeric: tabular-nums; }
.watchlist-change.positive { color: #26a69a; }
.watchlist-change.negative { color: #ef5350; }
.watchlist-delete { flex-shrink: 0; padding: 0 4px; border: 0; background: transparent; color: #888; cursor: pointer; }
.watchlist-delete:hover { color: #ff8a80; }
.chart-section {
  flex: 1;
  min-width: 0;
}

.chart-placeholder {
  display: grid;
  min-height: 500px;
  place-items: center;
  border: 1px solid #333;
  border-radius: 12px;
  color: #999;
  background: #171717;
}

@media (max-width: 768px) {
  .header-content {
    flex-direction: column;
    gap: 16px;
    align-items: flex-start;
  }

  .title {
    font-size: 24px;
  }

  .controls {
    width: 100%;
    flex-direction: column;
  }

  .market-layout { flex-direction: column; }
  .watchlist { flex-basis: auto; }
  .watchlist-items { max-height: none; overflow: visible; }
  .chart-placeholder { min-height: 360px; }
  .interval-buttons { width: 100%; }
  .interval-btn { padding: 8px 10px; font-size: 13px; white-space: nowrap; }
}
</style>
