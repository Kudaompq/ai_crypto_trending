import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api, type AnalysisResult, type Candle, type KlineData, type PriceQuote, type PriceStreamStatus } from '../services/api'

const storageKey = 'crypto-trending-custom-symbols'
const removedPresetsStorageKey = 'crypto-trending-removed-presets'
const watchlistOrderStorageKey = 'crypto-trending-watchlist-order'
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

function readWatchlistOrder(): string[] {
    try {
        const parsed: unknown = JSON.parse(localStorage.getItem(watchlistOrderStorageKey) || '[]')
        if (!Array.isArray(parsed)) return []
        return [...new Set(parsed.filter((item): item is string =>
            typeof item === 'string' && /^[A-Z0-9]{5,30}$/.test(item)
        ))]
    } catch {
        return []
    }
}

export const useAnalysisStore = defineStore('analysis', () => {
    const customSymbols = ref<string[]>(readCustomSymbols())
    const removedPresetSymbols = ref<string[]>(readRemovedPresetSymbols())
    const watchlistOrder = ref(readWatchlistOrder())
    const cachedAvailableSymbols = computed(() => {
        const visiblePresets = presets.filter(item => !removedPresetSymbols.value.includes(item.value))
        const custom = customSymbols.value.map(value => ({ label: value, value, icon: '☆' }))
        const available = [...visiblePresets, ...custom]
        const byValue = new Map(available.map(item => [item.value, item]))
        const orderedValues = watchlistOrder.value.filter(value => byValue.has(value))
        for (const item of available) {
            if (!orderedValues.includes(item.value)) orderedValues.push(item.value)
        }
        return orderedValues.map(value => byValue.get(value)!)
    })
    const serverSymbols = ref<string[] | null>(null)
    const watchlistRevision = ref(0)
    const availableSymbols = computed(() => {
        const values = serverSymbols.value ?? cachedAvailableSymbols.value.map(item => item.value)
        return values.map(value => presets.find(item => item.value === value) ?? { label: value, value, icon: '☆' })
    })
    const storageWarning = ref<string | null>(null)
    const watchlistReady = ref(false)
    const watchlistSynced = ref(false)
    let initializationPromise: Promise<void> | null = null

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

    function removeLegacyWatchlistKeys() {
        try {
            localStorage.removeItem(storageKey)
            localStorage.removeItem(removedPresetsStorageKey)
            localStorage.removeItem(watchlistOrderStorageKey)
        } catch {
            storageWarning.value = 'Watchlist 已同步，但浏览器旧缓存未能清除'
        }
    }

    function applyWatchlistSnapshot(snapshot: { symbols: string[]; revision: number; legacy_import_pending: boolean }) {
        const symbols = [...new Set(snapshot.symbols.map(value => value.trim().toUpperCase()).filter(Boolean))]
        if (symbols.length === 0) throw new Error('服务端返回了空 Watchlist')
        serverSymbols.value = symbols
        watchlistRevision.value = snapshot.revision
        watchlistSynced.value = true
        customSymbols.value = symbols.filter(value => !presets.some(item => item.value === value))
        removedPresetSymbols.value = []
        watchlistOrder.value = symbols
        if (!symbols.includes(symbol.value)) setSymbol(symbols.includes('ETHUSDT') ? 'ETHUSDT' : symbols[0]!)
    }

    async function fetchAndApplyWatchlist() {
        let snapshot = await api.getWatchlist()
        if (snapshot.legacy_import_pending) {
            snapshot = await api.importLegacyWatchlist(cachedAvailableSymbols.value.map(item => item.value))
        }
        applyWatchlistSnapshot(snapshot)
        removeLegacyWatchlistKeys()
    }

    function initializeWatchlist(): Promise<void> {
        if (initializationPromise) return initializationPromise
        initializationPromise = (async () => {
            try {
                await fetchAndApplyWatchlist()
                storageWarning.value = null
            } catch {
                watchlistSynced.value = false
                storageWarning.value = 'Watchlist 数据库同步失败，当前显示本地缓存；同步恢复前不能修改列表'
            } finally {
                watchlistReady.value = true
            }
        })()
        return initializationPromise
    }

    function requireSyncedWatchlist() {
        if (!watchlistReady.value || !watchlistSynced.value || !serverSymbols.value) {
            throw new Error('Watchlist 尚未与服务端同步，暂时不能修改')
        }
    }

    function noteMutationFailure(error: unknown) {
        const status = (error as { response?: { status?: number } } | null)?.response?.status
        if (status === undefined || status >= 500) {
            storageWarning.value = 'Watchlist 未能同步到数据库，本次修改未提交'
        }
    }

    async function addCustomSymbol(raw: string): Promise<string> {
        requireSyncedWatchlist()
        const normalized = raw.trim().toUpperCase()
        if (availableSymbols.value.some(item => item.value === normalized)) {
            throw new Error('该交易对已在列表中')
        }

        try {
            const snapshot = await api.addWatchlistSymbol(raw)
            applyWatchlistSnapshot(snapshot)
            storageWarning.value = null
            return normalized
        } catch (error: unknown) {
            noteMutationFailure(error)
            throw error
        }
    }

    async function removeSymbol(value: string) {
        if (availableSymbols.value.length <= 1 || !availableSymbols.value.some(item => item.value === value)) return

        requireSyncedWatchlist()
        try {
            const snapshot = await api.removeWatchlistSymbol(value)
            applyWatchlistSnapshot(snapshot)
            storageWarning.value = null
        } catch (error: unknown) {
            noteMutationFailure(error)
            throw error
        }
    }

    async function reorderSymbols(value: string, targetIndex: number) {
        const order = availableSymbols.value.map(item => item.value)
        const sourceIndex = order.indexOf(value)
        if (sourceIndex < 0 || !Number.isInteger(targetIndex) || targetIndex < 0 || targetIndex >= order.length) return
        order.splice(sourceIndex, 1)
        order.splice(Math.min(targetIndex, order.length), 0, value)
        if (order.every((item, index) => item === availableSymbols.value[index]?.value)) return

        try {
            requireSyncedWatchlist()
            const snapshot = await api.reorderWatchlist(watchlistRevision.value, order)
            applyWatchlistSnapshot(snapshot)
            storageWarning.value = null
        } catch (error: unknown) {
            const status = (error as { response?: { status?: number } } | null)?.response?.status
            if (status === 409) {
                try {
                    await fetchAndApplyWatchlist()
                    storageWarning.value = 'Watchlist 已被其他客户端修改，已加载最新列表，请重新排序'
                } catch {
                    watchlistSynced.value = false
                    storageWarning.value = 'Watchlist 排序冲突，且最新列表同步失败'
                }
                return
            }
            noteMutationFailure(error)
        }
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
            klineData.value = { symbol: symbol.value, interval: interval.value, data: [candle], has_more_before: false }
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
        availableSymbols, customSymbols, storageWarning, watchlistReady, watchlistSynced, initializeWatchlist, symbol, interval, limit, loading, error, invalidSymbol,
        klineData, analysisResult, lastUpdate, trendDirection, trendStrength, trendColor,
        pricesBySymbol, unavailableSymbols, priceStreamState, startWatchlistPriceStream, stopWatchlistPriceStream,
        addCustomSymbol, removeSymbol, fetchAnalysis, fetchFallbackKline,
        updateRealtimeCandle, setSymbol, setInterval, setLimit, reorderSymbols
    }
})
