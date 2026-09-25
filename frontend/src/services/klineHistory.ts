import type { Candle } from './api'

export type KlineChartPeriodType = 'second' | 'minute' | 'hour' | 'day' | 'week' | 'month' | 'year'

export interface KlineHistoryPage {
  data: Candle[]
  has_more_before: boolean
}

export interface KlineHistoryRequest {
  symbol: string
  interval: string
  limit: number
  endTime?: number
}

export type KlineHistoryFetcher = (request: KlineHistoryRequest, signal?: AbortSignal) => Promise<KlineHistoryPage>

export const BINANCE_KLINE_INTERVALS = [
  '1m', '3m', '5m', '15m', '30m', '1h', '2h', '4h', '6h', '8h', '12h', '1d', '3d', '1w', '1M'
] as const

const intervalPeriods: Record<string, { span: number; type: KlineChartPeriodType }> = {
  '1m': { span: 1, type: 'minute' }, '3m': { span: 3, type: 'minute' },
  '5m': { span: 5, type: 'minute' }, '15m': { span: 15, type: 'minute' },
  '30m': { span: 30, type: 'minute' }, '1h': { span: 1, type: 'hour' },
  '2h': { span: 2, type: 'hour' }, '4h': { span: 4, type: 'hour' },
  '6h': { span: 6, type: 'hour' }, '8h': { span: 8, type: 'hour' },
  '12h': { span: 12, type: 'hour' }, '1d': { span: 1, type: 'day' },
  '3d': { span: 3, type: 'day' }, '1w': { span: 1, type: 'week' },
  '1M': { span: 1, type: 'month' }
}

export function toKlineChartPeriod(interval: string): { span: number; type: KlineChartPeriodType } {
  const period = intervalPeriods[interval]
  if (!period) throw new Error(`Unsupported Binance K-line interval: ${interval}`)
  return period
}

function mergeCandles(existing: Candle[], incoming: Candle[]): Candle[] {
  const byTimestamp = new Map<number, Candle>()
  for (const candle of incoming) byTimestamp.set(candle.timestamp, candle)
  // Current data wins at a page boundary so an already-received live update is not replaced.
  for (const candle of existing) byTimestamp.set(candle.timestamp, candle)
  return [...byTimestamp.values()].sort((left, right) => left.timestamp - right.timestamp)
}

export class KlineHistory {
  candles: Candle[] = []
  hasMoreBefore = true
  private generation = 0
  private symbol = ''
  private interval = ''
  private loading = false
  private controllers = new Set<AbortController>()
  private readonly fetchPage: KlineHistoryFetcher

  constructor(fetchPage: KlineHistoryFetcher) {
    this.fetchPage = fetchPage
  }

  private abortRequests() {
    for (const controller of this.controllers) controller.abort()
    this.controllers.clear()
    this.loading = false
  }

  reset(symbol: string, interval: string) {
    this.generation++
    this.abortRequests()
    this.symbol = symbol
    this.interval = interval
    this.candles = []
    this.hasMoreBefore = true
  }

  seed(symbol: string, interval: string, candles: Candle[], hasMoreBefore: boolean) {
    this.generation++
    this.abortRequests()
    this.symbol = symbol
    this.interval = interval
    this.candles = mergeCandles([], candles)
    this.hasMoreBefore = hasMoreBefore && this.candles.length > 0
  }

  private async fetch(request: KlineHistoryRequest, generation: number): Promise<KlineHistoryPage | null> {
    const controller = new AbortController()
    this.controllers.add(controller)
    try {
      const page = await this.fetchPage(request, controller.signal)
      return generation === this.generation ? page : null
    } finally {
      this.controllers.delete(controller)
    }
  }

  async loadInitial(symbol: string, interval: string, limit = 100): Promise<Candle[]> {
    this.reset(symbol, interval)
    const generation = this.generation
    this.loading = true
    try {
      const page = await this.fetch({ symbol, interval, limit }, generation)
      if (!page || generation !== this.generation) return this.candles
      this.candles = mergeCandles([], page.data)
      this.hasMoreBefore = page.has_more_before && this.candles.length > 0
      return this.candles
    } finally {
      if (generation === this.generation) this.loading = false
    }
  }

  async loadOlder(symbol = this.symbol, interval = this.interval, limit = 100): Promise<Candle[]> {
    if (this.loading || !this.hasMoreBefore || this.candles.length === 0) return this.candles
    if (symbol !== this.symbol || interval !== this.interval) return this.candles
    const earliest = this.candles[0]!.timestamp
    if (earliest <= 1) {
      this.hasMoreBefore = false
      return this.candles
    }
    const generation = this.generation
    this.loading = true
    const knownTimestamps = new Set(this.candles.map(candle => candle.timestamp))
    try {
      const page = await this.fetch({ symbol, interval, limit, endTime: earliest - 1 }, generation)
      if (!page || generation !== this.generation) return this.candles
      const olderBars = page.data
        .filter(candle => candle.timestamp < earliest && !knownTimestamps.has(candle.timestamp))
        .sort((left, right) => left.timestamp - right.timestamp)
      this.candles = mergeCandles(this.candles, page.data)
      this.hasMoreBefore = page.has_more_before && (this.candles[0]?.timestamp ?? earliest) < earliest
      return olderBars
    } finally {
      if (generation === this.generation) this.loading = false
    }
  }
}
