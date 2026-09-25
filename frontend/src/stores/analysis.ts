import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api, type AnalysisResult, type Candle, type KlineData, type PriceQuote, type PriceStreamStatus } from '../services/api'

const storageKey = 'crypto-trending-custom-symbols'
const removedPresetsStorageKey = 'crypto-trending-removed-presets'
const presets = [
    { label: 'BTC/USDT', value: 'BTCUSDT', icon: '₿' },
    { label: 'ETH/USDT', value: 'ETHUSDT', icon: 'Ξ' },
    { label: 'BNB/USDT', value: 'BNBUSDT', icon: '🔶' },
    { label: 'SOL/USDT', value: 'SOLUSDT', icon: '◎' },
    { label: 'XRP/USDT', value: 'XRPUSDT', icon: '✕' },
    { label: 'ADA/USDT', value: 'ADAUSDT', icon: '₳' },
    { label: 'DOGE/USDT', value: 'DOGEUSDT', icon: 'Ð' },
    { label: 'POL/USDT', value: 'POLUSDT', icon: '⬡' }
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

function readRemovedPresetSymbols(): string[] {
    try {
        const parsed: unknown = JSON.parse(localStorage.getItem(removedPresetsStorageKey) || '[]')
        if (!Array.isArray(parsed)) return []
        const presetValues = new Set(presets.map(item => item.value))
        return [...new Set(parsed.filter((item): item is string =>
            typeof item === 'string' && presetValues.has(item)
        ))]
    } catch {
        return []
    }
}

export const useAnalysisStore = defineStore('analysis', () => {
    const customSymbols = ref<string[]>(readCustomSymbols())
    const removedPresetSymbols = ref<string[]>(readRemovedPresetSymbols())
    const availableSymbols = computed(() => {
        const visiblePresets = presets.filter(item => !removedPresetSymbols.value.includes(item.value))
        const custom = customSymbols.value.map(value => ({ label: value, value, icon: '☆' }))
        const available = [...visiblePresets, ...custom]
        return available.length > 0 ? available : [presets[0]!]
    })
    const storageWarning = ref<string | null>(null)

    const symbol = ref(availableSymbols.value.find(item => item.value === 'ETHUSDT')?.value ?? availableSymbols.value[0]!.value)
    const interval = ref('1d')
    const limit = ref(100)
    const loading = ref(false)
    const error = ref<string | null>(null)
    const invalidSymbol = ref(false)
    const klineData = ref<KlineData | null>(null)
    const analysisResult = ref<AnalysisResult | null>(null)
    const lastUpdate = ref<Date | null>(null)
    const pricesBySymbol = ref<Record<string, { price: number; change24hPercent: number; eventTime: number }>>({})
    const unavailableSymbols = ref<string[]>([])
    const priceStreamState = ref<'connecting' | 'live' | 'reconnecting'>('connecting')

    let marketVersion = 0
    let requestVersion = 0
    let priceVersion = 0
    let priceStream: EventSource | null = null
    let quoteStreamVersion = 0
    let streamEventTimes: Record<string, number> = {}

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
            storageWarning.value = '交易对列表未能保存到浏览器，刷新后可能丢失'
        }
    }

    function persistRemovedPresets() {
        try {
            localStorage.setItem(removedPresetsStorageKey, JSON.stringify(removedPresetSymbols.value))
            storageWarning.value = null
        } catch {
            storageWarning.value = '交易对列表未能保存到浏览器，刷新后可能丢失'
        }
    }

    async function addCustomSymbol(raw: string): Promise<string> {
        const normalized = raw.trim().toUpperCase()
        if (availableSymbols.value.some(item => item.value === normalized)) {
            throw new Error('该交易对已在列表中')
        }
        const removedPreset = presets.find(item => item.value === normalized && removedPresetSymbols.value.includes(item.value))
        if (removedPreset) {
            removedPresetSymbols.value = removedPresetSymbols.value.filter(value => value !== removedPreset.value)
            persistRemovedPresets()
            return removedPreset.value
        }
        const validSymbol = await api.validateSymbol(raw)
        if (availableSymbols.value.some(item => item.value === validSymbol)) {
            throw new Error('该交易对已在列表中')
        }
        const validatedRemovedPreset = presets.find(item => item.value === validSymbol && removedPresetSymbols.value.includes(item.value))
        if (validatedRemovedPreset) {
            removedPresetSymbols.value = removedPresetSymbols.value.filter(value => value !== validatedRemovedPreset.value)
            persistRemovedPresets()
            return validatedRemovedPreset.value
        }
        customSymbols.value.push(validSymbol)
        persistSymbols()
        return validSymbol
    }

    function removeSymbol(value: string) {
        if (availableSymbols.value.length <= 1 || !availableSymbols.value.some(item => item.value === value)) return

        if (presets.some(item => item.value === value)) {
            removedPresetSymbols.value = [...new Set([...removedPresetSymbols.value, value])]
            persistRemovedPresets()
        } else {
            customSymbols.value = customSymbols.value.filter(item => item !== value)
            persistSymbols()
        }

        if (symbol.value === value) setSymbol(availableSymbols.value[0]!.value)
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

    async function startWatchlistPriceStream() {
        const version = ++quoteStreamVersion
        priceStream?.close()
        const symbols = availableSymbols.value.map(item => item.value)
        const symbolSet = new Set(symbols)
        const receivedSinceSnapshot = new Set<string>()
        priceStreamState.value = 'connecting'
        unavailableSymbols.value = []

        priceStream = api.createWatchlistPriceStream(symbols, (event: PriceQuote) => {
            if (version !== quoteStreamVersion || !symbolSet.has(event.symbol)) return
            receivedSinceSnapshot.add(event.symbol)
            const lastStreamTime = streamEventTimes[event.symbol]
            if (lastStreamTime === undefined || event.event_time >= lastStreamTime) {
                streamEventTimes[event.symbol] = event.event_time
                pricesBySymbol.value[event.symbol] = {
                    price: event.price,
                    change24hPercent: event.change_24h_percent,
                    eventTime: event.event_time
                }
            }
            priceStreamState.value = 'live'
        }, (status: PriceStreamStatus) => {
            if (version === quoteStreamVersion) {
                priceStreamState.value = status.state
                if (status.unavailable_symbols) unavailableSymbols.value = status.unavailable_symbols
            }
        })
        priceStream.onerror = () => {
            if (version === quoteStreamVersion) priceStreamState.value = 'reconnecting'
        }

        try {
            const snapshot = await api.getWatchlistPrices(symbols)
            if (version !== quoteStreamVersion) return
            unavailableSymbols.value = snapshot.unavailable_symbols || []
            for (const quote of snapshot.prices) {
                if (!symbolSet.has(quote.symbol) || receivedSinceSnapshot.has(quote.symbol)) continue
                const previous = pricesBySymbol.value[quote.symbol]
                const lastStreamTime = streamEventTimes[quote.symbol]
                if (lastStreamTime === undefined && (!previous || quote.event_time >= previous.eventTime)) {
                    pricesBySymbol.value[quote.symbol] = {
                        price: quote.price,
                        change24hPercent: quote.change_24h_percent,
                        eventTime: quote.event_time
                    }
                }
            }
        } catch {
            if (version === quoteStreamVersion && priceStreamState.value === 'connecting') {
                priceStreamState.value = 'reconnecting'
            }
        }
    }

    function stopWatchlistPriceStream() {
        quoteStreamVersion++
        priceStream?.close()
        priceStream = null
    }

    return {
        availableSymbols, customSymbols, storageWarning, symbol, interval, limit, loading, error, invalidSymbol,
        klineData, analysisResult, lastUpdate, trendDirection, trendStrength, trendColor,
        pricesBySymbol, unavailableSymbols, priceStreamState, startWatchlistPriceStream, stopWatchlistPriceStream,
        addCustomSymbol, removeSymbol, fetchAnalysis, fetchFallbackKline,
        updateRealtimeCandle, setSymbol, setInterval, setLimit
    }
})
