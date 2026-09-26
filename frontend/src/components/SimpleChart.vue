<template>
  <section class="chart-card">
    <div ref="chartHost" class="pro-chart-host" role="img" :aria-label="`${symbol} ${interval} K线图`"></div>
    <p v-if="historyError" class="chart-message" role="alert">
      {{ historyError }}
      <button class="chart-retry" type="button" @click="retryHistory">重试历史行情</button>
    </p>
    <p v-if="openInterestError" class="chart-message" role="alert">
      {{ openInterestError }}
      <button class="chart-retry" type="button" @click="refreshOpenInterest">重试 OI</button>
    </p>
    <p v-if="studyError" class="chart-message" role="alert">{{ studyError }}</p>
    <p v-if="chartStorageWarning" class="chart-message" role="alert">{{ chartStorageWarning }}</p>
  </section>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { KLineChartPro, loadLocales } from '@klinecharts/pro'
import type { SymbolInfo } from '@klinecharts/pro'
import {
  IndicatorSeries,
  registerIndicator,
  type Chart,
  type Indicator,
  type IndicatorCreate,
  type KLineData,
  type OverlayCreate,
  type OverlayRemove
} from 'klinecharts'
import '@klinecharts/pro/dist/klinecharts-pro.css'
import { api, type Candle, type OpenInterestSample } from '../services/api'
import { createBinanceProDatafeed, BINANCE_PRO_PERIODS, type BinanceProPeriod } from '../services/binanceProDatafeed'
import { mapOpenInterestToBars, openInterestIntervalMilliseconds } from '../services/openInterestSeries'
import {
  DEFAULT_STUDY_PREFERENCES,
  getChartPreferences,
  saveChartPreferences,
  validateChartStudyParameters,
  type ChartStudyName,
  type ChartStudyPreference,
  type SavedChartDrawing
} from '../services/chartPreferences'
import { persistProDrawings, restoreProOverlays, type CoreOverlaySnapshot } from '../services/proDrawingPreferences'

interface OpenInterestIndicatorResult {
  value: number | null
}

interface OpenInterestIndicatorExtension {
  samples: OpenInterestSample[]
  interval: string
}

registerIndicator<OpenInterestIndicatorResult>({
  name: 'OPEN_INTEREST',
  shortName: 'OI',
  series: IndicatorSeries.Normal,
  precision: 2,
  calcParams: [],
  shouldOhlc: false,
  shouldFormatBigNumber: true,
  visible: true,
  zLevel: 0,
  extendData: { samples: [], interval: '' } satisfies OpenInterestIndicatorExtension,
  figures: [{ key: 'value', title: 'OI Quantity', type: 'line' }],
  minValue: null,
  maxValue: null,
  styles: null,
  regenerateFigures: null,
  createTooltipDataSource: null,
  draw: null,
  calc: (dataList: KLineData[], indicator: Indicator<OpenInterestIndicatorResult>) => mapOpenInterestToBars(
    dataList,
    (indicator.extendData?.samples ?? []) as OpenInterestSample[],
    indicator.extendData?.interval ?? ''
  )
})

loadLocales('zh-CN', { open_interest: 'OI（未平仓持仓量）' })
loadLocales('en-US', { open_interest: 'Open Interest' })

const props = withDefaults(defineProps<{
  candles: Candle[]
  symbol?: string
  interval?: string
  hasMoreBefore?: boolean
  symbols?: string[]
}>(), {
  symbol: 'ETHUSDT',
  interval: '1d',
  hasMoreBefore: true,
  symbols: () => []
})

const emit = defineEmits<{
  selectionChange: [selection: { symbol: string; interval: string }]
}>()

