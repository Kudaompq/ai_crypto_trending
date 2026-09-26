<template>
  <div class="dashboard">
    <!-- Header -->
    <div class="header">
      <div class="header-content">
        <div class="brand-lockup">
          <svg width="32" height="32" viewBox="0 0 64 64" fill="none" xmlns="http://www.w3.org/2000/svg"
            class="logo-icon" aria-hidden="true">
            <rect width="64" height="64" rx="18" fill="url(#originx-mark-gradient)" />
            <path d="M13 43L24 32L32 40L51 21" stroke="white" stroke-width="3.5" stroke-linecap="round"
              stroke-linejoin="round" />
            <path d="M51 21V29M51 21H43" stroke="white" stroke-width="3.5" stroke-linecap="round"
              stroke-linejoin="round" />
            <circle cx="51" cy="21" r="3" fill="#DDFBFF" />
            <defs>
              <linearGradient id="originx-mark-gradient" x1="7" y1="5" x2="60" y2="62" gradientUnits="userSpaceOnUse">
                <stop stop-color="#3278B8" />
                <stop offset="1" stop-color="#5E55D8" />
              </linearGradient>
            </defs>
          </svg>
          <div class="brand-copy">
            <h1 class="title">Originx</h1>
          </div>
        </div>

        <div class="header-tools">
          <div class="last-update">
            <span class="stream-status" :class="streamState">
              {{ streamStatusText }}
            </span>
            <span v-if="store.lastUpdate">最近行情: {{ formatTime(store.lastUpdate) }}</span>
            <span v-if="streamNote" class="stream-note">{{ streamNote }}</span>
          </div>
          <div class="controls">
            <!-- GitHub Link -->
            <a href="https://github.com/kudaompq" target="_blank" rel="noopener noreferrer" class="github-link"
              title="GitHub" aria-label="GitHub">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
                <path
                  d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
              </svg>
              <span>GitHub</span>
            </a>
          </div>
        </div>
      </div>

      <div v-if="store.storageWarning" class="symbol-error" role="alert">{{ store.storageWarning }}</div>
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
          <button class="symbol-action" type="button" :disabled="!store.watchlistReady" @click="showSymbolEditor = !showSymbolEditor"
            title="添加自选交易对" aria-label="添加自选交易对">+</button>
        </div>
        <form v-if="showSymbolEditor" class="symbol-editor" @submit.prevent="addSymbol">
          <input v-model="newSymbol" class="symbol-input" placeholder="如 AVAXUSDT"
            aria-label="自选交易对" :disabled="!store.watchlistReady || symbolValidating" />
          <button class="symbol-action" type="submit" :disabled="!store.watchlistReady || symbolValidating">
            {{ symbolValidating ? '校验中…' : '添加' }}
          </button>
          <span v-if="symbolError" class="symbol-error" role="alert">{{ symbolError }}</span>
        </form>
        <TransitionGroup name="watchlist-order" tag="div" class="watchlist-items">
          <div v-for="item in store.availableSymbols" :key="item.value" class="watchlist-item"
            :class="{
              active: store.symbol === item.value,
              dragging: watchlistDragSymbol === item.value,
              'drop-target': watchlistDragTarget === item.value && watchlistDragSymbol !== item.value,
              'drop-before': isWatchlistDropBefore(item.value),
              'drop-after': isWatchlistDropAfter(item.value)
            }" :data-symbol="item.value"
            @pointerenter="updateWatchlistDragTarget(item.value, $event)"
            @pointermove="updateWatchlistDragTarget(item.value, $event)"
            @pointerup="finishWatchlistDrag(item.value, $event)" @pointercancel="cancelWatchlistDrag">
            <button class="watchlist-drag-handle" type="button" title="拖动调整顺序" :disabled="!store.watchlistReady"
              :aria-label="`拖动 ${item.label} 调整顺序`"
              @pointerdown.stop="beginWatchlistDrag(item.value, $event)">⠿</button>
            <button class="watchlist-select" type="button" :disabled="!store.watchlistReady" :aria-label="`选择 ${item.label}`"
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
            <button v-if="store.availableSymbols.length > 1" class="watchlist-delete" type="button" :disabled="!store.watchlistReady"
              title="删除交易对" :aria-label="`删除 ${item.label}`" @click="removeSymbol(item.value)">×</button>
          </div>
        </TransitionGroup>
      </aside>

      <div class="chart-section">
        <SimpleChart v-if="store.watchlistReady" v-show="store.klineData" :candles="store.klineData?.data ?? []"
          :symbol="store.symbol" :interval="store.interval" :has-more-before="store.klineData?.has_more_before ?? false"
          :symbols="chartSymbols" @selection-change="selectChartSelection" />
        <div v-if="!store.klineData" class="chart-placeholder">
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

