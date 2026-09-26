// @vitest-environment jsdom

import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useAnalysisStore } from '../stores/analysis'

type WatchlistResponse = { symbols: string[]; revision: number; legacy_import_pending: boolean }
const apiMocks = vi.hoisted(() => ({
  getWatchlist: vi.fn<() => Promise<WatchlistResponse>>(),
  importLegacyWatchlist: vi.fn<(symbols: string[]) => Promise<WatchlistResponse>>(),
  addWatchlistSymbol: vi.fn<(symbol: string) => Promise<WatchlistResponse>>(),
  removeWatchlistSymbol: vi.fn<(symbol: string) => Promise<WatchlistResponse>>(),
  reorderWatchlist: vi.fn<(revision: number, symbols: string[]) => Promise<WatchlistResponse>>(),
  validateSymbol: vi.fn(async (raw: string) => raw.trim().toUpperCase()),
  getWatchlistPrices: vi.fn(async () => ({ prices: [], unavailable_symbols: [] })),
  createWatchlistPriceStream: vi.fn(() => ({ close: vi.fn() })),
  getKlineData: vi.fn(),
  getAnalysis: vi.fn(),
  errorMessage: vi.fn((_error: unknown, fallback: string) => fallback),
  invalidSelection: vi.fn(() => false)
}))

vi.mock('../services/api', () => ({ api: apiMocks }))

const state = (symbols: string[], revision: number, pending = false): WatchlistResponse => ({
  symbols, revision, legacy_import_pending: pending
})

