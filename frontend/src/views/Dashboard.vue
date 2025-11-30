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
            <select v-model="store.symbol" @change="changeSymbol" class="symbol-select">
              <option v-for="sym in store.availableSymbols" :key="sym.value" :value="sym.value">
                {{ sym.icon }} {{ sym.label }}
              </option>
            </select>
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

      <div v-if="store.lastUpdate" class="last-update">
        最后更新: {{ formatTime(store.lastUpdate) }}
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
          :resistance="store.analysisResult.sr_levels.resistance" :support="store.analysisResult.sr_levels.support"
          :symbol="store.symbol" :ema="store.analysisResult.indicators.ema"
          :fibonacci="store.analysisResult.indicators.fibonacci" :atr="store.analysisResult.indicators.atr" />
      </div>

      <!-- Analysis Panels -->
      <div class="panels-grid">
        <!-- Left Column -->
        <div class="left-column">
          <!-- Trading Recommendation -->
          <TradingRecommendation :analysis="store.analysisResult" />

          <TrendPanel :trend="store.analysisResult.trend" :trend-color="store.trendColor" />

          <IndicatorPanel :indicators="store.analysisResult.indicators" />
        </div>

        <!-- Right Column -->
        <div class="right-column">
          <SRLevelPanel :sr-levels="store.analysisResult.sr_levels" />

          <PatternPanel :patterns="store.analysisResult.candlestick_patterns" />

          <MarketStructurePanel :structure="store.analysisResult.market_structure" />
        </div>
      </div>
    </div>

    <!-- Floating Opportunity Button -->
    <button class="opportunity-button" @click="openModal" :class="{ 'has-notification': hasNewOpportunities }">
      <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M9 11l3 3L22 4" />
        <path d="M21 12v7a2 2 0 01-2 2H5a2 2 0 01-2-2V5a2 2 0 012-2h11" />
      </svg>
      <span class="notification-badge" v-if="hasNewOpportunities">{{ opportunityCount }}</span>
    </button>

    <!-- Opportunity Modal -->
    <div v-if="showOpportunityModal" class="modal-overlay" @click="closeModal">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h2>🎯 交易机会</h2>
          <button class="close-button" @click="closeModal">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M18 6L6 18M6 6l12 12" />
            </svg>
          </button>
        </div>
        <div class="modal-body">
          <TradingOpportunity :symbol="store.symbol" :interval="store.interval" :hide-header="true"
            @opportunities-updated="onOpportunitiesUpdated" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useAnalysisStore } from '../stores/analysis'
import { api } from '../services/api'
import type { TradingOpportunity as TradingOpportunityType } from '../services/api'
import SimpleChart from '../components/SimpleChart.vue'
import TradingOpportunity from '../components/TradingOpportunity.vue'
import TrendPanel from '../components/TrendPanel.vue'
import IndicatorPanel from '../components/IndicatorPanel.vue'
import SRLevelPanel from '../components/SRLevelPanel.vue'
import PatternPanel from '../components/PatternPanel.vue'
import MarketStructurePanel from '../components/MarketStructurePanel.vue'
import TradingRecommendation from '../components/TradingRecommendation.vue'

const store = useAnalysisStore()
let refreshTimer: number | null = null

// Modal state
const showOpportunityModal = ref(false)
const hasNewOpportunities = ref(false)
const opportunityCount = ref(0)

// Background opportunity monitoring
let opportunityCheckInterval: number | null = null
const lastSeenOpportunityIds = ref<Set<string>>(new Set())
const currentOpportunities = ref<TradingOpportunityType[]>([])

// Background check for new opportunities
const checkForNewOpportunities = async () => {
  try {
    const response = await api.getOpportunities(store.symbol, store.interval, 2.0)
    const opportunities = response.opportunities

    // Update current opportunities
    currentOpportunities.value = opportunities

    // Find new opportunities (not in lastSeen set)
    const newOpportunities = opportunities.filter((opp: TradingOpportunityType) =>
      !lastSeenOpportunityIds.value.has(opp.id)
    )

    if (newOpportunities.length > 0) {
      hasNewOpportunities.value = true
      opportunityCount.value = newOpportunities.length
      console.log(`🎯 发现 ${newOpportunities.length} 个新交易机会!`)
    }
  } catch (error) {
    console.error('Failed to check opportunities:', error)
  }
}