const chartHost = ref<HTMLElement | null>(null)
const chartStorageWarning = ref('')
const historyError = ref('')
const openInterestError = ref('')
const studyError = ref('')
let proChart: KLineChartPro | null = null
let coreChart: Chart | null = null
let datafeed: ReturnType<typeof createBinanceProDatafeed> | null = null
let resizeObserver: ResizeObserver | null = null
let openInterestController: AbortController | null = null
let openInterestVersion = 0
let openInterestTimer: ReturnType<typeof setTimeout> | null = null
let drawingPersistTimer: ReturnType<typeof setTimeout> | null = null
let activeDrawingSymbol = props.symbol
const overlayIds = new Set<string>()
const studies = ref<Record<ChartStudyName, ChartStudyPreference>>(loadStudyPreferences(props.symbol))

function loadStudyPreferences(symbol: string): Record<ChartStudyName, ChartStudyPreference> {
  const saved = getChartPreferences(symbol).studies
  return Object.fromEntries(Object.entries(DEFAULT_STUDY_PREFERENCES).map(([name, fallback]) => {
    const value = saved[name as ChartStudyName]
    return [name, value ? { enabled: value.enabled, params: [...value.params] } : { enabled: fallback.enabled, params: [...fallback.params] }]
  })) as Record<ChartStudyName, ChartStudyPreference>
}

function symbolInfo(ticker: string): SymbolInfo {
  return {
    ticker,
    name: ticker,
    shortName: ticker.replace(/USDT$/, '/USDT'),
    exchange: 'Binance',
    market: 'futures',
    pricePrecision: 8,
    volumePrecision: 2
  }
}

function periodInfo(interval: string): BinanceProPeriod {
  const period = BINANCE_PRO_PERIODS.find(item => item.text === interval)
  if (!period) throw new Error(`Unsupported Binance interval: ${interval}`)
  return period
}

function selectedIndicators(names: ChartStudyName[]): string[] {
  return names.filter(name => studies.value[name].enabled)
}

function selectionRequested(selection: { symbol: string; interval: string }) {
  if (selection.symbol !== props.symbol || selection.interval !== props.interval) emit('selectionChange', selection)
}

function setStudyPreference(name: ChartStudyName, next: ChartStudyPreference) {
  studies.value = { ...studies.value, [name]: { enabled: next.enabled, params: [...next.params] } }
  const preferences = getChartPreferences(activeDrawingSymbol)
  const saved = saveChartPreferences(activeDrawingSymbol, { ...preferences, studies: studies.value })
  chartStorageWarning.value = saved ? '' : '图表设置未能保存到浏览器，刷新后可能丢失'
}

function validStudyParams(name: ChartStudyName, params: number[]): boolean {
  const validation = validateChartStudyParameters(name, params)
  if (!validation.valid) {
    studyError.value = validation.error || '指标参数无效'
    return false
  }
  studyError.value = ''
  return true
}

function indicatorName(value: string | IndicatorCreate): string {
  return typeof value === 'string' ? value : value.name
}

function chartStudyName(name: string): ChartStudyName | null {
  return name in studies.value ? name as ChartStudyName : null
}

function installStudyPreferenceBridge(chart: Chart) {
  const originalCreateIndicator = chart.createIndicator.bind(chart)
  chart.createIndicator = ((value, isStack, paneOptions, callback) => {
    const name = indicatorName(value)
    const studyName = chartStudyName(name)
    if (studyName) {
      const requested = typeof value === 'string' ? studies.value[studyName].params : (value.calcParams ?? studies.value[studyName].params)
      if (!validStudyParams(studyName, requested)) return null
    }
    const created = originalCreateIndicator(value, isStack, paneOptions, callback)
    if (studyName && created) {
      const requested = typeof value === 'string' ? studies.value[studyName].params : (value.calcParams ?? studies.value[studyName].params)
      setStudyPreference(studyName, { enabled: true, params: [...requested] })
      if (studyName === 'OPEN_INTEREST') scheduleOpenInterestRefresh()
    }
    return created
  }) as Chart['createIndicator']

  const originalRemoveIndicator = chart.removeIndicator.bind(chart)
  chart.removeIndicator = ((paneId, name) => {
    originalRemoveIndicator(paneId, name)
    const removed = name ? [chartStudyName(name)].filter((item): item is ChartStudyName => item !== null) : Object.keys(studies.value) as ChartStudyName[]
    for (const studyName of removed) setStudyPreference(studyName, { ...studies.value[studyName], enabled: false })
    if (removed.includes('OPEN_INTEREST')) cancelOpenInterestRequest()
  }) as Chart['removeIndicator']

  const originalOverrideIndicator = chart.overrideIndicator.bind(chart)
  chart.overrideIndicator = ((override, paneId, callback) => {
    const name = chartStudyName(override.name)
    if (name && override.calcParams) {
      const params = override.calcParams as number[]
      if (!validStudyParams(name, params)) return
    }
    originalOverrideIndicator(override, paneId, callback)
    if (name && override.calcParams) setStudyPreference(name, { ...studies.value[name], params: [...override.calcParams as number[]] })
  }) as Chart['overrideIndicator']
}

