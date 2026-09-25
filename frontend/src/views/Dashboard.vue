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
          <div class="symbol-selector">
            <select :value="store.symbol" @change="changeSymbol" class="symbol-select" aria-label="选择交易对">
              <option v-for="sym in store.availableSymbols" :key="sym.value" :value="sym.value">
                {{ sym.label }}
              </option>
            </select>
            <button class="symbol-action" type="button" @click="showSymbolEditor = !showSymbolEditor" title="添加自选交易对">+</button>
            <button v-if="store.availableSymbols.length > 1" class="symbol-action" type="button"
              @click="removeSelectedSymbol" title="删除当前交易对">−</button>
          </div>

          <div class="interval-buttons">
            <button v-for="option in intervalOptions" :key="option.value" class="interval-btn"
              :class="{ active: store.interval === option.value }" @click="changeInterval(option.value)">
              {{ option.label }}
            </button>
          </div>

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

      <form v-if="showSymbolEditor" class="symbol-editor" @submit.prevent="addSymbol">
        <input v-model="newSymbol" class="symbol-input" placeholder="输入合约交易对，如 AVAXUSDT"
          aria-label="自选交易对" :disabled="symbolValidating" />
        <button class="symbol-action" type="submit" :disabled="symbolValidating">
          {{ symbolValidating ? '校验中…' : '添加' }}
        </button>
        <span v-if="symbolError" class="symbol-error" role="alert">{{ symbolError }}</span>
      </form>
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
    <div v-if="store.analysisResult" class="content">
      <!-- Simple Chart -->
      <div class="chart-section">
        <SimpleChart v-if="store.klineData" :candles="store.klineData.data"
          :symbol="store.symbol" :atr="store.analysisResult.indicators.atr" />
      </div>

    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useAnalysisStore } from '../stores/analysis'
import { api } from '../services/api'
import type { MarketEvent } from '../services/api'
import SimpleChart from '../components/SimpleChart.vue'

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
let streamStartedAt = 0
let lastLiveEventAt = 0
let lastFallbackAttempt = 0
let fallbackGeneration: number | null = null
let validatingStreamError = false
let marketGeneration = 0

const intervalOptions = [
  { label: '15分钟', value: '15m' },
  { label: '1小时', value: '1h' },
  { label: '4小时', value: '4h' },
  { label: '1天', value: '1d' }
]

onMounted(() => {
  void reloadMarket()
  watchdogTimer = window.setInterval(checkMarketFreshness, 1000)

  // Periodic REST fallback also refreshes the calculated indicators.
  refreshTimer = window.setInterval(() => {
    void handleRefresh()
  }, 60 * 1000)

})

onUnmounted(() => {
  marketGeneration++
  if (watchdogTimer) clearInterval(watchdogTimer)
  if (refreshTimer) {
    clearInterval(refreshTimer)
  }

  marketStream?.close()
})

function changeInterval(interval: string) {
  store.setInterval(interval)
  void reloadMarket()
}

function changeSymbol(event: Event) {
  store.setSymbol((event.target as HTMLSelectElement).value)
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

function removeSelectedSymbol() {
  store.removeSymbol(store.symbol)
  void reloadMarket()
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

.symbol-selector {
  display: flex;
  align-items: center;
  gap: 6px;
  position: relative;
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

.symbol-select {
  padding: 10px 16px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 10px;
  color: #fff;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s ease;
  outline: none;
  min-width: 160px;
}

.symbol-select:hover {
  background: rgba(255, 255, 255, 0.08);
  border-color: rgba(102, 126, 234, 0.5);
  transform: translateY(-1px);
}

.symbol-select:focus {
  border-color: #667eea;
  box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.2);
}

.symbol-select option {
  background: #1e1e1e;
  color: #fff;
  padding: 8px;
}

.interval-buttons {
  display: flex;
  gap: 8px;
  background: rgba(255, 255, 255, 0.05);
  padding: 4px;
  border-radius: 10px;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.interval-btn {
  padding: 8px 20px;
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

.chart-section {
  width: 100%;
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
}
</style>
