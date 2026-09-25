import { describe, expect, it, vi } from 'vitest'

const axiosMocks = vi.hoisted(() => ({ get: vi.fn(), isAxiosError: vi.fn(() => false) }))
vi.mock('axios', () => ({ default: axiosMocks }))
import { api } from '../services/api'

describe('frontend API surface', () => {
  it('does not expose the retired trading opportunity endpoint', () => {
    expect(api).not.toHaveProperty('getOpportunities')
  })

  it('loads REST prices for the requested watchlist symbols', async () => {
    axiosMocks.get.mockResolvedValueOnce({ data: {
      prices: [{ symbol: 'BTCUSDT', price: 65000, change_24h_percent: 1.5, event_time: 1 }], unavailable_symbols: []
    } })

    await expect(api.getWatchlistPrices(['BTCUSDT', 'ETHUSDT'])).resolves.toEqual({
      prices: [{ symbol: 'BTCUSDT', price: 65000, change_24h_percent: 1.5, event_time: 1 }], unavailable_symbols: []
    })
    expect(axiosMocks.get).toHaveBeenCalledWith('/api/watchlist/prices', {
      params: { symbols: 'BTCUSDT,ETHUSDT' }
    })
  })

  it('passes the exclusive history cursor to the K-line API when requested', async () => {
    axiosMocks.get.mockResolvedValueOnce({ data: { symbol: 'ETHUSDT', interval: '1h', data: [], has_more_before: false } })
    const getKlineData = api.getKlineData as unknown as
      (symbol: string, interval: string, limit: number, endTime?: number) => Promise<unknown>

    await getKlineData('ETHUSDT', '1h', 100, 1_700_000_000_000)

    expect(axiosMocks.get).toHaveBeenCalledWith('/api/kline', {
      params: { symbol: 'ETHUSDT', interval: '1h', limit: 100, endTime: 1_700_000_000_000 }
    })
  })

  it('subscribes to named watchlist price and shared stream status events', () => {
    const addEventListener = vi.fn()
    const listeners = new Map<string, (event: MessageEvent) => void>()
    class FakeEventSource {
      addEventListener(type: string, handler: (event: Event) => void) {
        addEventListener(type, handler)
        listeners.set(type, handler as (event: MessageEvent) => void)
      }
      close = vi.fn()
    }
    vi.stubGlobal('EventSource', FakeEventSource)
    const onPrice = vi.fn()
    const onStatus = vi.fn()

    const source = api.createWatchlistPriceStream(['BTCUSDT', 'ETHUSDT'], onPrice, onStatus)

    expect(addEventListener.mock.calls.map(call => call[0])).toEqual(['ready', 'price', 'status'])
    listeners.get('ready')?.({ data: JSON.stringify({ unavailable_symbols: ['MATICUSDT'] }) } as MessageEvent)
    expect(onStatus).toHaveBeenCalledWith({
      state: 'connecting', message: '正在连接实时行情', unavailable_symbols: ['MATICUSDT']
    })
    expect(source.close).toBeDefined()

    const unavailableStream = api.createWatchlistPriceStream(['MATICUSDT'], vi.fn(), vi.fn())
    listeners.get('ready')?.({
      data: JSON.stringify({ symbols: [], unavailable_symbols: ['MATICUSDT'] })
    } as MessageEvent)
    expect(unavailableStream.close).toHaveBeenCalledOnce()
    vi.unstubAllGlobals()
  })
})