function completePoints(points: Array<Partial<{ timestamp: number; value: number }>>): Array<{ timestamp: number; value: number }> {
  return points.flatMap(point => Number.isFinite(point.timestamp) && Number.isFinite(point.value)
    ? [{ timestamp: point.timestamp!, value: point.value! }]
    : [])
}

function currentOverlays(): CoreOverlaySnapshot[] {
  if (!coreChart) return []
  return [...overlayIds].flatMap(id => {
    const overlay = coreChart?.getOverlayById(id)
    if (!overlay) return []
    return [{
      id: overlay.id,
      name: overlay.name,
      points: completePoints(overlay.points),
      totalStep: overlay.totalStep,
      styles: overlay.styles as Record<string, unknown> | undefined,
      visible: overlay.visible,
      lock: overlay.lock,
      mode: overlay.mode as SavedChartDrawing['mode']
    }]
  })
}

function saveDrawingsNow() {
  if (drawingPersistTimer !== null) clearTimeout(drawingPersistTimer)
  drawingPersistTimer = null
  const stored = persistProDrawings(activeDrawingSymbol, currentOverlays())
  chartStorageWarning.value = stored ? '' : '图表设置未能保存到浏览器，刷新后可能丢失'
}

function scheduleDrawingsSave() {
  if (drawingPersistTimer !== null) clearTimeout(drawingPersistTimer)
  drawingPersistTimer = setTimeout(saveDrawingsNow, 120)
}

function trackOverlayCreate(
  originalCreateOverlay: Chart['createOverlay'],
  value: string | OverlayCreate | Array<string | OverlayCreate>,
  paneId?: string
) {
  const wrapOne = (item: string | OverlayCreate): string | OverlayCreate => {
    if (typeof item === 'string') return item
    const onDrawEnd = item.onDrawEnd
    const onPressedMoveEnd = item.onPressedMoveEnd
    const onRemoved = item.onRemoved
    return {
      ...item,
      onDrawEnd: event => {
        const result = onDrawEnd?.(event) ?? true
        scheduleDrawingsSave()
        return result
      },
      onPressedMoveEnd: event => {
        const result = onPressedMoveEnd?.(event) ?? true
        scheduleDrawingsSave()
        return result
      },
      onRemoved: event => {
        const result = onRemoved?.(event) ?? true
        overlayIds.delete(event.overlay.id)
        scheduleDrawingsSave()
        return result
      }
    }
  }
  const wrapped = Array.isArray(value) ? value.map(wrapOne) : wrapOne(value)
  const result = originalCreateOverlay(wrapped, paneId)
  if (typeof result === 'string') overlayIds.add(result)
  else if (Array.isArray(result)) result.forEach(id => { if (id) overlayIds.add(id) })
  scheduleDrawingsSave()
  return result
}

