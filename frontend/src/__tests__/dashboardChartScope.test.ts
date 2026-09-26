// @vitest-environment jsdom

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import Dashboard from '../views/Dashboard.vue'

const { dashboardStore, apiMocks } = vi.hoisted(() => {
  const dashboardStore = {
    storageWarning: '',
    loading: false,
    analysisResult: { indicators: { atr: null } },
    error: null as string | null,
    symbol: 'BTCUSDT',
    interval: '1d',
    klineData: { symbol: 'BTCUSDT', interval: '1d', data: [], has_more_before: false },
    lastUpdate: new Date('2026-09-26T00:00:00Z'),
    trendDirection: 'neutral',
    trendStrength: 0,
    trendColor: '',
    invalidSymbol: false,
    priceStreamState: 'live',
    availableSymbols: [
      { value: 'BTCUSDT', label: 'BTC/USDT' },
      { value: 'ETHUSDT', label: 'ETH/USDT' }
    ],
    unavailableSymbols: [] as string[],
    pricesBySymbol: {
      BTCUSDT: { price: 65000, change24hPercent: 1 },
      ETHUSDT: { price: 3200, change24hPercent: 2 }
    },
    startWatchlistPriceStream: vi.fn(),
    stopWatchlistPriceStream: vi.fn(),
    fetchAnalysis: vi.fn(async () => undefined),
    fetchFallbackKline: vi.fn(async () => true),
    updateRealtimeCandle: vi.fn(),
    setSymbol: vi.fn(),
    setInterval: vi.fn(),
    reorderSymbols: vi.fn(),
    addCustomSymbol: vi.fn(async () => 'SOLUSDT'),
    removeSymbol: vi.fn()
  }
  const apiMocks = {
    createMarketStream: vi.fn(() => ({ close: vi.fn(), onerror: null as null | (() => void) })),
    validateSymbol: vi.fn(async (symbol: string) => symbol),
    invalidSelection: vi.fn(() => false),
    errorMessage: vi.fn((_error: unknown, fallback: string) => fallback)
  }
  return { dashboardStore, apiMocks }
})

vi.mock('../stores/analysis', () => ({ useAnalysisStore: () => dashboardStore }))
vi.mock('../services/api', () => ({ api: apiMocks }))
vi.mock('../components/SimpleChart.vue', async () => {
  const { defineComponent, h } = await import('vue')
  return {
    default: defineComponent({
      props: ['candles', 'symbol', 'interval', 'hasMoreBefore', 'symbols'],
      setup(props) {
        return () => h('div', {
          'data-testid': 'integrated-chart',
          'data-symbol': props.symbol,
          'data-interval': props.interval,
          'data-symbols': (props.symbols as string[] | undefined)?.join(',')
        })
      }
    })
  }
})

describe('Dashboard chart integration scope', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    dashboardStore.symbol = 'BTCUSDT'
    dashboardStore.interval = '1d'
    dashboardStore.klineData = { symbol: 'BTCUSDT', interval: '1d', data: [], has_more_before: false }
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('keeps the page header and left Watchlist while placing Pro in the right chart panel', async () => {
    const wrapper = mount(Dashboard, { global: { stubs: { 'el-icon': true, 'el-alert': true, Loading: true } } })
    await Promise.resolve()

    expect(wrapper.find('.header h1.title').text()).toContain('加密货币趋势分析系统')
    expect(wrapper.find('.watchlist-heading h2').text()).toBe('Watchlist')
    expect(wrapper.findAll('.watchlist-item')).toHaveLength(2)
    expect(wrapper.find('.interval-buttons').exists()).toBe(false)
    expect(wrapper.get('[data-testid="integrated-chart"]').attributes('data-symbol')).toBe('BTCUSDT')
    expect(wrapper.get('[data-testid="integrated-chart"]').attributes('data-symbols')).toBe('BTCUSDT,ETHUSDT')

    wrapper.unmount()
  })
})
