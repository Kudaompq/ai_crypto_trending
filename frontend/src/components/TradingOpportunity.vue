<template>
    <div class="trading-opportunities">
        <div class="opportunities-header" v-if="!hideHeader">
            <h3>🎯 交易机会</h3>
            <div class="summary" v-if="opportunities && opportunities.length > 0">
                <span class="summary-item">
                    <span class="label">总数:</span>
                    <span class="value">{{ summary.total_opportunities }}</span>
                </span>
                <span class="summary-item">
                    <span class="label">平均盈亏比:</span>
                    <span class="value highlight">{{ summary.avg_risk_reward.toFixed(2) }}:1</span>
                </span>
                <span class="summary-item">
                    <span class="label">高置信度:</span>
                    <span class="value">{{ summary.high_confidence_count }}</span>
                </span>
            </div>
        </div>

        <div v-if="loading" class="loading">
            <div class="spinner"></div>
            <p>正在分析市场机会...</p>
        </div>

        <div v-else-if="!opportunities || opportunities.length === 0" class="no-opportunities">
            <div class="icon">📊</div>
            <p class="message">当前暂无符合条件的交易机会</p>
            <p class="hint">系统只在检测到高质量设置时才会推荐 (盈亏比 ≥ 2:1)</p>
        </div>

        <div v-else class="opportunities-list">
            <div v-for="opp in opportunities" :key="opp.id" class="opportunity-card" :class="opp.type.toLowerCase()">
                <!-- Header -->
                <div class="card-header">
                    <div class="type-badge" :class="opp.type.toLowerCase()">
                        {{ opp.type === 'LONG' ? '🟢 做多' : '🔴 做空' }}
                    </div>
                    <div class="strategy">{{ getStrategyName(opp.strategy) }}</div>
                    <div class="rr-badge" :class="getRRClass(opp.risk_reward.ratio)">
                        R:R {{ opp.risk_reward.ratio.toFixed(1) }}:1
                    </div>
                </div>

                <!-- Price Levels -->
                <div class="price-levels">
                    <div class="price-row entry">
                        <span class="label">入场:</span>
                        <span class="price">${{ opp.entry.price.toFixed(2) }}</span>
                    </div>
                    <div class="price-row stop">
                        <span class="label">止损:</span>
                        <span class="price">${{ opp.stop_loss.price.toFixed(2) }}</span>
                        <span class="pct negative">-{{ opp.stop_loss.distance_pct.toFixed(2) }}%</span>
                    </div>
                    <div class="price-row target" v-for="tp in opp.take_profit" :key="tp.level">
                        <span class="label">目标{{ tp.level }}:</span>
                        <span class="price">${{ tp.price.toFixed(2) }}</span>
                        <span class="pct positive">+{{ tp.distance_pct.toFixed(2) }}%</span>
                        <span class="close-pct">(平{{ tp.position_close_pct }}%)</span>
                    </div>
                </div>

                <!-- Confidence -->
                <div class="confidence-section">
                    <div class="confidence-header">
                        <span class="label">置信度:</span>
                        <span class="score" :class="opp.confidence.level.toLowerCase()">
                            {{ getConfidenceStars(opp.confidence.score) }} {{ opp.confidence.score }}/100
                        </span>
                        <span class="level-badge" :class="opp.confidence.level.toLowerCase()">
                            {{ getConfidenceLevelText(opp.confidence.level) }}
                        </span>
                    </div>
                </div>

                <!-- Reasons -->
                <div class="reasons-section">
                    <div class="reasons-header">入场理由:</div>
                    <ul class="reasons-list">
                        <li v-for="(reason, idx) in opp.entry.reasons" :key="idx">{{ reason }}</li>
                    </ul>
                </div>

                <!-- Validity -->
                <div class="validity-section">
                    <span class="validity-label">有效期:</span>
                    <span class="expiry">{{ getExpiryText(opp.validity.expires_at) }}</span>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import type { TradingOpportunity, OpportunitySummary } from '../services/api'
import { api } from '../services/api'

const props = defineProps<{
    symbol: string
    interval: string
    hideHeader?: boolean
}>()

const emit = defineEmits<{
    (e: 'opportunities-updated', count: number): void
}>()

