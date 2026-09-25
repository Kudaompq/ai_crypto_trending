// @vitest-environment jsdom

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'
import Dashboard from '../views/Dashboard.vue'

const mocks = vi.hoisted(() => ({
  api: {
    createMarketStream: vi.fn(() => ({ addEventListener: vi.fn(), close: vi.fn() })),
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

describe('Dashboard retired analysis and opportunity features', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.assign(mocks.store, {
      symbol: 'ETHUSDT',
      interval: '1d',
      availableSymbols: [
        { label: 'BTC/USDT', value: 'BTCUSDT', icon: '₿' },
        { label: 'ETH/USDT', value: 'ETHUSDT', icon: 'Ξ' }
      ],
      customSymbols: [],
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
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('keeps market controls while omitting every analysis panel and the opportunity entry', async () => {
    const wrapper = shallowMount(Dashboard, {
      global: { stubs: { 'el-icon': true, 'el-alert': true, Loading: true } }
    })
    await flushPromises()

    expect(wrapper.find('.symbol-select').exists()).toBe(true)
    expect(wrapper.find('.symbol-select').findAll('option').map(option => option.text().trim()))
      .toEqual(['BTC/USDT', 'ETH/USDT'])
    expect(wrapper.findAll('.interval-btn')).toHaveLength(4)
    expect(wrapper.find('.stream-status').exists()).toBe(true)
    expect(wrapper.find('.panels-grid').exists()).toBe(false)
    expect(wrapper.find('.opportunity-button').exists()).toBe(false)
    expect(wrapper.find('.modal-overlay').exists()).toBe(false)

    wrapper.unmount()
  })

  it('allows deleting a selected preset symbol', async () => {
    const wrapper = shallowMount(Dashboard, {
      global: { stubs: { 'el-icon': true, 'el-alert': true, Loading: true } }
    })
    await flushPromises()

    const deleteButton = wrapper.find('button[title="删除当前交易对"]')
    expect(deleteButton.exists()).toBe(true)
    await deleteButton.trigger('click')
    expect(mocks.store.removeSymbol).toHaveBeenCalledWith('ETHUSDT')

    wrapper.unmount()
  })

})
