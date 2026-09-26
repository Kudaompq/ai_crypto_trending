// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import SimpleChart from '../components/SimpleChart.vue'
import { CHART_PREFERENCES_STORAGE_KEY, getChartPreferences } from '../services/chartPreferences'
import { api } from '../services/api'

const chartMocks = vi.hoisted(() => {
  const overlays = new Map<string, Record<string, any>>()
  const instances: Array<{ options: any; chart: any }> = []
  const coreOriginal = {
    createIndicator: vi.fn((_value?: unknown, _isStack?: boolean) => 'pane-1'),
    overrideIndicator: vi.fn((_value?: unknown, _paneId?: unknown, _callback?: unknown) => undefined),
    removeIndicator: vi.fn((_paneId?: unknown, _name?: unknown) => undefined),
    getDataList: vi.fn(() => []),
    resize: vi.fn(),
    createOverlay: vi.fn((value: Record<string, any>) => {
      const id = value.id || `overlay-${overlays.size + 1}`
      overlays.set(id, { ...value, id, points: value.points || [], totalStep: value.totalStep || 2, visible: value.visible ?? true, lock: value.lock ?? false, mode: value.mode || 'normal' })
      return id
    }),
    getOverlayById: vi.fn((id: string) => overlays.get(id) || null),
    removeOverlay: vi.fn((value?: string | { id?: string }) => {
      if (typeof value === 'string') overlays.delete(value)
      else if (value?.id) overlays.delete(value.id)
      else overlays.clear()
      return true
    }),
    overrideOverlay: vi.fn((value: Record<string, any>) => {
      const overlay = overlays.get(value.id)
      if (overlay) Object.assign(overlay, value)
    })
  }
  const coreChart = { ...coreOriginal }
  const constructors = vi.fn(function (options: any) {
    let symbol = options.symbol
    let period = options.period
    const chart = {
      getSymbol: vi.fn(() => symbol),
      setSymbol: vi.fn((value: any) => { symbol = value }),
      getPeriod: vi.fn(() => period),
      setPeriod: vi.fn((value: any) => { period = value }),
      getChart: vi.fn(() => coreChart),
      dispose: vi.fn()
    }
    instances.push({ options, chart })
    return chart
  })
  return { overlays, instances, coreChart, coreOriginal, constructors, indicator: null as any, resizeObservers: [] as Array<{ observe: ReturnType<typeof vi.fn>; disconnect: ReturnType<typeof vi.fn> }> }
})

vi.mock('@klinecharts/pro', () => ({
  KLineChartPro: chartMocks.constructors,
  loadLocales: vi.fn()
}))

vi.mock('klinecharts', () => ({
  IndicatorSeries: { Normal: 'normal' },
  registerIndicator: vi.fn((indicator: unknown) => { chartMocks.indicator = indicator })
}))

vi.mock('../services/api', () => ({
  api: {
    getKlineData: vi.fn(async () => ({ symbol: 'ETHUSDT', interval: '5m', data: [], has_more_before: true })),
    getOpenInterestData: vi.fn(async () => ({ symbol: 'ETHUSDT', interval: '5m', data: [] })),
    errorMessage: vi.fn((_error: unknown, fallback: string) => fallback)
  }
}))

const candles = [
  { timestamp: 300_000, open: 99, high: 102, low: 98, close: 100, volume: 12 },
  { timestamp: 600_000, open: 100, high: 103, low: 99, close: 101, volume: 18 }
]

function savedStudies(studies: Record<string, { enabled: boolean; params: number[] }>) {
  localStorage.setItem(CHART_PREFERENCES_STORAGE_KEY, JSON.stringify({ version: 2, studies, symbols: {} }))
}

async function flushTimers() {
  await new Promise(resolve => setTimeout(resolve, 0))
  await flushPromises()
}

