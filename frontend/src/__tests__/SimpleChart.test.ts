// @vitest-environment jsdom

import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import SimpleChart from '../components/SimpleChart.vue'

const chartMocks = vi.hoisted(() => {
  const registered: { loader: unknown } = { loader: null }
  const observers: Array<{ observe: ReturnType<typeof vi.fn>; disconnect: ReturnType<typeof vi.fn> }> = []
  const chart = {
    setDataLoader: vi.fn((loader: unknown) => { registered.loader = loader }),
    setSymbol: vi.fn(),
    setPeriod: vi.fn(),
    setStyles: vi.fn(),
    setScrollEnabled: vi.fn(),
    setZoomEnabled: vi.fn(),
    resize: vi.fn()
  }
  return {
    registered,
    observers,
    chart,
    init: vi.fn(() => chart),
    dispose: vi.fn(),
    getKlineData: vi.fn()
  }
})

vi.mock('klinecharts', () => ({ init: chartMocks.init, dispose: chartMocks.dispose }))
vi.mock('../services/api', async importOriginal => {
  const actual = await importOriginal<typeof import('../services/api')>()
  return { ...actual, api: { ...actual.api, getKlineData: chartMocks.getKlineData } }
})

const initialCandles = [
  { timestamp: 1_758_720_000_000, open: 99, high: 102, low: 98, close: 100, volume: 12 },
  { timestamp: 1_758_723_600_000, open: 100, high: 103, low: 99, close: 101, volume: 18 }
]

