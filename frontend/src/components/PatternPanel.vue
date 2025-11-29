<template>
  <el-card class="pattern-panel" shadow="hover">
    <template #header>
      <div class="card-header">
        <span class="title">🕯️ 蜡烛图形态</span>
      </div>
    </template>

    <div v-if="patterns && patterns.length > 0" class="patterns-content">
      <div
        v-for="(pattern, index) in patterns"
        :key="index"
        class="pattern-item"
        :class="pattern.direction === '看涨' ? 'bullish' : pattern.direction === '看跌' ? 'bearish' : 'neutral'"
      >
        <div class="pattern-header">
          <span class="pattern-name">{{ pattern.pattern }}</span>
          <el-tag
            :type="pattern.direction === '看涨' ? 'success' : pattern.direction === '看跌' ? 'danger' : 'warning'"
            size="small"
          >
            {{ pattern.direction }}
          </el-tag>
        </div>
        
        <div class="pattern-details">
          <div class="detail-row">
            <span class="label">类型:</span>
            <span class="value">{{ pattern.type }}</span>
          </div>
          <div class="detail-row">
            <span class="label">位置:</span>
            <span class="value">{{ pattern.position === 0 ? '当前' : `${pattern.position}根前` }}</span>
          </div>
          <div class="detail-row">
            <span class="label">可靠性:</span>
            <el-progress
              :percentage="Math.round(pattern.reliability * 100)"
              :color="getReliabilityColor(pattern.reliability)"
              :stroke-width="12"
            />
          </div>
        </div>

        <div class="pattern-description">
          {{ pattern.description }}
        </div>
      </div>
    </div>

    <div v-else-if="patterns && patterns.length === 0" class="no-patterns">
      <el-empty description="未识别到明显形态" />
    </div>

    <div v-else class="loading">
      <el-skeleton :rows="3" animated />
    </div>
  </el-card>
</template>

<script setup lang="ts">
import type { CandlestickPattern } from '../services/api'

defineProps<{
  patterns?: CandlestickPattern[]
}>()

function getReliabilityColor(reliability: number): string {
  if (reliability >= 0.9) return '#26a69a'
  if (reliability >= 0.75) return '#66bb6a'
  if (reliability >= 0.6) return '#ffa726'
  return '#ef5350'
}
</script>

<style scoped>
.pattern-panel {
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

.patterns-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.pattern-item {
  padding: 16px;
  border-radius: 8px;
  border-left: 4px solid;
  background: rgba(255, 255, 255, 0.05);
}

.pattern-item.bullish {
  border-color: #26a69a;
}

.pattern-item.bearish {
  border-color: #ef5350;
}

.pattern-item.neutral {
  border-color: #ffa726;
}

.pattern-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.pattern-name {
  font-size: 16px;
  font-weight: 700;
  color: #fff;
}

.pattern-details {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 12px;
}

.detail-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.label {
  font-size: 13px;
  color: #999;
  min-width: 60px;
}

.value {
  font-size: 14px;
  color: #fff;
  font-weight: 500;
}

.pattern-description {
  font-size: 13px;
  color: #bbb;
  line-height: 1.6;
  padding-top: 8px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.no-patterns {
  padding: 40px 0;
}

.loading {
  padding: 20px 0;
}

:deep(.el-progress__text) {
  color: #fff !important;
  font-size: 12px !important;
}
</style>
