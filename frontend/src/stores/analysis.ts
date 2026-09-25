import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api, type AnalysisResult, type Candle, type KlineData } from '../services/api'

const storageKey = 'crypto-trending-custom-symbols'
const presets = [
    { label: 'BTC/USDT', value: 'BTCUSDT', icon: '₿' },
    { label: 'ETH/USDT', value: 'ETHUSDT', icon: 'Ξ' },
    { label: 'BNB/USDT', value: 'BNBUSDT', icon: '🔶' },
    { label: 'SOL/USDT', value: 'SOLUSDT', icon: '◎' },
    { label: 'XRP/USDT', value: 'XRPUSDT', icon: '✕' },
    { label: 'ADA/USDT', value: 'ADAUSDT', icon: '₳' },
    { label: 'DOGE/USDT', value: 'DOGEUSDT', icon: 'Ð' },
    { label: 'MATIC/USDT', value: 'MATICUSDT', icon: '⬡' }
]

function readCustomSymbols(): string[] {
    try {
        const parsed: unknown = JSON.parse(localStorage.getItem(storageKey) || '[]')
        if (!Array.isArray(parsed)) return []
        const presetValues = new Set(presets.map(item => item.value))
        return [...new Set(parsed.filter((item): item is string =>
            typeof item === 'string' && /^[A-Z0-9]{5,30}$/.test(item) && !presetValues.has(item)
        ))]
    } catch {
        return []
    }
}

export const useAnalysisStore = defineStore('analysis', () => {
    const customSymbols = ref<string[]>(readCustomSymbols())
    const availableSymbols = computed(() => [
        ...presets,
        ...customSymbols.value.map(value => ({ label: value, value, icon: '☆' }))
    ])
    const storageWarning = ref<string | null>(null)

    const symbol = ref('ETHUSDT')
    const interval = ref('1d')
    const limit = ref(100)
    const loading = ref(false)
    const error = ref<string | null>(null)
    const invalidSymbol = ref(false)
    const klineData = ref<KlineData | null>(null)
    const analysisResult = ref<AnalysisResult | null>(null)
    const lastUpdate = ref<Date | null>(null)

    let marketVersion = 0
    let requestVersion = 0
    let priceVersion = 0

    const trendDirection = computed(() => analysisResult.value?.trend.direction || '加载中...')
    const trendStrength = computed(() => analysisResult.value?.trend.strength || 0)
    const trendColor = computed(() => {
        const dir = trendDirection.value
        if (dir === '上升') return '#26a69a'
        if (dir === '下降') return '#ef5350'
        return '#ffa726'
    })

    function persistSymbols() {
        try {
            localStorage.setItem(storageKey, JSON.stringify(customSymbols.value))
            storageWarning.value = null
        } catch {
            storageWarning.value = '自选交易对未能保存到浏览器，刷新后可能丢失'
        }
    }

    async function addCustomSymbol(raw: string): Promise<string> {
        const normalized = raw.trim().toUpperCase()
        if (availableSymbols.value.some(item => item.value === normalized)) {
            throw new Error('该交易对已在列表中')
        }
        const validSymbol = await api.validateSymbol(raw)
        if (availableSymbols.value.some(item => item.value === validSymbol)) {
            throw new Error('该交易对已在列表中')
        }
        customSymbols.value.push(validSymbol)
        persistSymbols()
        return validSymbol
    }

    function removeCustomSymbol(value: string) {
        if (!customSymbols.value.includes(value)) return
        customSymbols.value = customSymbols.value.filter(item => item !== value)
        persistSymbols()
        if (symbol.value === value) setSymbol('ETHUSDT')
    }

    function resetMarket() {
        marketVersion++
        requestVersion++
        klineData.value = null
        analysisResult.value = null
        lastUpdate.value = null
        loading.value = false
        error.value = null
        invalidSymbol.value = false
    }

    function setSymbol(newSymbol: string) {
        if (symbol.value === newSymbol) return
        symbol.value = newSymbol
        resetMarket()
    }

    function setInterval(newInterval: string) {
        if (interval.value === newInterval) return
        interval.value = newInterval
        resetMarket()
    }

    function setLimit(newLimit: number) {
        limit.value = newLimit
    }

    async function fetchAnalysis() {
        const version = marketVersion
        const request = ++requestVersion
        const selectedSymbol = symbol.value
        const selectedInterval = interval.value
        const priceAtStart = priceVersion
        loading.value = true
        error.value = null
        invalidSymbol.value = false

        try {
            const [kline, analysis] = await Promise.all([
                api.getKlineData(selectedSymbol, selectedInterval, limit.value),
                api.getAnalysis(selectedSymbol, selectedInterval, limit.value)
            ])
            if (version !== marketVersion || request !== requestVersion) return
            if (priceAtStart === priceVersion || !klineData.value) klineData.value = kline
            analysisResult.value = analysis
            lastUpdate.value = new Date()
        } catch (err: unknown) {
            if (version === marketVersion && request === requestVersion) {
                error.value = api.errorMessage(err, '获取数据失败')
                invalidSymbol.value = api.invalidSelection(err)
            }
        } finally {
            if (version === marketVersion && request === requestVersion) loading.value = false
        }
    }

    async function fetchFallbackKline(): Promise<boolean> {
        const version = marketVersion
        const priceAtStart = priceVersion
        try {
            const data = await api.getKlineData(symbol.value, interval.value, limit.value)
            if (version !== marketVersion || priceAtStart !== priceVersion) return false
            klineData.value = data
            priceVersion++
            lastUpdate.value = new Date()
            return true
        } catch (err: unknown) {
            if (version === marketVersion) error.value = api.errorMessage(err, '备用行情获取失败')
            return false
        }
    }

    function updateRealtimeCandle(candle: Candle) {
        if (!klineData.value) {
            klineData.value = { symbol: symbol.value, interval: interval.value, data: [candle] }
            priceVersion++
            lastUpdate.value = new Date()
            return
        }
        const candles = klineData.value.data
        const lastIndex = candles.length - 1
        const lastCandle = candles[lastIndex]
        if (lastCandle && lastCandle.timestamp === candle.timestamp) {
            candles[lastIndex] = candle
        } else if (!lastCandle || lastCandle.timestamp < candle.timestamp) {
            candles.push(candle)
            if (candles.length > limit.value) candles.splice(0, candles.length - limit.value)
        }
        priceVersion++
        lastUpdate.value = new Date()
    }

    return {
        availableSymbols, customSymbols, storageWarning, symbol, interval, limit, loading, error, invalidSymbol,
        klineData, analysisResult, lastUpdate, trendDirection, trendStrength, trendColor,
        addCustomSymbol, removeCustomSymbol, fetchAnalysis, fetchFallbackKline,
        updateRealtimeCandle, setSymbol, setInterval, setLimit
    }
})
