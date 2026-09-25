// @vitest-environment jsdom

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'
import Dashboard from '../views/Dashboard.vue'

type TestMarketEvent = {
  symbol: string
  interval: string
  candle: { timestamp: number; open: number; high: number; low: number; close: number; volume: number }
  is_final: boolean
  event_time: number
}

const streamHandlers = vi.hoisted(() => ({ handlers: [] as Array<(event: TestMarketEvent) => void> }))

const mocks = vi.hoisted(() => ({
  api: {
    createMarketStream: vi.fn((_symbol: string, _interval: string, onEvent: (event: TestMarketEvent) => void) => {
      streamHandlers.handlers.push(onEvent)
      return { addEventListener: vi.fn(), close: vi.fn() }
    }),
    getWatchlistPrices: vi.fn(async () => []),
    createWatchlistPriceStream: vi.fn(() => ({ addEventListener: vi.fn(), close: vi.fn() })),
    validateSymbol: vi.fn().mockResolvedValue('ETHUSDT'),
    errorMessage: vi.fn((_error: unknown, fallback: string) => fallback),
    invalidSelection: vi.fn(() => false)
  },
  store: {
    symbol: 'ETHUSDT',
    interval: '1d',
    availableSymbols: [
      { label: 'BTC/USDT', value: 'BTCUSDT', icon: '₿' },
      { label: 'ETH/USDT', value: 'ETHUSDT', icon: 'Ξ' }
    ],
    customSymbols: [],
    pricesBySymbol: {
      BTCUSDT: { price: 67420.5, change24hPercent: 0.5, eventTime: 1 },
      ETHUSDT: { price: 3521.75, change24hPercent: -0.25, eventTime: 1 }
    },
    unavailableSymbols: [],
    priceStreamState: 'live',
    startWatchlistPriceStream: vi.fn(),
    stopWatchlistPriceStream: vi.fn(),
    storageWarning: null,
    loading: false,
    error: null,
    invalidSymbol: false,
    lastUpdate: new Date('2026-09-25T00:00:00Z'),
    klineData: { data: [] },
    analysisResult: {
      sr_levels: { resistance: [], support: [] },
      indicators: { ema: {}, fibonacci: {}, atr: null },
      trend: {},
      candlestick_patterns: [],
      market_structure: {}
    },
    trendColor: '#26a69a',
    fetchAnalysis: vi.fn().mockResolvedValue(undefined),
    fetchFallbackKline: vi.fn().mockResolvedValue(true),
    updateRealtimeCandle: vi.fn(),
    setSymbol: vi.fn(),
    setInterval: vi.fn(),
    addCustomSymbol: vi.fn(),
    removeSymbol: vi.fn()
  }
}))

vi.mock('../services/api', () => ({ api: mocks.api }))
vi.mock('../stores/analysis', () => ({ useAnalysisStore: () => mocks.store }))