const opportunities = ref<TradingOpportunity[]>([])
const summary = ref<OpportunitySummary>({
    total_opportunities: 0,
    avg_risk_reward: 0,
    high_confidence_count: 0
})
const loading = ref(false)
let refreshInterval: number | null = null

const fetchOpportunities = async () => {
    try {
        loading.value = true
        const response = await api.getOpportunities(props.symbol, props.interval, 2.0)
        opportunities.value = response.opportunities
        summary.value = response.summary

        // Emit event to parent
        emit('opportunities-updated', response.opportunities.length)
    } catch (error) {
        console.error('Failed to fetch opportunities:', error)
    } finally {
        loading.value = false
    }
}

const getStrategyName = (strategy: string): string => {
    const names: Record<string, string> = {
        'SUPPORT_BOUNCE': '支撑位反弹',
        'BREAKOUT_RETEST': '突破回踩',
        'TREND_CONTINUATION': '趋势延续'
    }
    return names[strategy] || strategy
}

const getRRClass = (ratio: number): string => {
    if (ratio >= 5) return 'excellent'
    if (ratio >= 4) return 'great'
    if (ratio >= 3) return 'good'
    return 'fair'
}

const getConfidenceStars = (score: number): string => {
    const stars = Math.round(score / 20)
    return '⭐'.repeat(stars)
}

const getConfidenceLevelText = (level: string): string => {
    const texts: Record<string, string> = {
        'HIGH': '高',
        'MEDIUM': '中',
        'LOW': '低'
    }
    return texts[level] || level
}

const getExpiryText = (expiresAt: number): string => {
    const now = Date.now()
    const diff = expiresAt - now

    if (diff < 0) return '已过期'

    const hours = Math.floor(diff / (1000 * 60 * 60))
    const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60))

    if (hours > 0) {
        return `${hours}小时${minutes}分钟后过期`
    }
    return `${minutes}分钟后过期`
}

onMounted(() => {
    fetchOpportunities()
    // Auto-refresh every 5 minutes
    refreshInterval = window.setInterval(fetchOpportunities, 5 * 60 * 1000)
})

onUnmounted(() => {
    if (refreshInterval) {
        clearInterval(refreshInterval)
    }
})
</script>

<style scoped>
.trading-opportunities {
    background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
    border-radius: 12px;
    padding: 20px;
    box-shadow: 0 4px 6px rgba(0, 0, 0, 0.3);
}

.opportunities-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
    padding-bottom: 15px;
    border-bottom: 2px solid rgba(255, 255, 255, 0.1);
}

.opportunities-header h3 {
    margin: 0;
    font-size: 20px;
    color: #fff;
    font-weight: 700;
}

.summary {
    display: flex;
    gap: 20px;
}

.summary-item {
    display: flex;
    align-items: center;
    gap: 5px;
}

.summary-item .label {
    color: #999;
    font-size: 13px;
}

.summary-item .value {
    color: #fff;
    font-weight: 600;
    font-size: 14px;
}

.summary-item .value.highlight {
    color: #00d9ff;
    font-size: 16px;
}

/* Loading State */
.loading {
    text-align: center;
    padding: 40px 20px;
    color: #999;
}

.spinner {
    width: 40px;
    height: 40px;
    margin: 0 auto 15px;
    border: 4px solid rgba(255, 255, 255, 0.1);
    border-top-color: #00d9ff;
    border-radius: 50%;
    animation: spin 1s linear infinite;
}

@keyframes spin {
    to {
        transform: rotate(360deg);
    }
}

/* No Opportunities State */
.no-opportunities {
    text-align: center;
    padding: 60px 20px;
}

.no-opportunities .icon {
    font-size: 48px;
    margin-bottom: 15px;
}

.no-opportunities .message {
    color: #fff;
    font-size: 16px;
    margin-bottom: 10px;
}

.no-opportunities .hint {
    color: #999;
    font-size: 13px;
}

/* Opportunities List */
.opportunities-list {
    display: flex;
    flex-direction: column;
    gap: 15px;
}

/* Opportunity Card */
.opportunity-card {
    background: rgba(255, 255, 255, 0.05);
    border-radius: 10px;
    padding: 18px;
    border: 2px solid transparent;
    transition: all 0.3s ease;
}

