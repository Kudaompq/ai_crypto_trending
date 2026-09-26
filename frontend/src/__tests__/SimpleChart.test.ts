// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import SimpleChart from '../components/SimpleChart.vue'
import { CHART_PREFERENCES_STORAGE_KEY } from '../services/chartPreferences'

const chartMocks = vi.hoisted(() => {
  const registered: { loader: unknown; openInterestIndicator: unknown; overlays: Array<Record<string, unknown>>; nextOverlayId: number } = {
    loader: null, openInterestIndicator: null, overlays: [], nextOverlayId: 1
  }
  const observers: Array<{ observe: ReturnType<typeof vi.fn>; disconnect: ReturnType<typeof vi.fn> }> = []
  const chart = {
    setDataLoader: vi.fn((loader: unknown) => { registered.loader = loader }),
    setSymbol: vi.fn(),
    setPeriod: vi.fn(),
    setStyles: vi.fn(),
    setScrollEnabled: vi.fn(),
    setZoomEnabled: vi.fn(),
    resetData: vi.fn(),
    createIndicator: vi.fn(() => 'indicator-id'),
    removeIndicator: vi.fn(),
    overrideIndicator: vi.fn(),
    createOverlay: vi.fn((value: unknown) => {
      const overlay = typeof value === 'string' ? { name: value } : value as Record<string, unknown>
      const id = typeof overlay.id === 'string' ? overlay.id : `drawing-${registered.nextOverlayId++}`
      registered.overlays.push({ ...overlay, id, points: overlay.points || [] })
      return id
    }),
    getOverlays: vi.fn(() => registered.overlays),
    removeOverlay: vi.fn((filter?: { id?: string }) => {
      if (!filter?.id) registered.overlays = []
      else registered.overlays = registered.overlays.filter(overlay => overlay.id !== filter.id)
      return true
    }),
    resize: vi.fn()
  }
  return {
    registered,
    observers,
    chart,
    init: vi.fn(() => chart),
    dispose: vi.fn(),
    getKlineData: vi.fn(),
    getOpenInterestData: vi.fn()
  }
})

vi.mock('klinecharts', () => ({
  init: chartMocks.init,
  dispose: chartMocks.dispose,
  registerIndicator: vi.fn((indicator: unknown) => { chartMocks.registered.openInterestIndicator = indicator })
}))
vi.mock('../services/api', async importOriginal => {
  const actual = await importOriginal<typeof import('../services/api')>()
  return { ...actual, api: { ...actual.api, getKlineData: chartMocks.getKlineData, getOpenInterestData: chartMocks.getOpenInterestData } }
})

const initialCandles = [
  { timestamp: 1_758_720_000_000, open: 99, high: 102, low: 98, close: 100, volume: 12 },
  { timestamp: 1_758_723_600_000, open: 100, high: 103, low: 99, close: 101, volume: 18 }
]
const fiveMinuteCandles = [
  { timestamp: 300_000, open: 99, high: 102, low: 98, close: 100, volume: 12 },
  { timestamp: 600_000, open: 100, high: 103, low: 99, close: 101, volume: 18 }
]