describe('KLineChart candle view', () => {
  beforeEach(() => {
    chartMocks.chart.setDataLoader.mockClear()
    chartMocks.chart.setSymbol.mockClear()
    chartMocks.chart.setPeriod.mockClear()
    chartMocks.init.mockClear()
    chartMocks.dispose.mockClear()
    chartMocks.registered.loader = null
    chartMocks.observers.length = 0
    chartMocks.getKlineData.mockReset()
    vi.stubGlobal('ResizeObserver', class {
      observe = vi.fn()
      disconnect = vi.fn()
      constructor(_callback: ResizeObserverCallback) { chartMocks.observers.push(this) }
    })
  })

  afterEach(() => vi.unstubAllGlobals())

  it('initializes KLineChart with Binance symbol, mapped period and millisecond candles', async () => {
    const wrapper = mount(SimpleChart, {
      props: {
        symbol: 'ETHUSDT', interval: '1h', candles: initialCandles,
        hasMoreBefore: true, atr: { value: 1.25, period: 14 }
      }
    })

    expect(chartMocks.init).toHaveBeenCalledOnce()
    expect(chartMocks.chart.setSymbol).toHaveBeenCalledWith(expect.objectContaining({ ticker: 'ETHUSDT' }))
    expect(chartMocks.chart.setPeriod).toHaveBeenCalledWith({ span: 1, type: 'hour' })
    const loader = chartMocks.registered.loader as {
      getBars: (params: { type: string; symbol: { ticker: string }; callback: (bars: unknown[], more?: unknown) => void }) => Promise<void>
    }
    const callback = vi.fn()
    await loader.getBars({ type: 'init', symbol: { ticker: 'ETHUSDT' }, callback })
    expect(callback).toHaveBeenCalledWith(initialCandles, { forward: true, backward: false })
    expect(wrapper.find('.kline-chart').exists()).toBe(true)
    expect(wrapper.find('.lightweight-chart').exists()).toBe(false)
    expect(wrapper.text()).toContain('ETH/USDT K线图')
    expect(wrapper.text()).toContain('ATR(14):')
    wrapper.unmount()
    expect(chartMocks.dispose).toHaveBeenCalledOnce()
  })

  it('loads older bars through the exclusive cursor and reports page exhaustion', async () => {
    chartMocks.getKlineData.mockResolvedValue({
      symbol: 'ETHUSDT', interval: '1h', data: [{ ...initialCandles[0]!, timestamp: 1_758_720_000_000 - 3_600_000 }],
      has_more_before: false
    })
    const wrapper = mount(SimpleChart, {
      props: { symbol: 'ETHUSDT', interval: '1h', candles: initialCandles, hasMoreBefore: true }
    })
    const loader = chartMocks.registered.loader as {
      getBars: (params: { type: string; symbol: { ticker: string }; callback: (bars: unknown[], more?: unknown) => void }) => Promise<void>
    }
    const callback = vi.fn()
    await loader.getBars({ type: 'forward', symbol: { ticker: 'ETHUSDT' }, callback })

    expect(chartMocks.getKlineData).toHaveBeenCalledWith(
      'ETHUSDT', '1h', 100, initialCandles[0]!.timestamp - 1, expect.any(AbortSignal)
    )
    expect(callback).toHaveBeenCalledWith(
      [{ ...initialCandles[0]!, timestamp: 1_758_720_000_000 - 3_600_000 }],
      { forward: false, backward: false }
    )
    wrapper.unmount()
  })

  it('keeps loaded bars after a page error and lets another drag retry the request', async () => {
    chartMocks.getKlineData
      .mockRejectedValueOnce(new Error('temporary history failure'))
      .mockResolvedValueOnce({
        symbol: 'ETHUSDT', interval: '1h', data: [{ ...initialCandles[0]!, timestamp: initialCandles[0]!.timestamp - 3_600_000 }],
        has_more_before: false
      })
    const wrapper = mount(SimpleChart, {
      props: { symbol: 'ETHUSDT', interval: '1h', candles: initialCandles, hasMoreBefore: true }
    })
    const loader = chartMocks.registered.loader as {
      getBars: (params: { type: string; symbol: { ticker: string }; callback: (bars: unknown[], more?: unknown) => void }) => Promise<void>
    }
    const callback = vi.fn()

    await loader.getBars({ type: 'forward', symbol: { ticker: 'ETHUSDT' }, callback })
    expect(wrapper.find('[role="alert"]').text()).toContain('temporary history failure')
    expect(callback).toHaveBeenNthCalledWith(1, [], { forward: true, backward: false })

    await loader.getBars({ type: 'forward', symbol: { ticker: 'ETHUSDT' }, callback })
    expect(chartMocks.getKlineData).toHaveBeenCalledTimes(2)
    expect(callback).toHaveBeenNthCalledWith(
      2, [{ ...initialCandles[0]!, timestamp: initialCandles[0]!.timestamp - 3_600_000 }],
      { forward: false, backward: false }
    )
    wrapper.unmount()
  })

  it('publishes store-driven real-time changes and disposes observer and chart on unmount', async () => {
    const wrapper = mount(SimpleChart, {
      props: { symbol: 'ETHUSDT', interval: '15m', candles: initialCandles }
    })
    const loader = chartMocks.registered.loader as {
      subscribeBar: (params: { callback: (bar: unknown) => void }) => void
      unsubscribeBar: (params: { symbol: { ticker: string }; period: { span: number; type: string } }) => void
    }
    const onBar = vi.fn()
    loader.subscribeBar({ callback: onBar })
    await wrapper.setProps({ candles: [...initialCandles.slice(0, 1), { ...initialCandles[1]!, close: 105 }] })
    expect(onBar).toHaveBeenCalledWith({ ...initialCandles[1]!, close: 105 })
    loader.unsubscribeBar({ symbol: { ticker: 'ETHUSDT' }, period: { span: 15, type: 'minute' } })
    await wrapper.setProps({ candles: [...initialCandles.slice(0, 1), { ...initialCandles[1]!, close: 106 }] })
    expect(onBar).toHaveBeenCalledOnce()

    const observer = chartMocks.observers[0]!
    wrapper.unmount()
    expect(chartMocks.dispose).toHaveBeenCalledOnce()
    expect(observer.disconnect).toHaveBeenCalledOnce()
  })
})
