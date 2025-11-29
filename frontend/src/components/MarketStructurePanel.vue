<template>
  <el-card class="market-structure-panel" shadow="hover">
    <template #header>
      <div class="card-header">
        <span class="title">🏗️ 市场结构</span>
        <el-tag :type="getRiskType()" size="large">
          风险: {{ structure?.risk_level || '未知' }}
        </el-tag>
      </div>
    </template>

    <div v-if="structure" class="structure-content">
      <div class="structure-grid">
        <div class="structure-item">
          <div class="icon">{{ structure.higher_high ? '✅' : '❌' }}</div>
          <div class="info">
            <div class="label">Higher High</div>
            <div class="status">{{ structure.higher_high ? '是' : '否' }}</div>
          </div>
        </div>

        <div class="structure-item">
          <div class="icon">{{ structure.higher_low ? '✅' : '❌' }}</div>
          <div class="info">
            <div class="label">Higher Low</div>
            <div class="status">{{ structure.higher_low ? '是' : '否' }}</div>
          </div>
        </div>

        <div class="structure-item">
          <div class="icon">{{ structure.structure_break ? '⚠️' : '✅' }}</div>
          <div class="info">
            <div class="label">结构破位</div>
            <div class="status" :class="structure.structure_break ? 'warning' : 'safe'">
              {{ structure.structure_break ? '是' : '否' }}
            </div>
          </div>
        </div>
      </div>

      <div class="risk-analysis">
        <div class="risk-title">风险分析</div>
        <div class="risk-description">
          <template v-if="structure.risk_level === '低'">
            当前市场结构健康，风险较低。{{ getStructureDescription() }}
          </template>
          <template v-else-if="structure.risk_level === '中'">
            市场结构出现变化，需要密切关注。{{ getStructureDescription() }}
          </template>
          <template v-else>
            市场结构破位，风险较高，建议谨慎操作。{{ getStructureDescription() }}
          </template>
        </div>
      </div>
    </div>

    <div v-else class="loading">
      <el-skeleton :rows="3" animated />
    </div>
  </el-card>
</template>

<script setup lang="ts">
import type { MarketStructure } from '../services/api'

const props = defineProps<{
  structure?: MarketStructure
}>()

function getRiskType(): 'success' | 'warning' | 'danger' {
  if (!props.structure) return 'warning'
  if (props.structure.risk_level === '低') return 'success'
  if (props.structure.risk_level === '中') return 'warning'
  return 'danger'
}

function getStructureDescription(): string {
  if (!props.structure) return ''
  
  const { higher_high, higher_low, structure_break } = props.structure
  
  if (higher_high && higher_low) {
    return '上升结构完好，可持续关注多头机会。'
  } else if (!higher_high && !higher_low) {
    return '下降结构明显，注意风险控制。'
  } else if (structure_break) {
    return '结构已被破坏，趋势可能反转。'
  }
  
  return '市场处于震荡整理阶段。'
}
</script>

<style scoped>
.market-structure-panel {
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

.structure-content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.structure-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 16px;
}

.structure-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: rgba(255, 255, 255, 0.05);
  border-radius: 8px;
}

.icon {
  font-size: 32px;
}

.info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.label {
  font-size: 12px;
  color: #999;
}

.status {
  font-size: 16px;
  font-weight: 600;
  color: #fff;
}

.status.warning {
  color: #ef5350;
}

.status.safe {
  color: #26a69a;
}

.risk-analysis {
  padding: 16px;
  background: rgba(255, 167, 38, 0.1);
  border-left: 4px solid #ffa726;
  border-radius: 8px;
}

.risk-title {
  font-size: 14px;
  font-weight: 600;
  color: #ffa726;
  margin-bottom: 8px;
}

.risk-description {
  font-size: 14px;
  color: #ddd;
  line-height: 1.6;
}

.loading {
  padding: 20px 0;
}
</style>
