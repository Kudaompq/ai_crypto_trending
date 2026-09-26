// @vitest-environment jsdom

import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useAnalysisStore } from '../stores/analysis'

const apiMocks = vi.hoisted(() => ({
  getWatchlist: vi.fn(),
  importLegacyWatchlist: vi.fn(),
  addWatchlistSymbol: vi.fn(),
  removeWatchlistSymbol: vi.fn(),
  reorderWatchlist: vi.fn(),
  validateSymbol: vi.fn(async (raw: string) => raw.trim().toUpperCase()),
  getWatchlistPrices: vi.fn(async () => ({ prices: [
    { symbol: 'BTCUSDT', price: 100, change_24h_percent: 0.5, event_time: 1000 },
    { symbol: 'ETHUSDT', price: 200, change_24h_percent: -0.25, event_time: 1000 }
  ], unavailable_symbols: [] })),
  createWatchlistPriceStream: vi.fn(),
  getKlineData: vi.fn(),
  getAnalysis: vi.fn(),
  errorMessage: vi.fn((_error: unknown, fallback: string) => fallback),
  invalidSelection: vi.fn(() => false)
}))

const watchlist = (symbols: string[], revision = 1, pending = false) => ({
  symbols, revision, legacy_import_pending: pending
})

vi.mock('../services/api', () => ({ api: apiMocks }))

