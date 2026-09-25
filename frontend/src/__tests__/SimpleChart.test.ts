// @vitest-environment jsdom

import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import SimpleChart from '../components/SimpleChart.vue'

const chartMocks = vi.hoisted(() => {
  const series = { setData: vi.fn(), update: vi.fn() }
  const timeScale = { fitContent: vi.fn() }
  const chart = {
    addSeries: vi.fn(() => series),
    remove: vi.fn(),
    resize: vi.fn(),
    timeScale: vi.fn(() => timeScale)
  }
  return {
    series,
    timeScale,
    chart,
    createChart: vi.fn((_container: HTMLElement, _options: unknown) => chart),
    candlestick: { type: 'candlestick' },
    colorType: { Solid: 'solid' },
    observers: [] as Array<{ callback: ResizeObserverCallback; disconnect: ReturnType<typeof vi.fn> }>
  }
})

vi.mock('lightweight-charts', () => ({
  createChart: chartMocks.createChart,
  CandlestickSeries: chartMocks.candlestick,
  ColorType: chartMocks.colorType
}))

const initialCandles = [
  { timestamp: 1_758_720_000_000, open: 99, high: 102, low: 98, close: 100, volume: 12 },
  { timestamp: 1_758_723_600_000, open: 100, high: 103, low: 99, close: 101, volume: 18 }
]

describe('Lightweight Charts candle view', () => {
  beforeEach(() => {
    chartMocks.series.setData.mockClear()
    chartMocks.series.update.mockClear()
    chartMocks.chart.remove.mockClear()
    chartMocks.createChart.mockClear()
    chartMocks.chart.resize.mockClear()
    chartMocks.timeScale.fitContent.mockClear()
    chartMocks.observers.length = 0
    vi.stubGlobal('ResizeObserver', class {
      observe = vi.fn()
      disconnect = vi.fn()
      constructor(callback: ResizeObserverCallback) {
        chartMocks.observers.push({ callback, disconnect: this.disconnect })
      }
    })
  })

  afterEach(() => vi.unstubAllGlobals())

  it('maps candle timestamps to Lightweight Charts and preserves the market summary and attribution', () => {
    const wrapper = mount(SimpleChart, {
      props: {
        symbol: 'ETHUSDT', interval: '1h', candles: initialCandles,
        atr: { value: 1.25, period: 14 }
      }
    })

    expect(chartMocks.createChart).toHaveBeenCalledOnce()
    expect(chartMocks.createChart.mock.calls[0]?.[1]).toEqual(expect.objectContaining({
      handleScroll: true, handleScale: true
    }))
    expect(chartMocks.series.setData).toHaveBeenCalledWith([
      { time: 1_758_720_000, open: 99, high: 102, low: 98, close: 100 },
      { time: 1_758_723_600, open: 100, high: 103, low: 99, close: 101 }
    ])
    expect(wrapper.find('.lightweight-chart').exists()).toBe(true)
    expect(wrapper.text()).toContain('ETH/USDT K线图')
    expect(wrapper.text()).toContain('ATR(14):')
    expect(wrapper.find('.sr-lines, .level-badge').exists()).toBe(false)
    wrapper.unmount()
    expect(chartMocks.chart.remove).toHaveBeenCalledOnce()
  })

  it('updates the active candle through the series update API', async () => {
    const wrapper = mount(SimpleChart, {
      props: { symbol: 'ETHUSDT', interval: '15m', candles: initialCandles }
    })
    const updated = [...initialCandles, {
      timestamp: 1_758_727_200_000, open: 101, high: 104, low: 100, close: 103, volume: 8
    }]
    await wrapper.setProps({ candles: updated })

    expect(chartMocks.series.update).toHaveBeenCalledWith({
      time: 1_758_727_200, open: 101, high: 104, low: 100, close: 103
    })
  })

  it('replaces data and fits the view when symbol or interval changes', async () => {
    const wrapper = mount(SimpleChart, {
      props: { symbol: 'ETHUSDT', interval: '15m', candles: initialCandles }
    })
    chartMocks.series.setData.mockClear()

    await wrapper.setProps({ symbol: 'BTCUSDT', interval: '4h' })

    expect(chartMocks.series.setData).toHaveBeenLastCalledWith([
      { time: 1_758_720_000, open: 99, high: 102, low: 98, close: 100 },
      { time: 1_758_723_600, open: 100, high: 103, low: 99, close: 101 }
    ])
    expect(chartMocks.timeScale.fitContent).toHaveBeenCalled()
    expect(wrapper.text()).toContain('BTC/USDT K线图')
    wrapper.unmount()
  })

  it('resizes with its container and disconnects the observer on unmount', () => {
    const wrapper = mount(SimpleChart, {
      props: { symbol: 'ETHUSDT', interval: '1h', candles: initialCandles }
    })
    const observer = chartMocks.observers[0]!
    observer.callback([{ contentRect: { width: 320, height: 240 } } as ResizeObserverEntry], {} as ResizeObserver)

    expect(chartMocks.chart.resize).toHaveBeenCalledWith(320, 240)
    wrapper.unmount()
    expect(observer.disconnect).toHaveBeenCalledOnce()
  })
})
