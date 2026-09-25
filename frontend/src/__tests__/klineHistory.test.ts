import { describe, expect, it, vi } from 'vitest'
import { KlineHistory, toKlineChartPeriod, type KlineHistoryPage, type KlineHistoryRequest } from '../services/klineHistory'

const candle = (timestamp: number, close = timestamp) => ({
  timestamp, open: close - 1, high: close + 1, low: close - 2, close, volume: 10
})

describe('KLineChart period and history adapter', () => {
  it('maps every Binance native interval to KLineChart span and type', () => {
    const expected: Record<string, { span: number; type: string }> = {
      '1m': { span: 1, type: 'minute' }, '3m': { span: 3, type: 'minute' },
      '5m': { span: 5, type: 'minute' }, '15m': { span: 15, type: 'minute' },
      '30m': { span: 30, type: 'minute' }, '1h': { span: 1, type: 'hour' },
      '2h': { span: 2, type: 'hour' }, '4h': { span: 4, type: 'hour' },
      '6h': { span: 6, type: 'hour' }, '8h': { span: 8, type: 'hour' },
      '12h': { span: 12, type: 'hour' }, '1d': { span: 1, type: 'day' },
      '3d': { span: 3, type: 'day' }, '1w': { span: 1, type: 'week' },
      '1M': { span: 1, type: 'month' }
    }

    for (const [interval, period] of Object.entries(expected)) {
      expect(toKlineChartPeriod(interval), interval).toEqual(period)
    }
    expect(() => toKlineChartPeriod('2w')).toThrow(/unsupported/i)
  })

  it('sorts initial bars, pages before the first timestamp, and deduplicates boundaries', async () => {
    const fetchPage = vi.fn<(request: KlineHistoryRequest, signal?: AbortSignal) => Promise<KlineHistoryPage>>()
      .mockResolvedValueOnce({ data: [candle(3000), candle(2000)], has_more_before: true })
      .mockResolvedValueOnce({ data: [candle(2000, 999), candle(1000)], has_more_before: false })
    const history = new KlineHistory(fetchPage)

    await history.loadInitial('ETHUSDT', '1h', 2)
    expect(history.candles.map(item => item.timestamp)).toEqual([2000, 3000])
    const olderBars = await history.loadOlder('ETHUSDT', '1h', 2)

    expect(fetchPage.mock.calls[1]?.[0]).toEqual({
      symbol: 'ETHUSDT', interval: '1h', limit: 2, endTime: 1999
    })
    expect(history.candles.map(item => item.timestamp)).toEqual([1000, 2000, 3000])
    expect(olderBars.map(item => item.timestamp)).toEqual([1000])
    expect(history.candles[1]?.close).toBe(2000)
    expect(history.hasMoreBefore).toBe(false)
    await history.loadOlder('ETHUSDT', '1h', 2)
    expect(fetchPage).toHaveBeenCalledTimes(2)
  })

  it('retains loaded bars after a page failure and allows retry', async () => {
    const error = new Error('temporary history failure')
    const fetchPage = vi.fn<(request: KlineHistoryRequest, signal?: AbortSignal) => Promise<KlineHistoryPage>>()
      .mockResolvedValueOnce({ data: [candle(2000)], has_more_before: true })
      .mockRejectedValueOnce(error)
      .mockResolvedValueOnce({ data: [candle(1000)], has_more_before: false })
    const history = new KlineHistory(fetchPage)
    await history.loadInitial('ETHUSDT', '15m', 100)

    await expect(history.loadOlder('ETHUSDT', '15m', 100)).rejects.toBe(error)
    expect(history.candles.map(item => item.timestamp)).toEqual([2000])
    await history.loadOlder('ETHUSDT', '15m', 100)
    expect(history.candles.map(item => item.timestamp)).toEqual([1000, 2000])
  })

  it('ignores a pending page after the symbol or period is reset', async () => {
    let resolveOld!: (page: KlineHistoryPage) => void
    const fetchPage = vi.fn()
      .mockImplementationOnce(() => new Promise<KlineHistoryPage>(resolve => { resolveOld = resolve }))
      .mockResolvedValueOnce({ data: [candle(9000)], has_more_before: false })
    const history = new KlineHistory(fetchPage)
    const oldRequest = history.loadInitial('ETHUSDT', '1h', 100)
    await Promise.resolve()
    expect(resolveOld).toBeTypeOf('function')
    history.reset('BTCUSDT', '3m')
    await history.loadInitial('BTCUSDT', '3m', 100)
    resolveOld({ data: [candle(1000)], has_more_before: true })
    await oldRequest

    expect(history.candles.map(item => item.timestamp)).toEqual([9000])
    expect(history.hasMoreBefore).toBe(false)
  })
})
