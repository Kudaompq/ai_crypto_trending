<template>
  <div class="dashboard">
    <!-- Header -->
    <div class="header">
      <div class="header-content">
        <h1 class="title">
          <span class="icon">📊</span>
          ETH 量化分析系统
        </h1>
        
        <div class="controls">
          <el-select v-model="store.interval" @change="handleRefresh" size="large">
            <el-option label="15分钟" value="15m" />
            <el-option label="1小时" value="1h" />
            <el-option label="4小时" value="4h" />
            <el-option label="1天" value="1d" />
          </el-select>
          
          <el-button
            type="primary"
            :loading="store.loading"
            @click="handleRefresh"
            size="large"
          >
            <el-icon><Refresh /></el-icon>
            刷新数据
          </el-button>
        </div>
      </div>

      <div v-if="store.lastUpdate" class="last-update">
        最后更新: {{ formatTime(store.lastUpdate) }}
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="store.loading && !store.analysisResult" class="loading-overlay">
      <el-icon class="is-loading" :size="60"><Loading /></el-icon>
      <div class="loading-text">正在获取数据...</div>
    </div>

    <!-- Error State -->
    <el-alert
      v-if="store.error"
      :title="store.error"
      type="error"
      show-icon
      closable
      @close="store.error = null"
      style="margin-bottom: 20px"
    />

    <!-- Main Content -->
    <div v-if="store.analysisResult" class="content">
      <!-- Simple Chart -->
      <div class="chart-section">
        <SimpleChart
          v-if="store.klineData"
          :candles="store.klineData.data"
          :resistance="store.analysisResult.sr_levels.resistance"
          :support="store.analysisResult.sr_levels.support"
        />
      </div>

      <!-- Analysis Panels -->
      <div class="panels-grid">
        <!-- Left Column -->
        <div class="left-column">
          <TrendPanel
            :trend="store.analysisResult.trend"
            :trend-color="store.trendColor"
          />
          
          <IndicatorPanel
            :indicators="store.analysisResult.indicators"
          />
        </div>

        <!-- Right Column -->
        <div class="right-column">
          <SRLevelPanel
            :sr-levels="store.analysisResult.sr_levels"
          />
          
          <PatternPanel
            :patterns="store.analysisResult.candlestick_patterns"
          />
          
          <MarketStructurePanel
            :structure="store.analysisResult.market_structure"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { Refresh, Loading } from '@element-plus/icons-vue'
import { useAnalysisStore } from '../stores/analysis'
import SimpleChart from '../components/SimpleChart.vue'
import TrendPanel from '../components/TrendPanel.vue'
import IndicatorPanel from '../components/IndicatorPanel.vue'
import SRLevelPanel from '../components/SRLevelPanel.vue'
import PatternPanel from '../components/PatternPanel.vue'
import MarketStructurePanel from '../components/MarketStructurePanel.vue'

const store = useAnalysisStore()

onMounted(() => {
  handleRefresh()
})

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

.icon {
  font-size: 36px;
}

.controls {
  display: flex;
  gap: 12px;
  align-items: center;
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

.left-column,
.right-column {
  display: flex;
  flex-direction: column;
  gap: 24px;
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