describe('KLineChart candle view', () => {
  beforeEach(() => {
    localStorage.clear()
    chartMocks.chart.setDataLoader.mockClear()
    chartMocks.chart.setSymbol.mockClear()
    chartMocks.chart.setPeriod.mockClear()
    chartMocks.chart.resetData.mockClear()
    chartMocks.chart.createIndicator.mockClear()
    chartMocks.chart.removeIndicator.mockClear()
    chartMocks.chart.createOverlay.mockClear()
    chartMocks.chart.getOverlays.mockClear()
    chartMocks.chart.removeOverlay.mockClear()
    chartMocks.init.mockClear()
    chartMocks.dispose.mockClear()
    chartMocks.registered.loader = null
    chartMocks.registered.overlays = []
    chartMocks.registered.nextOverlayId = 1
    chartMocks.observers.length = 0
    chartMocks.getKlineData.mockReset()
    chartMocks.getOpenInterestData.mockReset()
    vi.stubGlobal('ResizeObserver', class {
      observe = vi.fn()
      disconnect = vi.fn()
      constructor(_callback: ResizeObserverCallback) { chartMocks.observers.push(this) }
    })
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

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

  it('restores shared enabled indicators when the chart initializes', () => {
    localStorage.setItem(CHART_PREFERENCES_STORAGE_KEY, JSON.stringify({
      version: 2,
      studies: {
        MA: { enabled: true, params: [7, 14, 28] },
        MACD: { enabled: true, params: [8, 21, 5] }
      },
      symbols: {}
    }))

    const wrapper = mount(SimpleChart, {
      props: { symbol: 'ETHUSDT', interval: '1h', candles: initialCandles }
    })

    expect(chartMocks.chart.createIndicator).toHaveBeenCalledWith(
      { name: 'MA', calcParams: [7, 14, 28], paneId: 'candle_pane' }, true
    )
    expect(chartMocks.chart.createIndicator).toHaveBeenCalledWith(
      { name: 'MACD', calcParams: [8, 21, 5] }
    )
    wrapper.unmount()
  })

  it('lets the user enable a price-pane study and persists it for all symbols', async () => {
    const wrapper = mount(SimpleChart, {
      props: { symbol: 'ETHUSDT', interval: '1h', candles: initialCandles }
    })
    const settingsButton = wrapper.find('[aria-label="指标设置"]')
    expect(settingsButton.exists()).toBe(true)
    await settingsButton.trigger('click')
    const ma = wrapper.find('input[aria-label="MA"]')
    await ma.setValue(true)

    expect(chartMocks.chart.createIndicator).toHaveBeenCalledWith(
      { name: 'MA', calcParams: [5, 10, 30, 60], paneId: 'candle_pane' }, true
    )
    const stored = JSON.parse(localStorage.getItem(CHART_PREFERENCES_STORAGE_KEY) || '{}')
    expect(stored.studies.MA).toEqual({ enabled: true, params: [5, 10, 30, 60] })
    expect(stored.symbols.ETHUSDT).toEqual({ drawings: [] })
    expect(stored.symbols.BTCUSDT).toBeUndefined()
    wrapper.unmount()
  })

  it('keeps a selected study enabled after switching to another symbol', async () => {
    const wrapper = mount(SimpleChart, {
      props: { symbol: 'ETHUSDT', interval: '1h', candles: initialCandles }
    })
    await wrapper.find('[aria-label="指标设置"]').trigger('click')
    await wrapper.find('input[aria-label="MA"]').setValue(true)

    await wrapper.setProps({ symbol: 'BTCUSDT' })
    await flushPromises()
    expect((wrapper.find('input[aria-label="MA"]').element as HTMLInputElement).checked).toBe(true)
    wrapper.unmount()
  })

  it('restores the same shared study after symbol and interval changes', () => {
    localStorage.setItem(CHART_PREFERENCES_STORAGE_KEY, JSON.stringify({
      version: 2,
      studies: { MA: { enabled: true, params: [5, 10, 30, 60] } },
      symbols: {}
    }))

    const ethChart = mount(SimpleChart, {
      props: { symbol: 'ETHUSDT', interval: '1h', candles: initialCandles }
    })
    expect(chartMocks.chart.createIndicator).toHaveBeenCalledWith(
      { name: 'MA', calcParams: [5, 10, 30, 60], paneId: 'candle_pane' }, true
    )
    ethChart.unmount()

    chartMocks.chart.createIndicator.mockClear()
    const btcChart = mount(SimpleChart, {
      props: { symbol: 'BTCUSDT', interval: '5m', candles: initialCandles }
    })
    expect(chartMocks.chart.createIndicator).toHaveBeenCalledWith(
      { name: 'MA', calcParams: [5, 10, 30, 60], paneId: 'candle_pane' }, true
    )
    const createdIndicatorArguments = chartMocks.chart.createIndicator.mock.calls as unknown as unknown[][]
    expect(createdIndicatorArguments.flat().some((value: unknown) =>
      typeof value === 'object' && value !== null && (value as { name?: unknown }).name === 'MA'
    )).toBe(true)
    btcChart.unmount()

    chartMocks.chart.createIndicator.mockClear()
    const remountedEth = mount(SimpleChart, {
      props: { symbol: 'ETHUSDT', interval: '5m', candles: initialCandles }
    })
    expect(chartMocks.chart.createIndicator).toHaveBeenCalledWith(
      { name: 'MA', calcParams: [5, 10, 30, 60], paneId: 'candle_pane' }, true
    )
    remountedEth.unmount()
  })

  it('loads Binance OI for the selected chart range and leaves missing samples empty', async () => {
    localStorage.setItem(CHART_PREFERENCES_STORAGE_KEY, JSON.stringify({
      version: 2,
      studies: { OPEN_INTEREST: { enabled: true, params: [] } },
      symbols: {}
    }))
    chartMocks.getOpenInterestData.mockResolvedValue({
      symbol: 'ETHUSDT', interval: '5m',
      data: [{ timestamp: 300_000, quantity: 10, value: 1000 }]
    })

    const wrapper = mount(SimpleChart, {
      props: { symbol: 'ETHUSDT', interval: '5m', candles: fiveMinuteCandles }
    })
    await flushPromises()

    expect(chartMocks.getOpenInterestData).toHaveBeenCalledWith(
      'ETHUSDT', '5m', 500, 300_000, 900_000, expect.any(AbortSignal)
    )
    expect(chartMocks.chart.createIndicator).toHaveBeenCalledWith(expect.objectContaining({ name: 'OPEN_INTEREST' }))
    const indicator = chartMocks.registered.openInterestIndicator as {
      calc: (bars: typeof fiveMinuteCandles, instance: { extendData: unknown }) => unknown[]
    }
    expect(indicator.calc(fiveMinuteCandles, {
      extendData: { samples: [{ timestamp: 300_000, quantity: 10, value: 1000 }], interval: '5m' }
    })).toEqual([{ value: 1000 }, { value: null }])
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('registers an OI indicator that refreshes when the Binance sample set changes', () => {
    const wrapper = mount(SimpleChart, {
      props: { symbol: 'ETHUSDT', interval: '5m', candles: fiveMinuteCandles }
    })
    const indicator = chartMocks.registered.openInterestIndicator as {
      shouldUpdate?: (previous: { extendData: unknown }, current: { extendData: unknown }) => boolean
    }
    expect(indicator.shouldUpdate).toEqual(expect.any(Function))
    expect(indicator.shouldUpdate?.(
      { extendData: { samples: [], interval: '5m' } },
      { extendData: { samples: [{ timestamp: 1, quantity: 1, value: 1 }], interval: '5m' } }
    )).toBe(true)
    wrapper.unmount()
  })

  it('keeps unsupported intervals and empty retained ranges blank without showing request errors', async () => {
    localStorage.setItem(CHART_PREFERENCES_STORAGE_KEY, JSON.stringify({
      version: 2,
      studies: { OPEN_INTEREST: { enabled: true, params: [] } },
      symbols: {}
    }))
    chartMocks.getOpenInterestData.mockResolvedValue({ symbol: 'ETHUSDT', interval: '5m', data: [] })

    const unsupported = mount(SimpleChart, {
      props: { symbol: 'ETHUSDT', interval: '1m', candles: fiveMinuteCandles }
    })
    await flushPromises()
    expect(chartMocks.getOpenInterestData).not.toHaveBeenCalled()
    expect(unsupported.find('[role="alert"]').exists()).toBe(false)
    unsupported.unmount()

    const outsideRetention = mount(SimpleChart, {
      props: { symbol: 'ETHUSDT', interval: '5m', candles: initialCandles }
    })
    await flushPromises()
    expect(chartMocks.getOpenInterestData).toHaveBeenCalledWith(
      'ETHUSDT', '5m', 500, initialCandles[0]!.timestamp, initialCandles[1]!.timestamp + 300_000,
      expect.any(AbortSignal)
    )
    expect(outsideRetention.find('[role="alert"]').exists()).toBe(false)
    outsideRetention.unmount()
  })

  it('shows recoverable OI request errors', async () => {
    localStorage.setItem(CHART_PREFERENCES_STORAGE_KEY, JSON.stringify({
      version: 2,
      studies: { OPEN_INTEREST: { enabled: true, params: [] } },
      symbols: {}
    }))
    chartMocks.getOpenInterestData
      .mockRejectedValueOnce(new Error('temporary OI failure'))
      .mockResolvedValueOnce({ symbol: 'ETHUSDT', interval: '5m', data: [] })
    const wrapper = mount(SimpleChart, {
      props: { symbol: 'ETHUSDT', interval: '5m', candles: fiveMinuteCandles }
    })
    await flushPromises()
    expect(wrapper.find('[role="alert"]').text()).toContain('OI 加载失败')
    await wrapper.find('.history-error button').trigger('click')
    await flushPromises()
    expect(chartMocks.getOpenInterestData).toHaveBeenCalledTimes(2)
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('aborts and ignores an OI response after switching symbols', async () => {
    localStorage.setItem(CHART_PREFERENCES_STORAGE_KEY, JSON.stringify({
      version: 2,
      studies: { OPEN_INTEREST: { enabled: true, params: [] } },
      symbols: {}
    }))
    let resolveOldRequest: ((value: unknown) => void) | undefined
    chartMocks.getOpenInterestData.mockImplementationOnce(() => new Promise(resolve => { resolveOldRequest = resolve }))
    const wrapper = mount(SimpleChart, {
      props: { symbol: 'ETHUSDT', interval: '5m', candles: fiveMinuteCandles }
    })
    await flushPromises()
    const requestResults = chartMocks.getOpenInterestData.mock.results
    const requestCalls = chartMocks.getOpenInterestData.mock.calls
    const oldRequest = requestResults[requestResults.length - 1]!.value as Promise<unknown>
    const oldSignal = requestCalls[requestCalls.length - 1]![5] as AbortSignal
    await wrapper.setProps({ symbol: 'BTCUSDT', interval: '1m', candles: fiveMinuteCandles })

    expect(oldSignal.aborted).toBe(true)
    chartMocks.chart.overrideIndicator.mockClear()
    resolveOldRequest?.({ symbol: 'ETHUSDT', interval: '15m', data: [{ timestamp: 1, quantity: 1, value: 999 }] })
    await oldRequest
    await flushPromises()
    expect(chartMocks.chart.overrideIndicator).not.toHaveBeenCalledWith(expect.objectContaining({
      extendData: expect.objectContaining({ samples: expect.arrayContaining([expect.objectContaining({ value: 999 })]) })
    }))
    wrapper.unmount()
  })

  it('creates, adjusts, persists, and deletes chart drawings using their time and price anchors', async () => {
    vi.useFakeTimers()
    const wrapper = mount(SimpleChart, {
      props: { symbol: 'ETHUSDT', interval: '1h', candles: initialCandles }
    })
    await wrapper.find('[aria-label="绘制趋势线"]').trigger('click')
    const create = chartMocks.chart.createOverlay.mock.calls[0]![0] as Record<string, unknown>
    expect(create.name).toBe('segment')
    expect(create.onDrawEnd).toEqual(expect.any(Function))

    const overlay = chartMocks.registered.overlays[0]!
    overlay.points = [
      { timestamp: initialCandles[0]!.timestamp, value: 100 },
      { timestamp: initialCandles[1]!.timestamp, value: 110 }
    ]
    overlay.currentStep = 2
    overlay.totalStep = 3
    ;(create.onDrawEnd as (event: unknown) => void)({ overlay })
    await vi.advanceTimersByTimeAsync(150)

    let stored = JSON.parse(localStorage.getItem(CHART_PREFERENCES_STORAGE_KEY) || '{}')
    expect(stored.symbols.ETHUSDT.drawings).toEqual([expect.objectContaining({
      id: 'drawing-1', name: 'segment', points: [
        { timestamp: initialCandles[0]!.timestamp, value: 100 },
        { timestamp: initialCandles[1]!.timestamp, value: 110 }
      ]
    })])

    overlay.points = [
      { timestamp: initialCandles[0]!.timestamp, value: 105 },
      { timestamp: initialCandles[1]!.timestamp, value: 115 }
    ]
    ;(create.onPressedMoveEnd as (event: unknown) => void)({ overlay })
    await vi.advanceTimersByTimeAsync(150)
    stored = JSON.parse(localStorage.getItem(CHART_PREFERENCES_STORAGE_KEY) || '{}')
    expect(stored.symbols.ETHUSDT.drawings[0].points[0].value).toBe(105)

    await wrapper.find('[aria-label="删除绘图 drawing-1"]').trigger('click')
    await vi.advanceTimersByTimeAsync(150)
    stored = JSON.parse(localStorage.getItem(CHART_PREFERENCES_STORAGE_KEY) || '{}')
    expect(chartMocks.chart.removeOverlay).toHaveBeenCalledWith({ id: 'drawing-1' })
    expect(stored.symbols.ETHUSDT.drawings).toEqual([])
    wrapper.unmount()
  })

  it('restores only the selected symbol drawings and preserves anchors across intervals', async () => {
    const drawing = {
      id: 'eth-segment', name: 'segment',
      points: [
        { timestamp: initialCandles[0]!.timestamp, value: 90 },
        { timestamp: initialCandles[1]!.timestamp, value: 120 }
      ]
    }
    localStorage.setItem(CHART_PREFERENCES_STORAGE_KEY, JSON.stringify({
      version: 2,
      studies: {},
      symbols: {
        ETHUSDT: { drawings: [drawing] },
        BTCUSDT: { drawings: [] }
      }
    }))

    const wrapper = mount(SimpleChart, {
      props: { symbol: 'ETHUSDT', interval: '1h', candles: initialCandles }
    })
    const restoreCall = chartMocks.chart.createOverlay.mock.calls[0]![0] as Record<string, unknown>
    expect(restoreCall).toMatchObject({ id: 'eth-segment', name: 'segment', points: drawing.points })
    expect(wrapper.find('[aria-label="删除绘图 eth-segment"]').exists()).toBe(true)

    chartMocks.chart.createOverlay.mockClear()
    await wrapper.setProps({ interval: '5m' })
    expect(chartMocks.chart.createOverlay).not.toHaveBeenCalled()
    expect(chartMocks.registered.overlays[0]?.points).toEqual(drawing.points)

    await wrapper.setProps({ symbol: 'BTCUSDT' })
    expect(chartMocks.chart.removeOverlay).toHaveBeenCalled()
    expect(chartMocks.chart.createOverlay).not.toHaveBeenCalled()
    expect(wrapper.find('[aria-label="删除绘图 eth-segment"]').exists()).toBe(false)
    wrapper.unmount()
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