const store = useAnalysisStore()
const chartSymbols = computed(() => store.availableSymbols.map(item => item.value))
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
const watchlistDragSymbol = ref<string | null>(null)
const watchlistDragTarget = ref<string | null>(null)
let streamStartedAt = 0
let lastLiveEventAt = 0
let lastFallbackAttempt = 0
let fallbackGeneration: number | null = null
let validatingStreamError = false
let marketGeneration = 0

onMounted(() => {
  void initializeDashboard()
  watchdogTimer = window.setInterval(checkMarketFreshness, 1000)

  // Periodic REST fallback also refreshes the calculated indicators.
  refreshTimer = window.setInterval(() => {
    void handleRefresh()
  }, 60 * 1000)

})

watch(() => store.watchlistReady ? store.availableSymbols.map(item => item.value).join(',') : '', value => {
  if (!value) return
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

function selectWatchlistSymbol(symbol: string) {
  if (!store.watchlistReady) return
  if (store.symbol === symbol) return
  store.setSymbol(symbol)
  void reloadMarket()
}

function selectChartSelection(selection: { symbol: string; interval: string }) {
  const changed = selection.symbol !== store.symbol || selection.interval !== store.interval
  if (!changed) return
  if (selection.symbol !== store.symbol) store.setSymbol(selection.symbol)
  if (selection.interval !== store.interval) store.setInterval(selection.interval)
  void reloadMarket()
}

function beginWatchlistDrag(symbol: string, event: PointerEvent) {
  if (!store.watchlistReady) return
  if (event.button !== undefined && event.button !== 0) return
  watchlistDragSymbol.value = symbol
  watchlistDragTarget.value = symbol
}

function updateWatchlistDragTarget(fallbackSymbol: string, event: PointerEvent) {
  if (!watchlistDragSymbol.value) return
  const pointed = document.elementFromPoint?.(event.clientX, event.clientY)?.closest<HTMLElement>('.watchlist-item')
  watchlistDragTarget.value = pointed?.dataset.symbol || fallbackSymbol
}

function finishWatchlistDrag(fallbackSymbol: string, event: PointerEvent) {
  if (!watchlistDragSymbol.value) return
  updateWatchlistDragTarget(fallbackSymbol, event)
  const sourceSymbol = watchlistDragSymbol.value
  const targetSymbol = watchlistDragTarget.value
  const targetIndex = store.availableSymbols.findIndex(item => item.value === targetSymbol)
  if (targetIndex >= 0 && sourceSymbol !== targetSymbol) store.reorderSymbols(sourceSymbol, targetIndex)
  cancelWatchlistDrag()
}

function cancelWatchlistDrag() {
  watchlistDragSymbol.value = null
  watchlistDragTarget.value = null
}

function isWatchlistDropBefore(symbol: string) {
  return isWatchlistDropTarget(symbol) && !isWatchlistDropAfter(symbol)
}

function isWatchlistDropAfter(symbol: string) {
  if (!watchlistDragSymbol.value || !watchlistDragTarget.value || watchlistDragTarget.value !== symbol) return false
  const sourceIndex = store.availableSymbols.findIndex(item => item.value === watchlistDragSymbol.value)
  const targetIndex = store.availableSymbols.findIndex(item => item.value === symbol)
  return sourceIndex >= 0 && targetIndex >= 0 && sourceIndex < targetIndex
}

function isWatchlistDropTarget(symbol: string) {
  return watchlistDragTarget.value === symbol && watchlistDragSymbol.value !== symbol
}

async function addSymbol() {
  if (!store.watchlistReady) return
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

async function removeSymbol(symbol: string) {
  const selectedBeforeRemoval = store.symbol
  try {
    await store.removeSymbol(symbol)
    if (store.symbol !== selectedBeforeRemoval) void reloadMarket()
  } catch (err: unknown) {
    store.storageWarning = api.errorMessage(err, '移除交易对失败')
  }
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
  if (!store.watchlistReady) return
  const generation = marketGeneration
  await store.fetchAnalysis()
  if (generation === marketGeneration) lastAnalysisRefresh = Date.now()
}

async function initializeDashboard() {
  await store.initializeWatchlist()
  await reloadMarket()
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
  if (!store.watchlistReady) return
  if (store.invalidSymbol) return
  const lastEvent = lastLiveEventAt || streamStartedAt
  if (Date.now() - lastEvent < 15_000) return
  if (Date.now() - lastFallbackAttempt >= 15_000) void runFallback()
}

async function runFallback() {
  if (!store.watchlistReady) return
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
  background: #11161e;
  border: 1px solid #252d38;
  border-radius: 10px;
  padding: 8px 14px;
  margin-bottom: 14px;
}

.stream-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  color: #ffa726;
  background: rgba(255, 167, 38, 0.08);
  border: 1px solid rgba(255, 167, 38, 0.2);
  border-radius: 999px;
  font-size: 11px;
}

.stream-status::before {
  content: '';
  width: 6px;
  height: 6px;
  flex: 0 0 auto;
  border-radius: 50%;
  background: currentColor;
}

.stream-status.live {
  color: #26a69a;
  background: rgba(38, 166, 154, 0.08);
  border-color: rgba(38, 166, 154, 0.2);
}

.stream-status.fallback { color: #ffa726; }
.stream-status.unavailable { color: #ef5350; }
.stream-note { color: #8994a5; }

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.brand-lockup {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 9px;
}

.brand-copy {
  display: grid;
  gap: 2px;
}

.title {
  font-size: 20px;
  font-weight: 650;
  letter-spacing: -0.025em;
  line-height: 1;
  color: #f1f6fc;
  margin: 0;
}

.logo-icon { width: 32px; height: 32px; flex: 0 0 auto; }

.header-tools {
  display: flex;
  align-items: center;
  gap: 16px;
}

.controls {
  display: flex;
  align-items: center;
}

.github-link {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 7px 9px;
  color: #d5deeb;
  background: transparent;
  border: 1px solid #303946;
  border-radius: 8px;
  font-size: 12px;
  text-decoration: none;
  transition: background 150ms ease, color 150ms ease;
}

.github-link svg { width: 15px; height: 15px; }
.github-link:hover { color: #fff; background: rgba(255, 255, 255, 0.08); }

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

.last-update {
  display: flex;
  min-height: 0;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;
  color: #8994a5;
  font-size: 12px;
}

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
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: 56px;
  padding: 8px 10px;
  border: 1px solid transparent;
  border-radius: 8px;
  color: #ddd;
  transition: background-color 150ms ease, border-color 150ms ease, box-shadow 150ms ease, opacity 150ms ease;
}

.watchlist-item.watchlist-order-move {
  transition: transform 200ms cubic-bezier(.2, .75, .25, 1), background-color 150ms ease,
    border-color 150ms ease, box-shadow 150ms ease, opacity 150ms ease;
}

.watchlist-item:hover { background: #242424; }
.watchlist-item.active { border-color: #667eea; background: rgba(102, 126, 234, .14); }
.watchlist-item.dragging {
  z-index: 1;
  opacity: .62;
  border-style: dashed;
  border-color: #9aa7ff;
  box-shadow: 0 0 0 3px rgba(102, 126, 234, .16);
}
.watchlist-item.drop-target {
  border-color: rgba(142, 156, 255, .72);
  background: rgba(102, 126, 234, .22);
  box-shadow: inset 0 0 0 1px rgba(142, 156, 255, .12);
}
.watchlist-item.drop-before::before,
.watchlist-item.drop-after::after {
  position: absolute;
  right: 7px;
  left: 7px;
  z-index: 2;
  height: 3px;
  border-radius: 999px;
  background: linear-gradient(90deg, #667eea, #b49cff);
  box-shadow: 0 0 10px rgba(142, 156, 255, .7);
  content: '';
  pointer-events: none;
}
.watchlist-item.drop-before::before { top: -5px; }
.watchlist-item.drop-after::after { bottom: -5px; }
.watchlist-drag-handle { flex-shrink: 0; padding: 3px 2px; border: 0; background: transparent; color: #777; cursor: grab; touch-action: none; }
.watchlist-drag-handle:active { cursor: grabbing; }
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

@media (prefers-reduced-motion: reduce) {
  .watchlist-item.watchlist-order-move,
  .watchlist-item {
    transition-duration: 0.01ms;
  }
}

@media (max-width: 600px) {
  .header { padding: 10px 12px; }

  .header-content {
    flex-wrap: wrap;
    gap: 10px;
  }

  .brand-lockup { gap: 8px; }

  .header-tools {
    width: 100%;
    justify-content: space-between;
    gap: 8px;
  }

  .last-update { gap: 6px 8px; }
}

@media (max-width: 768px) {
  .controls {
    flex: 0 0 auto;
  }

  .market-layout { flex-direction: column; }
  .watchlist { flex-basis: auto; }
  .watchlist-items { max-height: none; overflow: visible; }
  .chart-placeholder { min-height: 360px; }
  .interval-buttons { width: 100%; }
  .interval-btn { padding: 8px 10px; font-size: 13px; white-space: nowrap; }
}
</style>