describe('SimpleChart Pro integration', () => {
  beforeEach(() => {
    localStorage.clear()
    chartMocks.instances.length = 0
    chartMocks.overlays.clear()
    chartMocks.constructors.mockClear()
    Object.assign(chartMocks.coreChart, chartMocks.coreOriginal)
    Object.values(chartMocks.coreOriginal).forEach(mock => mock.mockClear())
    chartMocks.resizeObservers.length = 0
    chartMocks.indicator = null
    vi.mocked(api.getKlineData).mockReset().mockResolvedValue({ symbol: 'ETHUSDT', interval: '5m', data: [], has_more_before: true })
    vi.mocked(api.getOpenInterestData).mockReset().mockResolvedValue({ symbol: 'ETHUSDT', interval: '5m', data: [] })
    vi.mocked(api.errorMessage).mockImplementation((_error, fallback) => fallback)
    vi.stubGlobal('ResizeObserver', class {
      observe = vi.fn()
      disconnect = vi.fn()
      constructor(_callback: ResizeObserverCallback) { chartMocks.resizeObservers.push(this) }
    })
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('loads selected shared studies and applies their saved parameters through Core', async () => {
    savedStudies({
      MA: { enabled: true, params: [7, 14, 28, 56] },
      EMA: { enabled: true, params: [6, 12, 20] },
      BOLL: { enabled: true, params: [14, 2.5] },
      MACD: { enabled: true, params: [8, 21, 5] },
      OPEN_INTEREST: { enabled: true, params: [] }
    })
    vi.mocked(api.getOpenInterestData).mockResolvedValue({
      symbol: 'ETHUSDT', interval: '5m', data: [{ timestamp: 300_000, quantity: 10, value: 999_000 }]
    })

    const wrapper = mount(SimpleChart, { props: { symbol: 'ETHUSDT', interval: '5m', candles } })
    await flushTimers()

    expect(chartMocks.instances[0]?.options.mainIndicators).toEqual(['MA', 'EMA', 'BOLL'])
    expect(chartMocks.instances[0]?.options.subIndicators).toEqual(['MACD', 'OPEN_INTEREST'])
    expect(chartMocks.coreOriginal.overrideIndicator.mock.calls.map(([value]) => value)).toEqual(expect.arrayContaining([
      { name: 'MA', calcParams: [7, 14, 28, 56] },
      { name: 'EMA', calcParams: [6, 12, 20] },
      { name: 'BOLL', calcParams: [14, 2.5] },
      { name: 'MACD', calcParams: [8, 21, 5] }
    ]))
    expect(api.getOpenInterestData).toHaveBeenCalledWith('ETHUSDT', '5m', 500, 300_000, 900_000, expect.any(AbortSignal))
    const indicator = chartMocks.indicator as { calc: (bars: typeof candles, instance: { extendData: unknown }) => unknown[] }
    expect(indicator.calc(candles, {
      extendData: { samples: [{ timestamp: 300_000, quantity: 10, value: 999_000 }], interval: '5m' }
    })).toEqual([{ value: 10 }, { value: null }])
    wrapper.unmount()
  })

  it('persists indicator selections and valid parameter edits made through Core', () => {
    const wrapper = mount(SimpleChart, { props: { symbol: 'ETHUSDT', interval: '5m', candles } })

    chartMocks.coreChart.createIndicator({ name: 'EMA' }, true)
    chartMocks.coreChart.overrideIndicator({ name: 'EMA', calcParams: [4, 8, 16] })

    expect(getChartPreferences('ETHUSDT').studies.EMA).toEqual({ enabled: true, params: [4, 8, 16] })
    wrapper.unmount()
  })

  it('rejects invalid study parameters before applying them to Core', async () => {
    const wrapper = mount(SimpleChart, { props: { symbol: 'ETHUSDT', interval: '5m', candles } })

    const result = chartMocks.coreChart.createIndicator({ name: 'EMA', calcParams: [0, 8, 16] }, true)
    await flushPromises()

    expect(result).toBeNull()
    expect(chartMocks.coreOriginal.createIndicator).not.toHaveBeenCalled()
    expect(wrapper.find('[role="alert"]').text()).toContain('均线周期必须是 1 到 6 个正整数')
    wrapper.unmount()
  })

  it('aborts and ignores OI data when the symbol or period changes', async () => {
    savedStudies({ OPEN_INTEREST: { enabled: true, params: [] } })
    let resolveOld!: (value: { symbol: string; interval: string; data: Array<{ timestamp: number; quantity: number; value: number }> }) => void
    vi.mocked(api.getOpenInterestData).mockImplementationOnce(() => new Promise(resolve => { resolveOld = resolve }))
    const wrapper = mount(SimpleChart, { props: { symbol: 'ETHUSDT', interval: '5m', candles } })
    await flushTimers()
    const oldCall = vi.mocked(api.getOpenInterestData).mock.calls[0]!
    const oldSignal = oldCall[5] as AbortSignal

    await wrapper.setProps({ symbol: 'BTCUSDT', interval: '1m', candles })
    await flushTimers()
    expect(oldSignal.aborted).toBe(true)
    chartMocks.coreOriginal.overrideIndicator.mockClear()
    resolveOld({ symbol: 'ETHUSDT', interval: '5m', data: [{ timestamp: 300_000, quantity: 1, value: 999_000 }] })
    await flushPromises()
    expect(chartMocks.coreOriginal.overrideIndicator).not.toHaveBeenCalledWith(expect.objectContaining({
      extendData: expect.objectContaining({ samples: expect.arrayContaining([expect.objectContaining({ value: 999_000 })]) })
    }))
    wrapper.unmount()
  })

  it('persists drawing creation, edits, visibility, lock, and removal from Pro interactions', async () => {
    vi.useFakeTimers()
    const wrapper = mount(SimpleChart, { props: { symbol: 'ETHUSDT', interval: '5m', candles } })
    const id = chartMocks.coreChart.createOverlay({
      name: 'segment', totalStep: 3,
      points: [{ timestamp: 300_000, value: 100 }, { timestamp: 600_000, value: 110 }],
      styles: { line: { color: '#f00' } }
    })
    const create = chartMocks.coreOriginal.createOverlay.mock.calls[0]![0] as Record<string, any>
    const overlay = chartMocks.overlays.get(id)!
    create.onDrawEnd({ overlay })
    await vi.advanceTimersByTimeAsync(130)
    expect(getChartPreferences('ETHUSDT').drawings).toEqual([expect.objectContaining({ id, name: 'segment', points: overlay.points })])

    overlay.points = [{ timestamp: 300_000, value: 105 }, { timestamp: 600_000, value: 115 }]
    create.onPressedMoveEnd({ overlay })
    chartMocks.coreChart.overrideOverlay({ id, visible: false, lock: true, mode: 'weak_magnet' })
    await vi.advanceTimersByTimeAsync(130)
    expect(getChartPreferences('ETHUSDT').drawings[0]).toMatchObject({
      points: [{ timestamp: 300_000, value: 105 }, { timestamp: 600_000, value: 115 }],
      visible: false, lock: true, mode: 'weak_magnet'
    })

    chartMocks.coreChart.removeOverlay({ id })
    await vi.advanceTimersByTimeAsync(130)
    expect(getChartPreferences('ETHUSDT').drawings).toEqual([])
    wrapper.unmount()
  })

  it('restores drawings only for the selected symbol and keeps their anchors across period changes', async () => {
    localStorage.setItem(CHART_PREFERENCES_STORAGE_KEY, JSON.stringify({
      version: 2, studies: {},
      symbols: {
        ETHUSDT: { drawings: [{ id: 'eth-line', name: 'segment', points: [{ timestamp: 300_000, value: 90 }, { timestamp: 600_000, value: 120 }] }] },
        BTCUSDT: { drawings: [] }
      }
    }))
    const wrapper = mount(SimpleChart, { props: { symbol: 'ETHUSDT', interval: '5m', candles } })

    expect(chartMocks.coreOriginal.createOverlay).toHaveBeenCalledWith(
      expect.objectContaining({ id: 'eth-line', name: 'segment' }), undefined
    )
    const savedOverlay = chartMocks.overlays.get('eth-line')!
    await wrapper.setProps({ interval: '15m' })
    expect(savedOverlay.points).toEqual([{ timestamp: 300_000, value: 90 }, { timestamp: 600_000, value: 120 }])

    chartMocks.coreOriginal.createOverlay.mockClear()
    await wrapper.setProps({ symbol: 'BTCUSDT' })
    expect(chartMocks.coreOriginal.removeOverlay).toHaveBeenCalledWith({ id: 'eth-line' })
    expect(chartMocks.coreOriginal.createOverlay).not.toHaveBeenCalled()
    expect(getChartPreferences('ETHUSDT').drawings).toHaveLength(1)
    expect(getChartPreferences('BTCUSDT').drawings).toEqual([])
    wrapper.unmount()
  })

  it('shows retryable history errors and publishes store candle updates to the current Datafeed subscription', async () => {
    vi.mocked(api.getKlineData).mockRejectedValueOnce(new Error('temporary history failure'))
    vi.mocked(api.errorMessage).mockImplementation((error, fallback) => (error as Error).message || fallback)
    const wrapper = mount(SimpleChart, { props: { symbol: 'ETHUSDT', interval: '5m', candles } })
    const feed = chartMocks.instances[0]!.options.datafeed
    const symbol = { ticker: 'ETHUSDT' }
    const period = { text: '5m', multiplier: 5, timespan: 'minute' }
    await expect(feed.getHistoryKLineData(symbol, period, 0, 900_000)).resolves.toEqual([])
    expect(wrapper.find('[role="alert"]').text()).toContain('temporary history failure')

    await wrapper.find('.chart-retry').trigger('click')
    vi.mocked(api.getKlineData).mockResolvedValueOnce({ symbol: 'ETHUSDT', interval: '5m', data: candles, has_more_before: false })
    await feed.getHistoryKLineData(symbol, period, 0, 900_000)
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)

    const callback = vi.fn()
    feed.subscribe({ ticker: 'ETHUSDT' }, period, callback)
    const updated = { ...candles[1]!, close: 105 }
    await wrapper.setProps({ candles: [candles[0]!, updated] })
    expect(callback).toHaveBeenCalledWith(updated)
    wrapper.unmount()
  })

  it('disposes Pro, Datafeed, and ResizeObserver on unmount', () => {
    const wrapper = mount(SimpleChart, { props: { symbol: 'ETHUSDT', interval: '5m', candles } })
    const feed = chartMocks.instances[0]!.options.datafeed
    const disposeFeed = vi.spyOn(feed, 'dispose')

    wrapper.unmount()

    expect(chartMocks.instances[0]?.chart.dispose).toHaveBeenCalledOnce()
    expect(disposeFeed).toHaveBeenCalledOnce()
    expect(chartMocks.resizeObservers[0]?.disconnect).toHaveBeenCalledOnce()
  })
})
