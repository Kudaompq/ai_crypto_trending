<template>
  <section class="chart-card">
    <div ref="chartHost" class="pro-chart-host" role="img" :aria-label="`${symbol} ${interval} K线图`"></div>
    <div v-if="showEmaSettings" class="ema-dialog-backdrop" @click.self="cancelEmaSettings">
      <section class="ema-dialog" role="dialog" aria-modal="true" aria-labelledby="ema-settings-title">
        <header class="ema-dialog-header">
          <div>
            <h2 id="ema-settings-title">EMA 均线设置</h2>
            <p>设置均线周期和颜色，保存后应用到当前图表。</p>
          </div>
          <button type="button" class="ema-dialog-close" aria-label="关闭 EMA 设置" @click="cancelEmaSettings">×</button>
        </header>
        <p v-if="studyError" class="ema-dialog-error" role="alert">{{ studyError }}</p>
        <div class="ema-dialog-lines">
          <div v-for="(period, index) in emaSettingsDraft.params" :key="index" class="ema-line-setting">
            <label>
              <span>EMA {{ index + 1 }} 周期</span>
              <input type="text" inputmode="numeric" autocomplete="off" :value="period"
                :aria-label="`EMA 第 ${index + 1} 条周期`" @input="updateEmaPeriod(index, $event)" />
            </label>
            <label>
              <span>颜色</span>
              <input type="color" :value="emaSettingsDraft.colors[index]" :aria-label="`EMA 颜色第 ${index + 1} 条`"
                @input="updateEmaColor(index, $event)" />
            </label>
            <button type="button" class="ema-line-remove" :aria-label="`移除 EMA 第 ${index + 1} 条`"
              :disabled="emaSettingsDraft.params.length <= 1" @click="removeEmaLine(index)">移除</button>
          </div>
          <button type="button" class="ema-add-line" aria-label="添加一条 EMA 均线"
            :disabled="emaSettingsDraft.params.length >= MAX_EMA_LINES" @click="addEmaLine">＋ 添加均线</button>
        </div>
        <footer class="ema-dialog-actions">
          <button type="button" class="ema-cancel" aria-label="取消 EMA 设置" @click="cancelEmaSettings">取消</button>
          <button type="button" class="ema-save" aria-label="保存 EMA 设置" @click="saveEmaSettings">保存设置</button>
        </footer>
      </section>
    </div>
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
  DEFAULT_EMA_COLORS,
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
const showEmaSettings = ref(false)
const emaSettingsDraft = ref({ params: [] as string[], colors: [] as string[] })
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
let lastSyncedSymbol = props.symbol
let lastSyncedInterval = props.interval
const overlayIds = new Set<string>()
const studies = ref<Record<ChartStudyName, ChartStudyPreference>>(loadStudyPreferences(props.symbol))
const DEFAULT_SUBCHART_HEIGHT = 160
const MIN_SUBCHART_HEIGHT = 120
const SUBCHART_HEADER_HEIGHT = 36
const MIN_SUBCHART_HEADER_HEIGHT = 32
const MAX_EMA_LINES = 6
let activeSubIndicators: string[] | null = null
let hiddenSubIndicators = new Set<string>()
const expandedSubchartHeights = new Map<string, number>()

function normalizeEmaColors(colors: string[] | undefined, count: number): string[] {
  return Array.from({ length: count }, (_, index) => {
    const color = colors?.[index]
    return color && /^#[0-9a-fA-F]{6}$/.test(color) ? color : DEFAULT_EMA_COLORS[index % DEFAULT_EMA_COLORS.length]!
  })
}

function emaLineStyles(chart: Chart, colors: string[]) {
  const defaultLines = chart.getStyles().indicator.lines
  return {
    lines: colors.map((color, index) => ({
      ...defaultLines[index % defaultLines.length]!,
      color
    }))
  }
}