// Handle opportunities update from modal (when user opens modal)
const onOpportunitiesUpdated = () => {
  // This is called when modal is open and data is fetched
  // We don't need to do anything here as background check handles it
}

// Open modal
const openModal = () => {
  showOpportunityModal.value = true
}

// Close modal and mark opportunities as seen
const closeModal = () => {
  showOpportunityModal.value = false

  // Mark all current opportunities as seen
  if (currentOpportunities.value.length > 0) {
    currentOpportunities.value.forEach(opp => {
      lastSeenOpportunityIds.value.add(opp.id)
    })

    // Clear notification
    hasNewOpportunities.value = false
    opportunityCount.value = 0

    console.log(`✅ 已标记 ${currentOpportunities.value.length} 个机会为已读`)
  }
}

const intervalOptions = [
  { label: '15分钟', value: '15m' },
  { label: '1小时', value: '1h' },
  { label: '4小时', value: '4h' },
  { label: '1天', value: '1d' }
]

onMounted(() => {
  store.fetchAnalysis()

  // Real-time price updates every 1 second
  refreshTimer = window.setInterval(() => {
    handleRefresh()
  }, 1000)

  // Start background opportunity monitoring - every 5 seconds for real-time updates
  checkForNewOpportunities() // Initial check
  opportunityCheckInterval = window.setInterval(checkForNewOpportunities, 5 * 1000) // Every 5 seconds
  console.log('🔄 后台机会监控已启动 (每5秒检查一次)')
})

onUnmounted(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
  }

  if (opportunityCheckInterval) {
    clearInterval(opportunityCheckInterval)
    console.log('🛑 后台机会监控已停止')
  }
})

function changeInterval(interval: string) {
  store.setInterval(interval)
  handleRefresh()
}

function changeSymbol() {
  handleRefresh()
}

async function handleRefresh() {
  await store.fetchAnalysis()
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
  position: relative;
}

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

.panels-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
}

.full-width-panel {
  grid-column: 1 / -1;
}

.left-column,
.right-column {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

/* Floating Opportunity Button */
.opportunity-button {
  position: fixed;
  bottom: 40px;
  right: 40px;
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border: none;
  color: #fff;
  cursor: pointer;
  box-shadow: 0 8px 24px rgba(102, 126, 234, 0.4);
  transition: all 0.3s ease;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
}

.opportunity-button:hover {
  transform: translateY(-4px) scale(1.05);
  box-shadow: 0 12px 32px rgba(102, 126, 234, 0.6);
}

.opportunity-button.has-notification {
  animation: pulse 2s infinite;
}

@keyframes pulse {

  0%,
  100% {
    box-shadow: 0 8px 24px rgba(102, 126, 234, 0.4);
  }

  50% {
    box-shadow: 0 8px 32px rgba(102, 126, 234, 0.8);
  }
}

.notification-badge {
  position: absolute;
  top: -4px;
  right: -4px;
  background: #ef5350;
  color: #fff;
  border-radius: 50%;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 700;
  border: 2px solid #1a1a2e;
  animation: bounce 0.5s ease;
}

@keyframes bounce {

  0%,
  100% {
    transform: scale(1);
  }

  50% {
    transform: scale(1.2);
  }
}

/* Modal Overlay */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.8);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
  animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }

  to {
    opacity: 1;
  }
}

.modal-content {
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
  border-radius: 16px;
  width: 90%;
  max-width: 900px;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
  animation: slideUp 0.3s ease;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(40px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 24px 28px;
  border-bottom: 2px solid rgba(255, 255, 255, 0.1);
}

.modal-header h2 {
  margin: 0;
  font-size: 24px;
  color: #fff;
  font-weight: 700;
}

.close-button {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.3s ease;
}

.close-button:hover {
  background: rgba(255, 255, 255, 0.1);
  transform: rotate(90deg);
}

.modal-body {
  padding: 24px 28px;
  overflow-y: auto;
  flex: 1;
}

.modal-body::-webkit-scrollbar {
  width: 8px;
}

.modal-body::-webkit-scrollbar-track {
  background: rgba(255, 255, 255, 0.05);
  border-radius: 4px;
}

.modal-body::-webkit-scrollbar-thumb {
  background: rgba(102, 126, 234, 0.5);
  border-radius: 4px;
}

.modal-body::-webkit-scrollbar-thumb:hover {
  background: rgba(102, 126, 234, 0.7);
}

@media (max-width: 1200px) {
  .panels-grid {
    grid-template-columns: 1fr;
  }
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
