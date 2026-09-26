import { describe, expect, it, vi } from 'vitest'
import { BINANCE_PRO_PERIODS, createBinanceProDatafeed, type BinanceProSymbol } from '../services/binanceProDatafeed'
import type { KlineHistoryRequest } from '../services/klineHistory'

const symbol = (ticker: string): BinanceProSymbol => ({
  ticker, name: ticker, shortName: ticker, exchange: 'Binance', market: 'futures', pricePrecision: 2, volumePrecision: 2
})

const candle = (timestamp: number, close = timestamp) => ({
  timestamp, open: close - 1, high: close + 2, low: close - 2, close, volume: 10
})

describe('Binance Pro Datafeed', () => {
  it('exposes every Binance USDⓈ-M native period without client-side aggregation', () => {
    expect(BINANCE_PRO_PERIODS.map(period => period.text)).toEqual([
      '1m', '3m', '5m', '15m', '30m', '1h', '2h', '4h', '6h', '8h', '12h', '1d', '3d', '1w', '1M'
    ])
    expect(BINANCE_PRO_PERIODS.find(period => period.text === '1M')).toMatchObject({ multiplier: 1, timespan: 'month' })
  })

  it('searches only current supported USDⓈ-M symbols and filters case-insensitively', async () => {
    const feed = createBinanceProDatafeed({ getSupportedSymbols: () => ['BTCUSDT', 'ETHUSDT'], fetchHistory: vi.fn() })

    await expect(feed.searchSymbols('eth')).resolves.toEqual([symbol('ETHUSDT')])
    await expect(feed.searchSymbols('')).resolves.toEqual([symbol('BTCUSDT'), symbol('ETHUSDT')])
  })

  it('reports Pro symbol and period selections so Dashboard can keep the Watchlist and stream in sync', async () => {
    const onSelectionRequest = vi.fn()
    const feed = createBinanceProDatafeed({
      getSupportedSymbols: () => ['BTCUSDT', 'ETHUSDT'],
      fetchHistory: vi.fn().mockResolvedValue({ data: [candle(300_000)], has_more_before: true }),
      onSelectionRequest
    })

    await feed.getHistoryKLineData(symbol('ETHUSDT'), { multiplier: 5, timespan: 'minute', text: '5m' }, 0, 500_000)

    expect(onSelectionRequest).toHaveBeenCalledWith({ symbol: 'ETHUSDT', interval: '5m' })
  })

  it('notifies the OI adapter after each successful historical page extends the chart range', async () => {
    const onHistoryLoaded = vi.fn()
    const page = { data: [candle(300_000)], has_more_before: true }
    const feed = createBinanceProDatafeed({
      getSupportedSymbols: () => ['BTCUSDT'],
      fetchHistory: vi.fn().mockResolvedValue(page),
      onHistoryLoaded
    })

    await feed.getHistoryKLineData(symbol('BTCUSDT'), { multiplier: 5, timespan: 'minute', text: '5m' }, 0, 500_000)

    expect(onHistoryLoaded).toHaveBeenCalledWith({ symbol: 'BTCUSDT', interval: '5m' }, page)
  })

  it('maps Pro periods to Binance intervals and requests only candles in the requested range', async () => {
    const fetchHistory = vi.fn<(request: KlineHistoryRequest, signal?: AbortSignal) => Promise<{
      data: ReturnType<typeof candle>[]; has_more_before: boolean
    }>>().mockResolvedValue({
      data: [candle(300_000), candle(120_000), candle(240_000), candle(240_000, 999), candle(360_000)],
      has_more_before: true
    })
    const feed = createBinanceProDatafeed({ getSupportedSymbols: () => ['BTCUSDT'], fetchHistory })

    const bars = await feed.getHistoryKLineData(symbol('BTCUSDT'), { multiplier: 5, timespan: 'minute', text: '5m' }, 200_000, 300_000)

    expect(fetchHistory).toHaveBeenCalledWith(expect.objectContaining({
      symbol: 'BTCUSDT', interval: '5m', limit: 500, endTime: 300_000
    }), expect.any(AbortSignal))
    expect(bars.map(bar => bar.timestamp)).toEqual([240_000, 300_000])
    expect(bars.find(bar => bar.timestamp === 240_000)?.close).toBe(240_000)
    expect(feed.hasMoreHistory(symbol('BTCUSDT'), { multiplier: 5, timespan: 'minute', text: '5m' })).toBe(true)
  })

  it('stops Pro history pagination when Binance reports no older candles', async () => {
    const period = { multiplier: 5, timespan: 'minute', text: '5m' }
    const feed = createBinanceProDatafeed({
      getSupportedSymbols: () => ['BTCUSDT'],
      fetchHistory: vi.fn().mockResolvedValue({ data: [candle(300_000)], has_more_before: false })
    })

    await feed.getHistoryKLineData(symbol('BTCUSDT'), period, 0, 500_000)

    expect(feed.hasMoreHistory(symbol('BTCUSDT'), period)).toBe(false)
  })

  it('does not return a history response after the feed switches to another symbol or period', async () => {
    let resolveOld!: (page: { data: ReturnType<typeof candle>[]; has_more_before: boolean }) => void
    const fetchHistory = vi.fn()
      .mockImplementationOnce((_request: KlineHistoryRequest, _signal?: AbortSignal) => new Promise(resolve => { resolveOld = resolve }))
      .mockResolvedValueOnce({ data: [candle(900_000)], has_more_before: false })
    const feed = createBinanceProDatafeed({ getSupportedSymbols: () => ['BTCUSDT', 'ETHUSDT'], fetchHistory })

    const oldRequest = feed.getHistoryKLineData(symbol('BTCUSDT'), { multiplier: 1, timespan: 'minute', text: '1m' }, 0, 500_000)
    await Promise.resolve()
    const oldSignal = fetchHistory.mock.calls[0]?.[1]
    await feed.getHistoryKLineData(symbol('ETHUSDT'), { multiplier: 5, timespan: 'minute', text: '5m' }, 800_000, 1_000_000)
    expect(oldSignal?.aborted).toBe(true)
    resolveOld({ data: [candle(100_000)], has_more_before: true })

    await expect(oldRequest).resolves.toEqual([])
  })

  it('publishes only matching live candles and stops publishing after unsubscribe', () => {
    const feed = createBinanceProDatafeed({ getSupportedSymbols: () => ['BTCUSDT'], fetchHistory: vi.fn() })
    const callback = vi.fn()
    const period = { multiplier: 1, timespan: 'minute', text: '1m' }
    feed.subscribe(symbol('BTCUSDT'), period, callback)

    feed.publishCandle('ETHUSDT', '1m', candle(60_000))
    feed.publishCandle('BTCUSDT', '5m', candle(60_000))
    feed.publishCandle('BTCUSDT', '1m', candle(60_000))
    expect(callback).toHaveBeenCalledTimes(1)

    feed.unsubscribe(symbol('BTCUSDT'), period)
    feed.publishCandle('BTCUSDT', '1m', candle(120_000))
    expect(callback).toHaveBeenCalledTimes(1)
  })

  it('reports history failures and resolves with an empty page so Pro can retry later', async () => {
    const failure = new Error('history unavailable')
    const onHistoryError = vi.fn()
    const feed = createBinanceProDatafeed({
      getSupportedSymbols: () => ['BTCUSDT'],
      fetchHistory: vi.fn().mockRejectedValue(failure),
      onHistoryError
    })

    await expect(feed.getHistoryKLineData(
      symbol('BTCUSDT'), { multiplier: 1, timespan: 'minute', text: '1m' }, 0, 500_000
    )).resolves.toEqual([])
    expect(onHistoryError).toHaveBeenCalledWith(expect.objectContaining({ symbol: 'BTCUSDT', interval: '1m' }), failure)
    expect(feed.hasMoreHistory(symbol('BTCUSDT'), { multiplier: 1, timespan: 'minute', text: '1m' })).toBe(true)
  })
})