function loadStudyPreferences(symbol: string): Record<ChartStudyName, ChartStudyPreference> {
  const saved = getChartPreferences(symbol).studies
  return Object.fromEntries(Object.entries(DEFAULT_STUDY_PREFERENCES).map(([name, fallback]) => {
    const value = saved[name as ChartStudyName]
    const preference = value ? { enabled: value.enabled, params: [...value.params], colors: value.colors ? [...value.colors] : undefined } : {
      enabled: fallback.enabled, params: [...fallback.params], colors: fallback.colors ? [...fallback.colors] : undefined
    }
    if (name === 'EMA') preference.colors = normalizeEmaColors(preference.colors, preference.params.length)
    return [name, preference]
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

function getSubchartIndicators(chart: Chart): Array<{ paneId: string; name: string; visible: boolean }> {
  const panes = chart.getIndicatorByPaneId()
  if (!(panes instanceof Map)) return []
  return [...panes].flatMap(([paneId, indicators]) => {
    if (paneId === 'candle_pane' || !(indicators instanceof Map)) return []
    return [...indicators].map(([name, indicator]) => ({ paneId, name, visible: indicator.visible }))
  })
}

function rememberSubchartSelection(chart: Chart) {
  const indicators = getSubchartIndicators(chart)
  activeSubIndicators = [...new Set(indicators.map(indicator => indicator.name))]
  hiddenSubIndicators = new Set(indicators.filter(indicator => !indicator.visible).map(indicator => indicator.name))
}

function updateSubchartPaneVisibility(chart: Chart, paneId: string, visible: boolean) {
  const indicators = chart.getIndicatorByPaneId(paneId)
  if (!(indicators instanceof Map)) return
  const paneIndicators = indicators as Map<string, Indicator>
  const allHidden = [...paneIndicators.values()].every(indicator => !indicator.visible)
  if (visible) {
    const height = expandedSubchartHeights.get(paneId) ?? DEFAULT_SUBCHART_HEIGHT
    chart.setPaneOptions({ id: paneId, height, minHeight: MIN_SUBCHART_HEIGHT })
    expandedSubchartHeights.delete(paneId)
  } else if (allHidden) {
    if (!expandedSubchartHeights.has(paneId)) {
      const height = chart.getSize(paneId)?.height
      expandedSubchartHeights.set(paneId, height && height > SUBCHART_HEADER_HEIGHT ? height : DEFAULT_SUBCHART_HEIGHT)
    }
    chart.setPaneOptions({ id: paneId, height: SUBCHART_HEADER_HEIGHT, minHeight: MIN_SUBCHART_HEADER_HEIGHT })
  }
}

function restoreHiddenSubcharts(chart: Chart) {
  for (const { paneId, name } of getSubchartIndicators(chart)) {
    if (hiddenSubIndicators.has(name)) chart.overrideIndicator({ name, visible: false }, paneId)
  }
}

function selectionRequested(selection: { symbol: string; interval: string }) {
  if (selection.symbol !== props.symbol || selection.interval !== props.interval) emit('selectionChange', selection)
}

function setStudyPreference(name: ChartStudyName, next: ChartStudyPreference) {
  const preference = {
    enabled: next.enabled,
    params: [...next.params],
    ...(name === 'EMA' ? { colors: normalizeEmaColors(next.colors, next.params.length) } : {})
  }
  studies.value = { ...studies.value, [name]: preference }
  const preferences = getChartPreferences(activeDrawingSymbol)
  const saved = saveChartPreferences(activeDrawingSymbol, { ...preferences, studies: studies.value })
  chartStorageWarning.value = saved ? '' : '图表设置未能保存到浏览器，刷新后可能丢失'
}

function openEmaSettings() {
  emaSettingsDraft.value = {
    params: studies.value.EMA.params.map(String),
    colors: normalizeEmaColors(studies.value.EMA.colors, studies.value.EMA.params.length)
  }
  studyError.value = ''
  showEmaSettings.value = true
}

function updateEmaPeriod(index: number, event: Event) {
  const period = (event.target as HTMLInputElement).value
  const params = [...emaSettingsDraft.value.params]
  params[index] = period
  emaSettingsDraft.value = { ...emaSettingsDraft.value, params }
}

function updateEmaColor(index: number, event: Event) {
  const colors = [...emaSettingsDraft.value.colors]
  colors[index] = (event.target as HTMLInputElement).value
  emaSettingsDraft.value = { ...emaSettingsDraft.value, colors }
}

function addEmaLine() {
  const params = [...emaSettingsDraft.value.params]
  if (params.length >= MAX_EMA_LINES) return
  const lastPeriod = Number(params[params.length - 1])
  params.push(String(Number.isInteger(lastPeriod) && lastPeriod > 0 ? lastPeriod + 10 : 30))
  const colors = normalizeEmaColors(emaSettingsDraft.value.colors, params.length)
  emaSettingsDraft.value = { params, colors }
}

function removeEmaLine(index: number) {
  const params = [...emaSettingsDraft.value.params]
  if (params.length <= 1) return
  params.splice(index, 1)
  const colors = [...emaSettingsDraft.value.colors]
  colors.splice(index, 1)
  emaSettingsDraft.value = { params, colors }
}

function saveEmaSettings() {
  const { params: periodText } = emaSettingsDraft.value
  if (!periodText.every(period => /^\d+$/.test(period))) {
    studyError.value = 'EMA 周期必须为数字'
    return
  }
  const params = periodText.map(Number)
  if (!validStudyParams('EMA', params)) return
  const current = studies.value.EMA
  const colors = normalizeEmaColors(emaSettingsDraft.value.colors, params.length)
  setStudyPreference('EMA', { enabled: current.enabled, params, colors })
  if (current.enabled && coreChart) {
    coreChart.overrideIndicator({ name: 'EMA', calcParams: [...params], styles: emaLineStyles(coreChart, colors) }, 'candle_pane')
  }
  showEmaSettings.value = false
}

function cancelEmaSettings() {
  showEmaSettings.value = false
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
    let createValue = value
    let createPaneOptions = paneOptions
    if (studyName) {
      const requested = typeof value === 'string' ? studies.value[studyName].params : (value.calcParams ?? studies.value[studyName].params)
      if (!validStudyParams(studyName, requested)) return null
      if (studyName === 'EMA') {
        const styles = emaLineStyles(chart, normalizeEmaColors(studies.value.EMA.colors, requested.length))
        createValue = typeof value === 'string' ? { name: value, calcParams: [...requested], styles } : {
          ...value,
          calcParams: [...requested],
          styles: { ...value.styles, ...styles }
        }
      }
      if ((studyName === 'MACD' || studyName === 'OPEN_INTEREST') && !paneOptions?.height) {
        createPaneOptions = { ...paneOptions, height: DEFAULT_SUBCHART_HEIGHT, minHeight: MIN_SUBCHART_HEIGHT }
      }
    }
    const created = originalCreateIndicator(createValue, isStack, createPaneOptions, callback)
    if (studyName && created) {
      const requested = typeof value === 'string' ? studies.value[studyName].params : (value.calcParams ?? studies.value[studyName].params)
      setStudyPreference(studyName, { ...studies.value[studyName], enabled: true, params: [...requested] })
      if (studyName === 'EMA') openEmaSettings()
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
    if (paneId && paneId !== 'candle_pane' && typeof override.visible === 'boolean') {
      if (name) {
        if (override.visible) hiddenSubIndicators.delete(name)
        else hiddenSubIndicators.add(name)
      }
      updateSubchartPaneVisibility(chart, paneId, override.visible)
    }
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

function syncChartScope(symbol: string, interval: string) {
  if (!proChart || (symbol === lastSyncedSymbol && interval === lastSyncedInterval)) return

  saveDrawingsNow()
  if (coreChart) rememberSubchartSelection(coreChart)
  const symbolChanged = symbol !== lastSyncedSymbol
  lastSyncedSymbol = symbol
  lastSyncedInterval = interval
  if (symbolChanged) {
    activeDrawingSymbol = symbol
    studies.value = loadStudyPreferences(symbol)
  }
  showEmaSettings.value = false
  recreateProChart(false)
}

function setInitialStudyParameters(chart: Chart) {
  for (const name of ['MA', 'EMA', 'BOLL', 'MACD'] as const) {
    if (!studies.value[name].enabled) continue
    if (validStudyParams(name, studies.value[name].params)) {
      chart.overrideIndicator({
        name,
        calcParams: [...studies.value[name].params],
        ...(name === 'EMA' ? { styles: emaLineStyles(chart, normalizeEmaColors(studies.value.EMA.colors, studies.value.EMA.params.length)) } : {})
      })
    }
  }
}

function setDefaultSubchartHeights(chart: Chart) {
  const panes = chart.getIndicatorByPaneId()
  if (!(panes instanceof Map)) return
  for (const [paneId, indicators] of panes) {
    const names = indicators instanceof Map ? [...indicators.keys()] : []
    if (paneId !== 'candle_pane' && names.length > 0) {
      chart.setPaneOptions({ id: paneId, height: DEFAULT_SUBCHART_HEIGHT, minHeight: MIN_SUBCHART_HEIGHT })
    }
  }
}

function createChartDatafeed() {
  return createBinanceProDatafeed({
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
}

function initializeProChart(container: HTMLElement) {
  datafeed = createChartDatafeed()
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
    subIndicators: activeSubIndicators ?? selectedIndicators(['MACD', 'OPEN_INTEREST']),
    drawingBarVisible: true
  })

  coreChart = proChart.getChart()
  if (!coreChart) {
    historyError.value = 'K 线图表未能初始化'
    return
  }
  installStudyPreferenceBridge(coreChart)
  setDefaultSubchartHeights(coreChart)
  installOverlayPreferenceBridge(coreChart)
  setInitialStudyParameters(coreChart)
  restoreHiddenSubcharts(coreChart)
  restoreDrawings(activeDrawingSymbol)
  scheduleOpenInterestRefresh()

  if (typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(() => coreChart?.resize())
    resizeObserver.observe(container)
  }
}

function recreateProChart(saveCurrentDrawings = true) {
  const container = chartHost.value
  if (!container || !proChart) return

  if (saveCurrentDrawings) saveDrawingsNow()
  expandedSubchartHeights.clear()
  if (openInterestTimer !== null) clearTimeout(openInterestTimer)
  openInterestTimer = null
  cancelOpenInterestRequest()
  resizeObserver?.disconnect()
  resizeObserver = null

  proChart.dispose()
  proChart = null
  coreChart = null
  datafeed?.dispose()
  datafeed = null
  historyError.value = ''
  openInterestError.value = ''

  initializeProChart(container)
}

onMounted(() => {
  const container = chartHost.value
  if (!container) return
  activeDrawingSymbol = props.symbol
  lastSyncedSymbol = props.symbol
  lastSyncedInterval = props.interval
  initializeProChart(container)
})

watch(() => [props.symbol, props.interval] as const, ([symbol, interval]) => syncChartScope(symbol, interval))
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
  overflow: hidden;
  background: #141414;
  border: 1px solid #303030;
  border-radius: 8px;
}

.ema-dialog-backdrop {
  position: fixed;
  z-index: 1000;
  inset: 0;
  display: grid;
  place-items: center;
  padding: 20px;
  background: rgb(0 0 0 / 68%);
}

.ema-dialog {
  width: min(100%, 480px);
  max-height: min(88vh, 680px);
  overflow-y: auto;
  padding: 20px;
  border: 1px solid #3a3a3a;
  border-radius: 12px;
  background: #191919;
  box-shadow: 0 20px 70px rgb(0 0 0 / 55%);
  color: #e5e5e5;
  font-size: 13px;
}

.ema-dialog-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}

.ema-dialog-header h2 {
  margin: 0;
  color: #f3f3f3;
  font-size: 17px;
  font-weight: 600;
}

.ema-dialog-header p {
  margin: 6px 0 0;
  color: #8f8f8f;
  font-size: 12px;
}

.ema-dialog-error {
  margin: -4px 0 12px;
  color: #f0a35b;
  font-size: 12px;
}

.ema-dialog-close {
  flex: 0 0 auto;
  width: 28px;
  height: 28px;
  padding: 0;
  border: 0;
  border-radius: 5px;
  background: transparent;
  color: #999;
  font-size: 22px;
  line-height: 1;
  cursor: pointer;
}

.ema-dialog-close:hover {
  background: #292929;
  color: #fff;
}

.ema-dialog-lines {
  display: grid;
  gap: 9px;
}

.ema-line-setting {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) auto;
  align-items: end;
  gap: 12px;
  padding: 11px;
  border: 1px solid #303030;
  border-radius: 8px;
  background: #1d1d1d;
}

.ema-line-setting label {
  display: grid;
  gap: 7px;
  color: #a5a5a5;
  font-size: 11px;
}

.ema-line-setting input[type="text"] {
  box-sizing: border-box;
  width: 100%;
  height: 32px;
  padding: 5px 8px;
  border: 1px solid #414141;
  border-radius: 5px;
  outline: none;
  background: #111;
  color: #eee;
}

.ema-line-setting input[type="text"]:focus {
  border-color: #4388e8;
}

.ema-line-setting input[type="color"] {
  box-sizing: border-box;
  width: 100%;
  height: 32px;
  padding: 3px;
  border: 1px solid #414141;
  border-radius: 5px;
  background: #111;
  cursor: pointer;
}

.ema-line-remove,
.ema-add-line,
.ema-dialog-actions button {
  height: 32px;
  padding: 0 10px;
  border: 1px solid #414141;
  border-radius: 5px;
  background: #252525;
  color: #c7c7c7;
  font-size: 12px;
  cursor: pointer;
}

.ema-line-remove:disabled,
.ema-add-line:disabled {
  opacity: .45;
  cursor: not-allowed;
}

.ema-add-line {
  justify-self: start;
  border-style: dashed;
  background: transparent;
  color: #b9b9b9;
}

.ema-dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 20px;
  padding-top: 14px;
  border-top: 1px solid #303030;
}

.ema-dialog-actions .ema-cancel {
  background: transparent;
}

.ema-dialog-actions .ema-save {
  border-color: #3979d1;
  background: #2864b5;
  color: #fff;
}

.ema-dialog-actions button:hover:not(:disabled) {
  filter: brightness(1.15);
}

@media (max-width: 480px) {
  .ema-dialog {
    padding: 16px;
  }

  .ema-line-setting {
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  }

  .ema-line-remove {
    grid-column: 1 / -1;
    justify-self: end;
  }
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