describe('Dashboard watchlist and market controls', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    streamHandlers.handlers.length = 0
    Object.assign(mocks.store, {
      symbol: 'ETHUSDT',
      interval: '1d',
      availableSymbols: [
        { label: 'BTC/USDT', value: 'BTCUSDT', icon: '₿' },
        { label: 'ETH/USDT', value: 'ETHUSDT', icon: 'Ξ' }
      ],
      customSymbols: [],
      pricesBySymbol: {
        BTCUSDT: { price: 67420.5, change24hPercent: 0.5, eventTime: 1 },
        ETHUSDT: { price: 3521.75, change24hPercent: -0.25, eventTime: 1 }
      },
      unavailableSymbols: [],
      priceStreamState: 'live',
      storageWarning: null,
      loading: false,
      error: null,
      invalidSymbol: false,
      lastUpdate: new Date('2026-09-25T00:00:00Z'),
      klineData: { data: [] },
      analysisResult: {
        sr_levels: { resistance: [], support: [] },
        indicators: { ema: {}, fibonacci: {}, atr: null },
        trend: {},
        candlestick_patterns: [],
        market_structure: {}
      }
    })
    mocks.store.setSymbol.mockImplementation((symbol: string) => { mocks.store.symbol = symbol })
    mocks.store.setInterval.mockImplementation((interval: string) => { mocks.store.interval = interval })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('shows all symbols with live prices in the watchlist and removes the header symbol dropdown', async () => {
    const wrapper = shallowMount(Dashboard, {
      global: { stubs: { 'el-icon': true, 'el-alert': true, Loading: true } }
    })
    await flushPromises()

    expect(wrapper.find('.symbol-select').exists()).toBe(false)
    expect(wrapper.findAll('.watchlist-item')).toHaveLength(2)
    expect(wrapper.find('.tradingview-attribution').exists()).toBe(false)
    expect(wrapper.text()).toContain('BTC/USDT')
    expect(wrapper.text()).toContain('67,420.5')
    expect(wrapper.text()).toContain('ETH/USDT')
    expect(wrapper.text()).toContain('3,521.75')
    expect(wrapper.find('.watchlist-item.active').text()).toContain('ETH/USDT')
    expect(wrapper.find('.header .interval-buttons').exists()).toBe(false)
    expect(wrapper.find('.chart-section .interval-buttons').exists()).toBe(true)
    expect(wrapper.findAll('.chart-section .interval-btn')).toHaveLength(15)
    expect(wrapper.find('.stream-status').exists()).toBe(true)
    expect(wrapper.find('.panels-grid').exists()).toBe(false)
    expect(wrapper.find('.opportunity-button').exists()).toBe(false)
    expect(wrapper.find('.modal-overlay').exists()).toBe(false)

    wrapper.unmount()
    expect(mocks.store.stopWatchlistPriceStream).toHaveBeenCalledOnce()
    expect(mocks.api.createMarketStream.mock.results[0]?.value.close).toHaveBeenCalledOnce()
  })

  it('shows 24h percent changes with zero and positive in green and negative in red', async () => {
    Object.assign(mocks.store, {
      availableSymbols: [
        { label: 'BTC/USDT', value: 'BTCUSDT', icon: '₿' },
        { label: 'ETH/USDT', value: 'ETHUSDT', icon: 'Ξ' },
        { label: 'SOL/USDT', value: 'SOLUSDT', icon: '◎' }
      ],
      pricesBySymbol: {
        BTCUSDT: { price: 67420.5, change24hPercent: 1.25, eventTime: 1 },
        ETHUSDT: { price: 3521.75, change24hPercent: -2.5, eventTime: 1 },
        SOLUSDT: { price: 142.5, change24hPercent: 0, eventTime: 1 }
      }
    })
    const wrapper = shallowMount(Dashboard, {
      global: { stubs: { 'el-icon': true, 'el-alert': true, Loading: true } }
    })

    const change = (symbol: string) => wrapper.find(`.watchlist-item[data-symbol="${symbol}"] .watchlist-change`)
    expect(change('BTCUSDT').text()).toBe('+1.25%')
    expect(change('BTCUSDT').classes()).toContain('positive')
    expect(change('ETHUSDT').text()).toBe('-2.50%')
    expect(change('ETHUSDT').classes()).toContain('negative')
    expect(change('SOLUSDT').text()).toBe('0.00%')
    expect(change('SOLUSDT').classes()).toContain('positive')

    wrapper.unmount()
  })

  it('selects a watchlist symbol and keeps add and delete controls in the watchlist', async () => {
    const wrapper = shallowMount(Dashboard, {
      global: { stubs: { 'el-icon': true, 'el-alert': true, Loading: true } }
    })
    await flushPromises()

    await wrapper.find('.watchlist-item[data-symbol="BTCUSDT"] .watchlist-select').trigger('click')
    expect(mocks.store.setSymbol).toHaveBeenCalledWith('BTCUSDT')

    const addButton = wrapper.find('button[title="添加自选交易对"]')
    expect(addButton.exists()).toBe(true)
    expect(addButton.element.closest('.watchlist')).not.toBeNull()
    const deleteButton = wrapper.find('.watchlist-item[data-symbol="ETHUSDT"] button[title="删除交易对"]')
    expect(deleteButton.exists()).toBe(true)
    expect(deleteButton.element.closest('.watchlist')).not.toBeNull()
    await deleteButton.trigger('click')
    expect(mocks.store.removeSymbol).toHaveBeenCalledWith('ETHUSDT')

    wrapper.unmount()
  })

  it('changes to a native Binance interval from controls above the chart and replaces the stream', async () => {
    const wrapper = shallowMount(Dashboard, {
      global: { stubs: { 'el-icon': true, 'el-alert': true, Loading: true } }
    })
    await flushPromises()
    const firstStream = mocks.api.createMarketStream.mock.results[0]?.value as { close: ReturnType<typeof vi.fn> }

    const threeMinuteButton = wrapper.find('.chart-section .interval-btn[data-interval="3m"]')
    expect(threeMinuteButton.exists()).toBe(true)
    await threeMinuteButton.trigger('click')
    await flushPromises()

    expect(mocks.store.setInterval).toHaveBeenCalledWith('3m')
    expect(mocks.api.createMarketStream).toHaveBeenLastCalledWith('ETHUSDT', '3m', expect.any(Function), expect.any(Function))
    expect(firstStream.close).toHaveBeenCalledOnce()
    wrapper.unmount()
  })

  it('ignores updates from the previous kline subscription after selecting another symbol', async () => {
    const wrapper = shallowMount(Dashboard, {
      global: { stubs: { 'el-icon': true, 'el-alert': true, Loading: true } }
    })
    await flushPromises()
    expect(streamHandlers.handlers).toHaveLength(1)

    await wrapper.find('.watchlist-item[data-symbol="BTCUSDT"] .watchlist-select').trigger('click')
    await flushPromises()
    expect(mocks.store.symbol).toBe('BTCUSDT')
    expect(streamHandlers.handlers).toHaveLength(2)

    const staleEvent: TestMarketEvent = {
      symbol: 'ETHUSDT', interval: '1d',
      candle: { timestamp: 1, open: 1, high: 2, low: 1, close: 2, volume: 3 },
      is_final: false, event_time: 1
    }
    streamHandlers.handlers[0]!(staleEvent)
    expect(mocks.store.updateRealtimeCandle).not.toHaveBeenCalled()

    streamHandlers.handlers[1]!({ ...staleEvent, symbol: 'BTCUSDT' })
    expect(mocks.store.updateRealtimeCandle).toHaveBeenCalledOnce()
    wrapper.unmount()
  })

})