describe('editable preset symbols', () => {
  beforeEach(() => {
    localStorage.clear()
    apiMocks.validateSymbol.mockClear()
    setActivePinia(createPinia())
    apiMocks.getWatchlist.mockResolvedValue(watchlist(['BTCUSDT', 'ETHUSDT']))
    apiMocks.importLegacyWatchlist.mockImplementation(async (symbols: string[]) => watchlist(symbols, 2))
    apiMocks.addWatchlistSymbol.mockImplementation(async (symbol: string) => watchlist(['BTCUSDT', 'ETHUSDT', symbol], 2))
    apiMocks.removeWatchlistSymbol.mockImplementation(async (symbol: string) => watchlist(['BTCUSDT', 'ETHUSDT'].filter(item => item !== symbol), 2))
    apiMocks.reorderWatchlist.mockImplementation(async (_revision: number, symbols: string[]) => watchlist(symbols, 2))
    apiMocks.getWatchlistPrices.mockResolvedValue({ prices: [
      { symbol: 'BTCUSDT', price: 100, change_24h_percent: 0.5, event_time: 1000 },
      { symbol: 'ETHUSDT', price: 200, change_24h_percent: -0.25, event_time: 1000 }
    ], unavailable_symbols: [] })
    apiMocks.createWatchlistPriceStream.mockReset()
  })

  it('uses an active USD-M contract instead of the retired MATIC preset', () => {
    const store = useAnalysisStore()
    expect(store.availableSymbols.some(item => item.value === 'POLUSDT')).toBe(true)
    expect(store.availableSymbols.some(item => item.value === 'MATICUSDT')).toBe(false)
  })

  it('restores the saved watchlist order and reconciles stale and duplicate entries', () => {
    localStorage.setItem('crypto-trending-watchlist-order', JSON.stringify([
      'SOLUSDT', 'ETHUSDT', 'SOLUSDT', 'REMOVEDUSDT'
    ]))

    const store = useAnalysisStore()
    const order = store.availableSymbols.map(item => item.value)

    expect(order.slice(0, 2)).toEqual(['SOLUSDT', 'ETHUSDT'])
    expect(order).toHaveLength(new Set(order).size)
    expect(order).not.toContain('REMOVEDUSDT')
    expect(order).toContain('BTCUSDT')
  })

  it('appends and removes symbols through the shared server Watchlist', async () => {
    apiMocks.getWatchlist.mockResolvedValueOnce(watchlist(['SOLUSDT', 'ETHUSDT']))
    const store = useAnalysisStore()
    await store.initializeWatchlist()
    apiMocks.addWatchlistSymbol.mockResolvedValueOnce(watchlist(['SOLUSDT', 'ETHUSDT', 'AVAXUSDT'], 2))
    apiMocks.removeWatchlistSymbol.mockResolvedValueOnce(watchlist(['SOLUSDT', 'AVAXUSDT'], 3))

    await store.addCustomSymbol('AVAXUSDT')
    expect(store.availableSymbols[store.availableSymbols.length - 1]?.value).toBe('AVAXUSDT')

    await store.removeSymbol('ETHUSDT')
    expect(store.availableSymbols.map(item => item.value)).not.toContain('ETHUSDT')
    expect(apiMocks.addWatchlistSymbol).toHaveBeenCalledWith('AVAXUSDT')
    expect(apiMocks.removeWatchlistSymbol).toHaveBeenCalledWith('ETHUSDT')
  })

  it('does not let a previous period response replace the newly selected market data', async () => {
    const candle = (timestamp: number) => ({ timestamp, open: 1, high: 2, low: 0.5, close: 1.5, volume: 10 })
    let resolveOldKline!: (value: unknown) => void
    let resolveOldAnalysis!: (value: unknown) => void
    let resolveNewKline!: (value: unknown) => void
    let resolveNewAnalysis!: (value: unknown) => void
    apiMocks.getKlineData
      .mockImplementationOnce(() => new Promise(resolve => { resolveOldKline = resolve }))
      .mockImplementationOnce(() => new Promise(resolve => { resolveNewKline = resolve }))
    apiMocks.getAnalysis
      .mockImplementationOnce(() => new Promise(resolve => { resolveOldAnalysis = resolve }))
      .mockImplementationOnce(() => new Promise(resolve => { resolveNewAnalysis = resolve }))
    const store = useAnalysisStore()

    const oldRequest = store.fetchAnalysis()
    store.setInterval('3m')
    const newRequest = store.fetchAnalysis()
    resolveNewKline({ symbol: 'ETHUSDT', interval: '3m', data: [candle(3)] })
    resolveNewAnalysis({ symbol: 'ETHUSDT', interval: '3m' })
    await newRequest
    resolveOldKline({ symbol: 'ETHUSDT', interval: '1d', data: [candle(1)] })
    resolveOldAnalysis({ symbol: 'ETHUSDT', interval: '1d' })
    await oldRequest

    expect(store.klineData?.interval).toBe('3m')
    expect(store.klineData?.data[0]?.timestamp).toBe(3)
    expect(store.analysisResult?.interval).toBe('3m')
  })

  it('removes symbols from the server and selects a remaining symbol', async () => {
    apiMocks.getWatchlist
      .mockResolvedValueOnce(watchlist(['BTCUSDT', 'ETHUSDT']))
      .mockResolvedValueOnce(watchlist(['BTCUSDT'], 2))
    const store = useAnalysisStore()
    await store.initializeWatchlist()
    apiMocks.removeWatchlistSymbol.mockResolvedValueOnce(watchlist(['BTCUSDT'], 2))
    store.setSymbol('ETHUSDT')
    await store.removeSymbol('ETHUSDT')

    expect(store.availableSymbols.some(item => item.value === 'ETHUSDT')).toBe(false)
    expect(store.symbol).toBe('BTCUSDT')

    setActivePinia(createPinia())
    const restoredStore = useAnalysisStore()
    await restoredStore.initializeWatchlist()
    expect(restoredStore.availableSymbols.some(item => item.value === 'ETHUSDT')).toBe(false)
    expect(restoredStore.symbol).toBe('BTCUSDT')
  })

  it('allows a removed preset to be added back through the server', async () => {
    apiMocks.getWatchlist.mockResolvedValueOnce(watchlist(['BTCUSDT']))
    const store = useAnalysisStore()
    await store.initializeWatchlist()
    apiMocks.addWatchlistSymbol.mockResolvedValueOnce(watchlist(['BTCUSDT', 'ETHUSDT'], 2))

    await expect(store.addCustomSymbol('ETHUSDT')).resolves.toBe('ETHUSDT')
    expect(store.availableSymbols.some(item => item.value === 'ETHUSDT')).toBe(true)
    expect(apiMocks.validateSymbol).not.toHaveBeenCalled()
    expect(apiMocks.addWatchlistSymbol).toHaveBeenCalledWith('ETHUSDT')
  })

  it('keeps at least one symbol in the selector', async () => {
    apiMocks.getWatchlist.mockResolvedValueOnce(watchlist(['BTCUSDT', 'ETHUSDT']))
    apiMocks.removeWatchlistSymbol.mockClear()
    const store = useAnalysisStore()
    await store.initializeWatchlist()
    apiMocks.removeWatchlistSymbol.mockResolvedValueOnce(watchlist(['ETHUSDT'], 2))
    await store.removeSymbol('BTCUSDT')

    const lastSymbol = store.availableSymbols[0]
    expect(lastSymbol).toBeDefined()
    await store.removeSymbol(lastSymbol!.value)
    expect(store.availableSymbols).toHaveLength(1)
    expect(apiMocks.removeWatchlistSymbol).toHaveBeenCalledTimes(1)
  })

  it('loads every watchlist quote and keeps newer stream prices when an older snapshot arrives late', async () => {
    let onQuote: (event: { symbol: string; price: number; change_24h_percent: number; event_time: number }) => void = () => {}
    apiMocks.createWatchlistPriceStream.mockImplementation((_symbols, handler) => {
      onQuote = handler
      return { close: vi.fn() }
    })
    const store = useAnalysisStore()

    const loading = store.startWatchlistPriceStream()
    onQuote({ symbol: 'BTCUSDT', price: 101, change_24h_percent: 1, event_time: 2000 })
    await loading

    expect(apiMocks.createWatchlistPriceStream).toHaveBeenCalledWith(
      expect.arrayContaining(store.availableSymbols.map(item => item.value)),
      expect.any(Function), expect.any(Function)
    )
    expect(store.pricesBySymbol.BTCUSDT).toEqual({ price: 101, change24hPercent: 1, eventTime: 2000 })
    expect(store.pricesBySymbol.ETHUSDT).toEqual({ price: 200, change24hPercent: -0.25, eventTime: 1000 })
  })

  it('ignores quotes outside the watchlist and retains prices while the shared feed reconnects', async () => {
    let onQuote: (event: { symbol: string; price: number; change_24h_percent: number; event_time: number }) => void = () => {}
    let onStatus: (status: { state: 'connecting' | 'live' | 'reconnecting'; message: string; unavailable_symbols?: string[] }) => void = () => {}
    apiMocks.createWatchlistPriceStream.mockImplementation((_symbols, quoteHandler, statusHandler) => {
      onQuote = quoteHandler
      onStatus = statusHandler
      return { close: vi.fn() }
    })
    const store = useAnalysisStore()
    await store.startWatchlistPriceStream()

    onQuote({ symbol: 'NOTWATCHEDUSDT', price: 1, change_24h_percent: 0, event_time: 5 })
    onStatus({ state: 'reconnecting', message: '连接中断', unavailable_symbols: ['FAKEUSDT'] })

    expect(store.pricesBySymbol.NOTWATCHEDUSDT).toBeUndefined()
    expect(store.priceStreamState).toBe('reconnecting')
    expect(store.unavailableSymbols).toEqual(['FAKEUSDT'])
    expect(store.pricesBySymbol.BTCUSDT).toEqual({ price: 100, change24hPercent: 0.5, eventTime: 1000 })

    onStatus({ state: 'live', message: '连接已恢复' })
    expect(store.priceStreamState).toBe('live')
  })

  it('lets a received stream update supersede a REST snapshot timestamp from another clock', async () => {
    let onQuote: (event: { symbol: string; price: number; change_24h_percent: number; event_time: number }) => void = () => {}
    apiMocks.getWatchlistPrices.mockResolvedValue({ prices: [
      { symbol: 'BTCUSDT', price: 100, change_24h_percent: 0.5, event_time: 5000 }
    ], unavailable_symbols: [] })
    apiMocks.createWatchlistPriceStream.mockImplementation((_symbols, handler) => {
      onQuote = handler
      return { close: vi.fn() }
    })
    const store = useAnalysisStore()
    await store.startWatchlistPriceStream()

    onQuote({ symbol: 'BTCUSDT', price: 101, change_24h_percent: 1, event_time: 4000 })

    expect(store.pricesBySymbol.BTCUSDT).toEqual({ price: 101, change24hPercent: 1, eventTime: 4000 })
  })

  it('closes the previous quote stream when the watchlist subscription is replaced or stopped', () => {
    const oldStream = { close: vi.fn() }
    const newStream = { close: vi.fn() }
    apiMocks.createWatchlistPriceStream.mockReturnValueOnce(oldStream).mockReturnValueOnce(newStream)
    const store = useAnalysisStore()

    void store.startWatchlistPriceStream()
    void store.startWatchlistPriceStream()
    expect(oldStream.close).toHaveBeenCalledOnce()
    store.stopWatchlistPriceStream()
    expect(newStream.close).toHaveBeenCalledOnce()
  })

  it('resubscribes with the changed symbols after adding and removing a watchlist item', async () => {
    const streams = [{ close: vi.fn() }, { close: vi.fn() }, { close: vi.fn() }]
    apiMocks.createWatchlistPriceStream
      .mockReturnValueOnce(streams[0])
      .mockReturnValueOnce(streams[1])
      .mockReturnValueOnce(streams[2])
    const store = useAnalysisStore()
    await store.initializeWatchlist()
    await store.startWatchlistPriceStream()
    apiMocks.addWatchlistSymbol.mockResolvedValueOnce(watchlist(['BTCUSDT', 'ETHUSDT', 'AVAXUSDT'], 2))

    await store.addCustomSymbol('AVAXUSDT')
    await store.startWatchlistPriceStream()
    expect(apiMocks.createWatchlistPriceStream.mock.calls[1]?.[0]).toContain('AVAXUSDT')
    expect(streams[0]!.close).toHaveBeenCalledOnce()

    apiMocks.removeWatchlistSymbol.mockResolvedValueOnce(watchlist(['BTCUSDT', 'ETHUSDT'], 3))
    await store.removeSymbol('AVAXUSDT')
    await store.startWatchlistPriceStream()
    expect(apiMocks.createWatchlistPriceStream.mock.calls[2]?.[0]).not.toContain('AVAXUSDT')
    expect(streams[1]!.close).toHaveBeenCalledOnce()
  })
})
