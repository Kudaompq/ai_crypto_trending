import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api, type AnalysisResult, type KlineData } from '../services/api'

export const useAnalysisStore = defineStore('analysis', () => {
    // State
    const symbol = ref('ETHUSDT')
    const interval = ref('1d')
    const limit = ref(100)
    const loading = ref(false)
    const error = ref<string | null>(null)

    const klineData = ref<KlineData | null>(null)
    const analysisResult = ref<AnalysisResult | null>(null)
    const lastUpdate = ref<Date | null>(null)

    // Computed
    const trendDirection = computed(() => analysisResult.value?.trend.direction || '加载中...')
    const trendStrength = computed(() => analysisResult.value?.trend.strength || 0)
    const trendColor = computed(() => {
        const dir = trendDirection.value
        if (dir === '上升') return '#26a69a'
        if (dir === '下降') return '#ef5350'
        return '#ffa726'
    })

    // Actions
    async function fetchAnalysis() {
        loading.value = true
        error.value = null

        try {
            const [kline, analysis] = await Promise.all([
                api.getKlineData(symbol.value, interval.value, limit.value),
                api.getAnalysis(symbol.value, interval.value, limit.value)
            ])

            klineData.value = kline
            analysisResult.value = analysis
            lastUpdate.value = new Date()
        } catch (err: any) {
            error.value = err.message || '获取数据失败'
            console.error('Failed to fetch analysis:', err)
        } finally {
            loading.value = false
        }
    }

    function setSymbol(newSymbol: string) {
        symbol.value = newSymbol
    }

    function setInterval(newInterval: string) {
        interval.value = newInterval
    }

    function setLimit(newLimit: number) {
        limit.value = newLimit
    }

    return {
        // State
        symbol,
        interval,
        limit,
        loading,
        error,
        klineData,
        analysisResult,
        lastUpdate,

        // Computed
        trendDirection,
        trendStrength,
        trendColor,

        // Actions
        fetchAnalysis,
        setSymbol,
        setInterval,
        setLimit
    }
})