.opportunity-card.long {
    border-color: rgba(38, 166, 154, 0.3);
}

.opportunity-card.short {
    border-color: rgba(239, 83, 80, 0.3);
}

.opportunity-card:hover {
    background: rgba(255, 255, 255, 0.08);
    transform: translateY(-2px);
    box-shadow: 0 6px 12px rgba(0, 0, 0, 0.4);
}

/* Card Header */
.card-header {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 15px;
}

.type-badge {
    padding: 6px 12px;
    border-radius: 6px;
    font-weight: 700;
    font-size: 14px;
}

.type-badge.long {
    background: rgba(38, 166, 154, 0.2);
    color: #26a69a;
}

.type-badge.short {
    background: rgba(239, 83, 80, 0.2);
    color: #ef5350;
}

.strategy {
    color: #ccc;
    font-size: 13px;
    flex: 1;
}

.rr-badge {
    padding: 6px 12px;
    border-radius: 6px;
    font-weight: 700;
    font-size: 15px;
}

.rr-badge.excellent {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: #fff;
}

.rr-badge.great {
    background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
    color: #fff;
}

.rr-badge.good {
    background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
    color: #fff;
}

/* Price Levels */
.price-levels {
    background: rgba(0, 0, 0, 0.2);
    border-radius: 8px;
    padding: 12px;
    margin-bottom: 12px;
}

.price-row {
    display: flex;
    align-items: center;
    padding: 8px 0;
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.price-row:last-child {
    border-bottom: none;
}

.price-row .label {
    color: #999;
    font-size: 13px;
    min-width: 60px;
}

.price-row .price {
    color: #fff;
    font-weight: 700;
    font-size: 15px;
    margin-right: 10px;
}

.price-row .pct {
    font-size: 13px;
    font-weight: 600;
    margin-right: 8px;
}

.price-row .pct.positive {
    color: #26a69a;
}

.price-row .pct.negative {
    color: #ef5350;
}

.price-row .close-pct {
    color: #999;
    font-size: 12px;
    margin-left: auto;
}

.price-row.entry {
    border-left: 3px solid #00d9ff;
    padding-left: 10px;
}

.price-row.stop {
    border-left: 3px solid #ef5350;
    padding-left: 10px;
}

.price-row.target {
    border-left: 3px solid #26a69a;
    padding-left: 10px;
}

/* Confidence Section */
.confidence-section {
    margin-bottom: 12px;
}

.confidence-header {
    display: flex;
    align-items: center;
    gap: 10px;
}

.confidence-header .label {
    color: #999;
    font-size: 13px;
}

.confidence-header .score {
    color: #ffd700;
    font-weight: 700;
    font-size: 14px;
}

.level-badge {
    padding: 4px 10px;
    border-radius: 4px;
    font-size: 12px;
    font-weight: 600;
}

.level-badge.high {
    background: rgba(38, 166, 154, 0.2);
    color: #26a69a;
}

.level-badge.medium {
    background: rgba(255, 193, 7, 0.2);
    color: #ffc107;
}

.level-badge.low {
    background: rgba(239, 83, 80, 0.2);
    color: #ef5350;
}

/* Reasons Section */
.reasons-section {
    margin-bottom: 12px;
}

.reasons-header {
    color: #999;
    font-size: 13px;
    margin-bottom: 8px;
}

.reasons-list {
    list-style: none;
    padding: 0;
    margin: 0;
}

.reasons-list li {
    color: #ccc;
    font-size: 13px;
    padding: 4px 0;
    padding-left: 18px;
    position: relative;
}

.reasons-list li::before {
    content: '•';
    color: #00d9ff;
    font-weight: bold;
    position: absolute;
    left: 0;
}

/* Validity Section */
.validity-section {
    display: flex;
    align-items: center;
    gap: 8px;
    padding-top: 12px;
    border-top: 1px solid rgba(255, 255, 255, 0.05);
}

.validity-label {
    color: #999;
    font-size: 12px;
}

.expiry {
    color: #ffc107;
    font-size: 12px;
    font-weight: 600;
}
</style>
