<template>
  <el-card class="trend-panel" shadow="hover">
    <template #header>
      <div class="card-header">
        <span class="title">📈 趋势分析</span>
      </div>
    </template>

    <div v-if="trend" class="trend-content">
      <!-- Trend Direction -->
      <div class="trend-item">
        <div class="label">趋势方向</div>
        <div class="value" :style="{ color: trendColor }">
          <span class="trend-icon">{{ trendIcon }}</span>
          {{ trend.direction }}
        </div>
      </div>

      <!-- Trend Strength -->
      <div class="trend-item">
        <div class="label">趋势强度</div>
        <div class="strength-bar">
          <el-progress
            :percentage="Math.round(trend.strength * 100)"
            :color="trendColor"
            :stroke-width="20"
          />
        </div>
      </div>

      <!-- Change Probability -->
      <div class="trend-item">
        <div class="label">反转概率</div>
        <div class="value">
          {{ (trend.change_probability * 100).toFixed(1) }}%
        </div>
      </div>
    </div>

    <div v-else class="loading">
      <el-skeleton :rows="3" animated />
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { TrendAnalysis } from '../services/api'

const props = defineProps<{
  trend?: TrendAnalysis
  trendColor?: string
}>()

const trendIcon = computed(() => {
  if (!props.trend) return '⏸️'
  if (props.trend.direction === '上升') return '↗️'
  if (props.trend.direction === '下降') return '↘️'
  return '↔️'
})
</script>

<style scoped>
.trend-panel {
  background: linear-gradient(135deg, #1e1e1e 0%, #2a2a2a 100%);
  border: 1px solid #3a3a3a;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.title {
  font-size: 18px;
  font-weight: 600;
  color: #fff;
}

.trend-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.trend-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.label {
  font-size: 14px;
  color: #999;
  font-weight: 500;
}

.value {
  font-size: 24px;
  font-weight: 700;
  display: flex;
  align-items: center;
  gap: 8px;
}

.trend-icon {
  font-size: 28px;
}

.strength-bar {
  width: 100%;
}

.loading {
  padding: 20px 0;
}

:deep(.el-progress__text) {
  color: #fff !important;
  font-weight: 600;
}
</style>
