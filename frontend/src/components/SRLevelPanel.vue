<template>
  <el-card class="sr-panel" shadow="hover">
    <template #header>
      <div class="card-header">
        <span class="title">🎯 支撑/压力位</span>
      </div>
    </template>

    <div v-if="srLevels" class="sr-content">
      <!-- Resistance Levels -->
      <div class="sr-section">
        <div class="section-title resistance-title">
          <span>📈 压力位</span>
        </div>
        <div v-if="srLevels.resistance.length > 0" class="levels">
          <div
            v-for="(level, index) in srLevels.resistance.slice(0, 5)"
            :key="`r-${index}`"
            class="level-item resistance"
          >
            <span class="price">${{ level.price.toFixed(2) }}</span>
            <el-progress
              :percentage="Math.round(level.strength * 100)"
              color="#ef5350"
              :stroke-width="12"
              :show-text="false"
            />
            <span class="strength">{{ (level.strength * 100).toFixed(0) }}%</span>
          </div>
        </div>
        <div v-else class="no-data">暂无数据</div>
      </div>

      <!-- Support Levels -->
      <div class="sr-section">
        <div class="section-title support-title">
          <span>📉 支撑位</span>
        </div>
        <div v-if="srLevels.support.length > 0" class="levels">
          <div
            v-for="(level, index) in srLevels.support.slice(0, 5)"
            :key="`s-${index}`"
            class="level-item support"
          >
            <span class="price">${{ level.price.toFixed(2) }}</span>
            <el-progress
              :percentage="Math.round(level.strength * 100)"
              color="#26a69a"
              :stroke-width="12"
              :show-text="false"
            />
            <span class="strength">{{ (level.strength * 100).toFixed(0) }}%</span>
          </div>
        </div>
        <div v-else class="no-data">暂无数据</div>
      </div>
    </div>

    <div v-else class="loading">
      <el-skeleton :rows="4" animated />
    </div>
  </el-card>
</template>

<script setup lang="ts">
import type { SRLevels } from '../services/api'

defineProps<{
  srLevels?: SRLevels
}>()
</script>

<style scoped>
.sr-panel {
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

.sr-content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.sr-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  padding-bottom: 4px;
  border-bottom: 2px solid;
}

.resistance-title {
  color: #ef5350;
  border-color: #ef5350;
}

.support-title {
  color: #26a69a;
  border-color: #26a69a;
}

.levels {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.level-item {
  display: grid;
  grid-template-columns: 100px 1fr 50px;
  align-items: center;
  gap: 12px;
}

.price {
  font-size: 16px;
  font-weight: 700;
  color: #fff;
}

.strength {
  font-size: 14px;
  font-weight: 600;
  color: #999;
  text-align: right;
}

.no-data {
  text-align: center;
  color: #666;
  padding: 20px 0;
  font-size: 14px;
}

.loading {
  padding: 20px 0;
}
</style>
