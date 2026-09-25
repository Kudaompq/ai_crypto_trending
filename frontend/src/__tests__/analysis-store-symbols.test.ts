// @vitest-environment jsdom

import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useAnalysisStore } from '../stores/analysis'

const apiMocks = vi.hoisted(() => ({
  validateSymbol: vi.fn(async (raw: string) => raw.trim().toUpperCase()),
  getWatchlistPrices: vi.fn(async () => ({ prices: [
    { symbol: 'BTCUSDT', price: 100, event_time: 1000 },
    { symbol: 'ETHUSDT', price: 200, event_time: 1000 }
  ], unavailable_symbols: [] })),
  createWatchlistPriceStream: vi.fn()
}))

vi.mock('../services/api', () => ({ api: apiMocks }))

describe('editable preset symbols', () => {
  beforeEach(() => {
    localStorage.clear()
    apiMocks.validateSymbol.mockClear()
    setActivePinia(createPinia())
    apiMocks.getWatchlistPrices.mockResolvedValue({ prices: [
      { symbol: 'BTCUSDT', price: 100, event_time: 1000 },
      { symbol: 'ETHUSDT', price: 200, event_time: 1000 }
    ], unavailable_symbols: [] })
    apiMocks.createWatchlistPriceStream.mockReset()
  })

  it('uses an active USD-M contract instead of the retired MATIC preset', () => {
    const store = useAnalysisStore()
    expect(store.availableSymbols.some(item => item.value === 'POLUSDT')).toBe(true)
    expect(store.availableSymbols.some(item => item.value === 'MATICUSDT')).toBe(false)
  })

  it('removes preset symbols persistently and selects a remaining symbol', () => {
    const store = useAnalysisStore()
    store.removeSymbol('ETHUSDT')

    expect(store.availableSymbols.some(item => item.value === 'ETHUSDT')).toBe(false)
    expect(store.symbol).toBe('BTCUSDT')

    setActivePinia(createPinia())
    const restoredStore = useAnalysisStore()
    expect(restoredStore.availableSymbols.some(item => item.value === 'ETHUSDT')).toBe(false)
    expect(restoredStore.symbol).toBe('BTCUSDT')
  })

  it('allows a removed preset to be added back', async () => {
    const store = useAnalysisStore()
    store.removeSymbol('ETHUSDT')

    await expect(store.addCustomSymbol('ETHUSDT')).resolves.toBe('ETHUSDT')
    expect(store.availableSymbols.some(item => item.value === 'ETHUSDT')).toBe(true)
    expect(apiMocks.validateSymbol).not.toHaveBeenCalled()
  })

  it('keeps at least one symbol in the selector', () => {
    const store = useAnalysisStore()
    for (const item of store.availableSymbols.slice(0, -1)) {
      store.removeSymbol(item.value)
    }

    const lastSymbol = store.availableSymbols[0]
    expect(lastSymbol).toBeDefined()
    store.removeSymbol(lastSymbol!.value)
    expect(store.availableSymbols).toHaveLength(1)
  })

  it('loads every watchlist quote and keeps newer stream prices when an older snapshot arrives late', async () => {
    let onQuote: (event: { symbol: string; price: number; event_time: number }) => void = () => {}
    apiMocks.createWatchlistPriceStream.mockImplementation((_symbols, handler) => {
      onQuote = handler
      return { close: vi.fn() }
    })
    const store = useAnalysisStore()

    const loading = store.startWatchlistPriceStream()
    onQuote({ symbol: 'BTCUSDT', price: 101, event_time: 2000 })
    await loading

    expect(apiMocks.createWatchlistPriceStream).toHaveBeenCalledWith(
      expect.arrayContaining(store.availableSymbols.map(item => item.value)),
      expect.any(Function), expect.any(Function)
    )
    expect(store.pricesBySymbol.BTCUSDT).toEqual({ price: 101, eventTime: 2000 })
    expect(store.pricesBySymbol.ETHUSDT).toEqual({ price: 200, eventTime: 1000 })
  })

  it('ignores quotes outside the watchlist and retains prices while the shared feed reconnects', async () => {
    let onQuote: (event: { symbol: string; price: number; event_time: number }) => void = () => {}
    let onStatus: (status: { state: 'connecting' | 'live' | 'reconnecting'; message: string; unavailable_symbols?: string[] }) => void = () => {}
    apiMocks.createWatchlistPriceStream.mockImplementation((_symbols, quoteHandler, statusHandler) => {
      onQuote = quoteHandler
      onStatus = statusHandler
      return { close: vi.fn() }
    })
    const store = useAnalysisStore()
    await store.startWatchlistPriceStream()

    onQuote({ symbol: 'NOTWATCHEDUSDT', price: 1, event_time: 5 })
    onStatus({ state: 'reconnecting', message: '连接中断', unavailable_symbols: ['FAKEUSDT'] })

    expect(store.pricesBySymbol.NOTWATCHEDUSDT).toBeUndefined()
    expect(store.priceStreamState).toBe('reconnecting')
    expect(store.unavailableSymbols).toEqual(['FAKEUSDT'])
    expect(store.pricesBySymbol.BTCUSDT).toEqual({ price: 100, eventTime: 1000 })

    onStatus({ state: 'live', message: '连接已恢复' })
    expect(store.priceStreamState).toBe('live')
  })

  it('lets a received stream update supersede a REST snapshot timestamp from another clock', async () => {
    let onQuote: (event: { symbol: string; price: number; event_time: number }) => void = () => {}
    apiMocks.getWatchlistPrices.mockResolvedValue({ prices: [
      { symbol: 'BTCUSDT', price: 100, event_time: 5000 }
    ], unavailable_symbols: [] })
    apiMocks.createWatchlistPriceStream.mockImplementation((_symbols, handler) => {
      onQuote = handler
      return { close: vi.fn() }
    })
    const store = useAnalysisStore()
    await store.startWatchlistPriceStream()

    onQuote({ symbol: 'BTCUSDT', price: 101, event_time: 4000 })

    expect(store.pricesBySymbol.BTCUSDT).toEqual({ price: 101, eventTime: 4000 })
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
    await store.startWatchlistPriceStream()

    await store.addCustomSymbol('AVAXUSDT')
    await store.startWatchlistPriceStream()
    expect(apiMocks.createWatchlistPriceStream.mock.calls[1]?.[0]).toContain('AVAXUSDT')
    expect(streams[0]!.close).toHaveBeenCalledOnce()

    store.removeSymbol('AVAXUSDT')
    await store.startWatchlistPriceStream()
    expect(apiMocks.createWatchlistPriceStream.mock.calls[2]?.[0]).not.toContain('AVAXUSDT')
    expect(streams[1]!.close).toHaveBeenCalledOnce()
  })
})
