import type { Candle } from './api'
import {
  BINANCE_KLINE_INTERVALS,
  toKlineChartPeriod,
  type KlineHistoryPage,
  type KlineHistoryRequest
} from './klineHistory'

export interface BinanceProSymbol {
  ticker: string
  name: string
  shortName: string
  exchange: string
  market: string
  pricePrecision: number
  volumePrecision: number
}

export interface BinanceProPeriod {
  multiplier: number
  timespan: string
  text: string
}

export interface BinanceProDatafeedOptions {
  getSupportedSymbols: () => string[]
  fetchHistory: (request: KlineHistoryRequest, signal?: AbortSignal) => Promise<KlineHistoryPage>
  onHistoryError?: (request: { symbol: string; interval: string }, error: unknown) => void
  onHistoryLoaded?: (request: { symbol: string; interval: string }, page: KlineHistoryPage) => void
  onSelectionRequest?: (selection: { symbol: string; interval: string }) => void
}

export interface BinanceProDatafeed {
  searchSymbols(search?: string): Promise<BinanceProSymbol[]>
  getHistoryKLineData(symbol: BinanceProSymbol, period: BinanceProPeriod, from: number, to: number): Promise<Candle[]>
  subscribe(symbol: BinanceProSymbol, period: BinanceProPeriod, callback: (candle: Candle) => void): void
  unsubscribe(symbol: BinanceProSymbol, period: BinanceProPeriod): void
  hasMoreHistory(symbol: BinanceProSymbol, period: BinanceProPeriod): boolean
  publishCandle(symbol: string, interval: string, candle: Candle): void
  dispose(): void
}

export const BINANCE_PRO_PERIODS: BinanceProPeriod[] = BINANCE_KLINE_INTERVALS.map(text => {
  const period = toKlineChartPeriod(text)
  return { text, multiplier: period.span, timespan: period.type }
})

const periodsByText = new Map(BINANCE_PRO_PERIODS.map(period => [period.text, period]))

function scopeKey(symbol: string, interval: string): string {
  return `${symbol.toUpperCase()}:${interval}`
}

function resolvePeriod(period: BinanceProPeriod): string {
  const configured = periodsByText.get(period.text)
  if (!configured || configured.multiplier !== period.multiplier || configured.timespan !== period.timespan) {
    throw new Error(`Unsupported Binance Pro period: ${period.text}`)
  }
  return configured.text
}

function supportedSymbol(ticker: string): BinanceProSymbol {
  return {
    ticker,
    name: ticker,
    shortName: ticker,
    exchange: 'Binance',
    market: 'futures',
    pricePrecision: 2,
    volumePrecision: 2
  }
}

function inRange(candle: Candle, from: number, to: number): boolean {
  return Number.isFinite(candle.timestamp) && candle.timestamp >= from && candle.timestamp <= to
}

export function createBinanceProDatafeed(options: BinanceProDatafeedOptions): BinanceProDatafeed {
  let generation = 0
  let activeScope = ''
  let hasMore = true
  let disposed = false
  const controllers = new Set<AbortController>()
  const subscriptions = new Map<string, Set<(candle: Candle) => void>>()

  function abortRequests() {
    for (const controller of controllers) controller.abort()
    controllers.clear()
  }

  function activateScope(symbol: string, interval: string): number {
    if (disposed) return generation
    const nextScope = scopeKey(symbol, interval)
    if (nextScope !== activeScope) {
      generation++
      activeScope = nextScope
      hasMore = true
      abortRequests()
      subscriptions.clear()
    }
    return generation
  }

  function supportedTickers(): string[] {
    return [...new Set(options.getSupportedSymbols()
      .map(symbol => symbol.trim().toUpperCase())
      .filter(symbol => /^[A-Z0-9]{5,30}$/.test(symbol)))]
  }

  return {
    async searchSymbols(search = '') {
      if (disposed) return []
      const query = search.trim().toUpperCase()
      return supportedTickers()
        .filter(ticker => !query || ticker.includes(query))
        .map(supportedSymbol)
    },

    async getHistoryKLineData(symbol, period, from, to) {
      if (disposed || !Number.isFinite(from) || !Number.isFinite(to) || from > to) return []
      const interval = resolvePeriod(period)
      const ticker = symbol.ticker.trim().toUpperCase()
      if (!supportedTickers().includes(ticker)) return []

      options.onSelectionRequest?.({ symbol: ticker, interval })
      const requestGeneration = activateScope(ticker, interval)
      const controller = new AbortController()
      controllers.add(controller)
      try {
        const page = await options.fetchHistory({
          symbol: ticker,
          interval,
          limit: 500,
          endTime: to
        }, controller.signal)
        if (disposed || controller.signal.aborted || requestGeneration !== generation || activeScope !== scopeKey(ticker, interval)) {
          return []
        }
        hasMore = page.has_more_before
        options.onHistoryLoaded?.({ symbol: ticker, interval }, page)

        const unique = new Map<number, Candle>()
        for (const candle of page.data) {
          if (inRange(candle, from, to) && !unique.has(candle.timestamp)) unique.set(candle.timestamp, candle)
        }
        return [...unique.values()].sort((left, right) => left.timestamp - right.timestamp)
      } catch (error: unknown) {
        if (!controller.signal.aborted && !disposed && requestGeneration === generation) {
          hasMore = true
          options.onHistoryError?.({ symbol: ticker, interval }, error)
        }
        return []
      } finally {
        controllers.delete(controller)
      }
    },

    subscribe(symbol, period, callback) {
      if (disposed) return
      const interval = resolvePeriod(period)
      const ticker = symbol.ticker.trim().toUpperCase()
      if (!supportedTickers().includes(ticker)) return
      activateScope(ticker, interval)
      const key = scopeKey(ticker, interval)
      const callbacks = subscriptions.get(key) ?? new Set()
      callbacks.add(callback)
      subscriptions.set(key, callbacks)
    },

    unsubscribe(symbol, period) {
      const interval = resolvePeriod(period)
      const key = scopeKey(symbol.ticker, interval)
      subscriptions.delete(key)
    },

    hasMoreHistory(symbol, period) {
      if (disposed) return false
      const interval = resolvePeriod(period)
      return activeScope === scopeKey(symbol.ticker, interval) ? hasMore : true
    },

    publishCandle(symbol, interval, candle) {
      if (disposed || activeScope !== scopeKey(symbol, interval)) return
      for (const callback of subscriptions.get(activeScope) ?? []) callback(candle)
    },

    dispose() {
      if (disposed) return
      disposed = true
      generation++
      activeScope = ''
      hasMore = false
      abortRequests()
      subscriptions.clear()
    }
  }
}