function installOverlayPreferenceBridge(chart: Chart) {
  const originalCreateOverlay = chart.createOverlay.bind(chart)
  chart.createOverlay = ((value, paneId) => {
    return trackOverlayCreate(originalCreateOverlay, value, paneId)
  }) as Chart['createOverlay']

  const originalRemoveOverlay = chart.removeOverlay.bind(chart)
  chart.removeOverlay = ((remove?: string | OverlayRemove) => {
    originalRemoveOverlay(remove)
    for (const id of [...overlayIds]) if (!chart.getOverlayById(id)) overlayIds.delete(id)
    scheduleDrawingsSave()
  }) as Chart['removeOverlay']

  const originalOverrideOverlay = chart.overrideOverlay.bind(chart)
  chart.overrideOverlay = ((override) => {
    originalOverrideOverlay(override)
    scheduleDrawingsSave()
  }) as Chart['overrideOverlay']
}

function restoreDrawings(symbol: string) {
  if (!coreChart) return
  for (const id of [...overlayIds]) coreChart.removeOverlay({ id })
  overlayIds.clear()
  const drawings = getChartPreferences(symbol).drawings
  for (const overlay of restoreProOverlays(drawings)) {
    coreChart.createOverlay(overlay as OverlayCreate)
  }
}

function cancelOpenInterestRequest() {
  openInterestVersion++
  openInterestController?.abort()
  openInterestController = null
}

function scheduleOpenInterestRefresh() {
  if (openInterestTimer !== null) clearTimeout(openInterestTimer)
  openInterestTimer = setTimeout(() => {
    openInterestTimer = null
    void refreshOpenInterest()
  }, 0)
}

async function refreshOpenInterest() {
  cancelOpenInterestRequest()
  openInterestError.value = ''
  if (!coreChart || !studies.value.OPEN_INTEREST.enabled) return

  const interval = props.interval
  const intervalMs = openInterestIntervalMilliseconds(interval)
  if (intervalMs === null) {
    coreChart.overrideIndicator({ name: 'OPEN_INTEREST', extendData: { samples: [], interval } })
    return
  }
  const candles = coreChart.getDataList().length ? coreChart.getDataList() : props.candles
  if (candles.length === 0) {
    coreChart.overrideIndicator({ name: 'OPEN_INTEREST', extendData: { samples: [], interval } })
    return
  }

  const symbol = props.symbol
  const version = openInterestVersion
  const controller = new AbortController()
  openInterestController = controller
  try {
    const result = await api.getOpenInterestData(
      symbol,
      interval,
      500,
      candles[0]!.timestamp,
      candles[candles.length - 1]!.timestamp + intervalMs,
      controller.signal
    )
    if (controller.signal.aborted || version !== openInterestVersion || symbol !== props.symbol || interval !== props.interval) return
    coreChart?.overrideIndicator({ name: 'OPEN_INTEREST', extendData: { samples: result.data, interval } })
  } catch (error: unknown) {
    if (controller.signal.aborted || version !== openInterestVersion || isAbortError(error)) return
    openInterestError.value = `OI 加载失败：${api.errorMessage(error, '请求失败')}，可重试`
    coreChart?.overrideIndicator({ name: 'OPEN_INTEREST', extendData: { samples: [], interval } })
  } finally {
    if (openInterestController === controller) openInterestController = null
  }
}

function isAbortError(error: unknown): boolean {
  return error instanceof DOMException && error.name === 'AbortError' ||
    typeof error === 'object' && error !== null && 'name' in error && (error as { name?: unknown }).name === 'CanceledError'
}

function retryHistory() {
  if (!proChart) return
  historyError.value = ''
  proChart.setPeriod({ ...proChart.getPeriod() })
}

function syncSymbol(symbol: string) {
  if (!proChart || proChart.getSymbol().ticker === symbol) return
  saveDrawingsNow()
  activeDrawingSymbol = symbol
  studies.value = loadStudyPreferences(symbol)
  proChart.setSymbol(symbolInfo(symbol))
  restoreDrawings(symbol)
  scheduleOpenInterestRefresh()
}

function syncPeriod(interval: string) {
  if (!proChart || proChart.getPeriod().text === interval) return
  cancelOpenInterestRequest()
  proChart.setPeriod(periodInfo(interval))
  scheduleOpenInterestRefresh()
}