describe('server-backed shared Watchlist', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.clearAllMocks()
    setActivePinia(createPinia())
    apiMocks.getWatchlist.mockResolvedValue(state(['BTCUSDT', 'ETHUSDT'], 1))
    apiMocks.importLegacyWatchlist.mockImplementation(async symbols => state(symbols, 2))
    apiMocks.addWatchlistSymbol.mockImplementation(async symbol => state(['BTCUSDT', 'ETHUSDT', symbol], 2))
    apiMocks.removeWatchlistSymbol.mockImplementation(async symbol => state(['BTCUSDT', 'ETHUSDT'].filter(item => item !== symbol), 2))
    apiMocks.reorderWatchlist.mockImplementation(async (_revision, symbols) => state(symbols, 2))
  })

  it('imports the first browser legacy list once and adopts the confirmed shared order', async () => {
    localStorage.setItem('crypto-trending-custom-symbols', JSON.stringify(['AVAXUSDT']))
    localStorage.setItem('crypto-trending-removed-presets', JSON.stringify(['BNBUSDT']))
    localStorage.setItem('crypto-trending-watchlist-order', JSON.stringify(['SOLUSDT', 'BTCUSDT', 'AVAXUSDT']))
    apiMocks.getWatchlist.mockResolvedValueOnce(state(['BTCUSDT', 'ETHUSDT'], 1, true))
    apiMocks.importLegacyWatchlist.mockResolvedValueOnce(state(['SOLUSDT', 'BTCUSDT', 'AVAXUSDT'], 2))
    const store = useAnalysisStore()

    await store.initializeWatchlist()

    expect(apiMocks.importLegacyWatchlist).toHaveBeenCalledWith(expect.arrayContaining(['SOLUSDT', 'BTCUSDT', 'AVAXUSDT']))
    expect(store.availableSymbols.map(item => item.value)).toEqual(['SOLUSDT', 'BTCUSDT', 'AVAXUSDT'])
    expect(localStorage.getItem('crypto-trending-custom-symbols')).toBeNull()
    expect(localStorage.getItem('crypto-trending-removed-presets')).toBeNull()
    expect(localStorage.getItem('crypto-trending-watchlist-order')).toBeNull()
    expect(store.storageWarning).toBeNull()
  })

  it('uses the database list when migration is already complete, regardless of local cache', async () => {
    localStorage.setItem('crypto-trending-custom-symbols', JSON.stringify(['SOLUSDT']))
    apiMocks.getWatchlist.mockResolvedValueOnce(state(['ETHUSDT', 'BTCUSDT'], 8))
    const store = useAnalysisStore()

    await store.initializeWatchlist()

    expect(apiMocks.importLegacyWatchlist).not.toHaveBeenCalled()
    expect(store.availableSymbols.map(item => item.value)).toEqual(['ETHUSDT', 'BTCUSDT'])
    expect(localStorage.getItem('crypto-trending-custom-symbols')).toBeNull()
  })

  it('keeps cached symbols as an explicitly unsynchronized fallback when loading fails', async () => {
    localStorage.setItem('crypto-trending-custom-symbols', JSON.stringify(['SOLUSDT']))
    apiMocks.getWatchlist.mockRejectedValueOnce(new Error('database unavailable'))
    const store = useAnalysisStore()

    await store.initializeWatchlist()

    expect(store.availableSymbols.map(item => item.value)).toContain('SOLUSDT')
    expect(store.storageWarning).toMatch(/同步|数据库|服务端/)
    expect(apiMocks.importLegacyWatchlist).not.toHaveBeenCalled()
    expect(store.watchlistReady).toBe(true)
    expect(store.watchlistSynced).toBe(false)
  })

  it('does not present a failed add as committed', async () => {
    const store = useAnalysisStore()
    await store.initializeWatchlist()
    apiMocks.addWatchlistSymbol.mockRejectedValueOnce(new Error('database unavailable'))

    await expect(store.addCustomSymbol('AVAXUSDT')).rejects.toThrow('database unavailable')

    expect(store.availableSymbols.map(item => item.value)).toEqual(['BTCUSDT', 'ETHUSDT'])
    expect(store.storageWarning).toMatch(/同步|保存/)
  })

  it('persists add, remove, and reorder through the server and preserves selected symbol', async () => {
    const store = useAnalysisStore()
    await store.initializeWatchlist()
    store.setSymbol('ETHUSDT')
    apiMocks.addWatchlistSymbol.mockResolvedValueOnce(state(['BTCUSDT', 'ETHUSDT', 'AVAXUSDT'], 2))
    apiMocks.removeWatchlistSymbol.mockResolvedValueOnce(state(['ETHUSDT', 'AVAXUSDT'], 3))
    apiMocks.reorderWatchlist.mockResolvedValueOnce(state(['AVAXUSDT', 'ETHUSDT'], 4))

    await store.addCustomSymbol('AVAXUSDT')
    await store.removeSymbol('BTCUSDT')
    await store.reorderSymbols('AVAXUSDT', 0)

    expect(apiMocks.addWatchlistSymbol).toHaveBeenCalledWith('AVAXUSDT')
    expect(apiMocks.removeWatchlistSymbol).toHaveBeenCalledWith('BTCUSDT')
    expect(apiMocks.reorderWatchlist).toHaveBeenCalledWith(3, ['AVAXUSDT', 'ETHUSDT'])
    expect(store.availableSymbols.map(item => item.value)).toEqual(['AVAXUSDT', 'ETHUSDT'])
    expect(store.symbol).toBe('ETHUSDT')
  })

  it('reloads the latest shared list after a stale reorder conflict', async () => {
    apiMocks.getWatchlist
      .mockResolvedValueOnce(state(['BTCUSDT', 'ETHUSDT'], 3))
      .mockResolvedValueOnce(state(['ETHUSDT', 'BTCUSDT', 'AVAXUSDT'], 4))
    const store = useAnalysisStore()
    await store.initializeWatchlist()
    apiMocks.reorderWatchlist.mockRejectedValueOnce({ response: { status: 409 } })

    await store.reorderSymbols('BTCUSDT', 1)

    expect(apiMocks.reorderWatchlist).toHaveBeenCalledWith(3, ['ETHUSDT', 'BTCUSDT'])
    expect(store.availableSymbols.map(item => item.value)).toEqual(['ETHUSDT', 'BTCUSDT', 'AVAXUSDT'])
    expect(store.storageWarning).toMatch(/其他客户端|最新列表/)
  })
})
