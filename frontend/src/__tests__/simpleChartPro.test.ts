// @vitest-environment jsdom

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import SimpleChart from '../components/SimpleChart.vue'

const { instances, constructors, coreChart, resizeObservers } = vi.hoisted(() => {
  const coreChart = {
    resize: vi.fn(),
    setSymbol: vi.fn(),
    setPeriod: vi.fn(),
    resetData: vi.fn(),
    setScrollEnabled: vi.fn(),
    setZoomEnabled: vi.fn(),
    setDataLoader: vi.fn(),
    getOverlays: vi.fn(() => []),
    createIndicator: vi.fn(() => 'pane-1'),
    overrideIndicator: vi.fn(),
    removeIndicator: vi.fn(),
    getIndicatorByPaneId: vi.fn(() => null),
    createOverlay: vi.fn(() => 'overlay-1'),
    getOverlayById: vi.fn(() => null),
    removeOverlay: vi.fn(),
    overrideOverlay: vi.fn()
  }
  const instances: Array<{ options: any; chart: any }> = []
  const resizeObservers: Array<{ callback: ResizeObserverCallback; observe: ReturnType<typeof vi.fn>; disconnect: ReturnType<typeof vi.fn> }> = []
  const constructors = vi.fn(function (options: any) {
    let currentSymbol = options.symbol
    let currentPeriod = options.period
    const chart = {
      setSymbol: vi.fn((symbol: any) => { currentSymbol = symbol }),
      getSymbol: vi.fn(() => currentSymbol),
      setPeriod: vi.fn((period: any) => { currentPeriod = period }),
      getPeriod: vi.fn(() => currentPeriod),
      getChart: vi.fn(() => coreChart),
      dispose: vi.fn()
    }
    instances.push({ options, chart })
    return chart
  })
  return { instances, constructors, coreChart, resizeObservers }
})

vi.mock('@klinecharts/pro', () => ({
  KLineChartPro: constructors,
  loadLocales: vi.fn()
}))

vi.mock('klinecharts', () => ({
  registerIndicator: vi.fn(),
  IndicatorSeries: { Normal: 'normal', Price: 'price', Volume: 'volume' },
  ActionType: { OnOverlaySelected: 'onOverlaySelected' },
  init: vi.fn(() => coreChart),
  dispose: vi.fn()
}))

vi.mock('../services/api', () => ({
  api: {
    getKlineData: vi.fn(async () => ({ data: [], has_more_before: false })),
    getOpenInterestData: vi.fn(async () => ({ data: [] })),
    errorMessage: vi.fn((_error, fallback) => fallback)
  }
}))

const candles = [{ timestamp: 60_000, open: 10, high: 12, low: 9, close: 11, volume: 5 }]

describe('SimpleChart Pro integration', () => {
  beforeEach(() => {
    instances.length = 0
    constructors.mockClear()
    resizeObservers.length = 0
    vi.stubGlobal('ResizeObserver', class {
      constructor(callback: ResizeObserverCallback) {
        resizeObservers.push({ callback, observe: vi.fn(), disconnect: vi.fn() })
      }
      observe(...args: unknown[]) { resizeObservers[resizeObservers.length - 1]?.observe(...args) }
      disconnect() { resizeObservers[resizeObservers.length - 1]?.disconnect() }
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('mounts Pro in the existing chart region with all Binance periods and the selected instrument', () => {
    mount(SimpleChart, { props: { candles, symbol: 'BTCUSDT', interval: '1m', symbols: ['BTCUSDT', 'ETHUSDT'] } })

    expect(constructors).toHaveBeenCalledOnce()
    expect(instances[0]?.options.symbol.ticker).toBe('BTCUSDT')
    expect(instances[0]?.options.period.text).toBe('1m')
    expect(instances[0]?.options.watermark).toBe('')
    expect(instances[0]?.options.periods.map((period: { text: string }) => period.text)).toEqual([
      '1m', '3m', '5m', '15m', '30m', '1h', '2h', '4h', '6h', '8h', '12h', '1d', '3d', '1w', '1M'
    ])
    expect(instances[0]?.options.datafeed).toBeDefined()
  })

  it('updates Pro when Watchlist selection or the selected interval changes', async () => {
    const wrapper = mount(SimpleChart, { props: { candles, symbol: 'BTCUSDT', interval: '1m', symbols: ['BTCUSDT', 'ETHUSDT'] } })
    const instance = instances[0]!

    await wrapper.setProps({ symbol: 'ETHUSDT', interval: '5m' })

    expect(instance.chart.setSymbol).toHaveBeenCalledWith(expect.objectContaining({ ticker: 'ETHUSDT' }))
    expect(instance.chart.setPeriod).toHaveBeenCalledWith(expect.objectContaining({ text: '5m' }))
  })

  it('releases the Pro and Datafeed subscriptions when Vue unmounts the chart', () => {
    const wrapper = mount(SimpleChart, { props: { candles, symbol: 'BTCUSDT', interval: '1m', symbols: ['BTCUSDT'] } })
    const datafeed = instances[0]?.options.datafeed
    const disposeDatafeed = vi.spyOn(datafeed, 'dispose')

    wrapper.unmount()

    expect(instances[0]?.chart.dispose).toHaveBeenCalledOnce()
    expect(disposeDatafeed).toHaveBeenCalledOnce()
    expect(resizeObservers[0]?.disconnect).toHaveBeenCalledOnce()
  })
})