function setInitialStudyParameters(chart: Chart) {
  for (const name of ['MA', 'EMA', 'BOLL', 'MACD'] as const) {
    if (!studies.value[name].enabled) continue
    if (validStudyParams(name, studies.value[name].params)) {
      chart.overrideIndicator({ name, calcParams: [...studies.value[name].params] })
    }
  }
}

onMounted(() => {
  const container = chartHost.value
  if (!container) return

  datafeed = createBinanceProDatafeed({
    getSupportedSymbols: () => props.symbols.length > 0 ? props.symbols : [props.symbol],
    fetchHistory: (request, signal) => api.getKlineData(request.symbol, request.interval, request.limit, request.endTime, signal),
    onSelectionRequest: selectionRequested,
    onHistoryError: ({ symbol, interval }, error) => {
      if (symbol === props.symbol && interval === props.interval) historyError.value = `${symbol} ${interval} 历史K线加载失败：${api.errorMessage(error, '请求失败')}`
    },
    onHistoryLoaded: ({ symbol, interval }) => {
      if (symbol === props.symbol && interval === props.interval && studies.value.OPEN_INTEREST.enabled) scheduleOpenInterestRefresh()
    }
  })

  proChart = new KLineChartPro({
    container,
    symbol: symbolInfo(props.symbol),
    period: periodInfo(props.interval),
    periods: BINANCE_PRO_PERIODS,
    datafeed,
    locale: 'zh-CN',
    theme: 'dark',
    timezone: 'Asia/Shanghai',
    watermark: '',
    mainIndicators: selectedIndicators(['MA', 'EMA', 'BOLL']),
    subIndicators: selectedIndicators(['MACD', 'OPEN_INTEREST']),
    drawingBarVisible: true
  })

  coreChart = proChart.getChart()
  if (!coreChart) {
    historyError.value = 'K 线图表未能初始化'
    return
  }
  installStudyPreferenceBridge(coreChart)
  installOverlayPreferenceBridge(coreChart)
  setInitialStudyParameters(coreChart)
  activeDrawingSymbol = props.symbol
  restoreDrawings(props.symbol)
  scheduleOpenInterestRefresh()

  if (typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(() => coreChart?.resize())
    resizeObserver.observe(container)
  }
})

watch(() => props.symbol, symbol => syncSymbol(symbol))
watch(() => props.interval, interval => syncPeriod(interval))
watch(() => {
  const latest = props.candles[props.candles.length - 1]
  return [props.symbol, props.interval, latest?.timestamp, latest?.open, latest?.high, latest?.low, latest?.close, latest?.volume]
}, () => {
  const candle = props.candles[props.candles.length - 1]
  if (candle) datafeed?.publishCandle(props.symbol, props.interval, candle)
}, { flush: 'post' })

onBeforeUnmount(() => {
  saveDrawingsNow()
  if (drawingPersistTimer !== null) clearTimeout(drawingPersistTimer)
  drawingPersistTimer = null
  if (openInterestTimer !== null) clearTimeout(openInterestTimer)
  openInterestTimer = null
  cancelOpenInterestRequest()
  resizeObserver?.disconnect()
  resizeObserver = null
  datafeed?.dispose()
  datafeed = null
  proChart?.dispose()
  proChart = null
  coreChart = null
})
</script>

<style scoped>
.chart-card {
  min-width: 0;
  padding: 12px;
  overflow: hidden;
  background: #141414;
  border: 1px solid #303030;
  border-radius: 8px;
}

.pro-chart-host {
  width: 100%;
  height: clamp(460px, 68vh, 850px);
  min-height: 460px;
}

.chart-message {
  margin: 8px 2px 0;
  color: #e8a44a;
  font-size: 13px;
}

.chart-retry {
  margin-left: 8px;
  padding: 2px 8px;
  color: #d9e6ff;
  background: #292f3a;
  border: 1px solid #4b5870;
  border-radius: 4px;
  cursor: pointer;
}
</style>
