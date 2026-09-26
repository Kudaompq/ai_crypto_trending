<template>
  <section class="chart-card">
    <div class="chart-header">
      <div>
        <h3>{{ symbolLabel }} K线图</h3>
        <span class="chart-interval">{{ interval }}</span>
      </div>
      <div v-if="candles.length" class="current-price">
        <span class="label">当前价格:</span>
        <strong class="price" :class="priceChange >= 0 ? 'positive' : 'negative'">${{ currentPrice.toFixed(2) }}</strong>
        <span class="change" :class="priceChange >= 0 ? 'positive' : 'negative'">
          {{ priceChange >= 0 ? '+' : '' }}{{ priceChange.toFixed(2) }}%
        </span>
      </div>
    </div>

    <div class="chart-summary">
      <div class="info-item"><span class="label">最高:</span><strong>${{ highPrice.toFixed(2) }}</strong></div>
      <div class="info-item"><span class="label">最低:</span><strong>${{ lowPrice.toFixed(2) }}</strong></div>
      <div class="info-item"><span class="label">成交量:</span><strong>{{ totalVolume.toFixed(0) }}</strong></div>
      <div v-if="atr" class="info-item"><span class="label">ATR({{ atr.period }}):</span><strong>${{ atr.value.toFixed(2) }}</strong></div>
    </div>

    <div class="chart-toolbar">
      <button class="chart-tool-button" type="button" aria-label="指标设置"
        :aria-expanded="showStudySettings" @click="showStudySettings = !showStudySettings">
        指标设置
      </button>
      <div class="drawing-tools" role="group" aria-label="绘图工具">
        <button v-for="tool in drawingTools" :key="tool.name" class="chart-tool-button" type="button"
          :aria-label="`绘制${tool.label}`" :aria-pressed="activeDrawingTool === tool.name"
          @click="startDrawing(tool.name)">
          {{ tool.label }}
        </button>
      </div>
      <section v-if="showStudySettings" class="study-settings" aria-label="图表指标设置">
        <div v-for="study in studyOptions" :key="study.name" class="study-setting">
          <label class="study-toggle">
            <input type="checkbox" :aria-label="study.label" :checked="studyPreference(study.name).enabled"
              @change="onStudyToggle(study.name, $event)" />
            <strong>{{ study.label }}</strong>
          </label>
          <div class="study-parameters">
            <label v-for="(parameter, index) in study.parameters" :key="parameter" class="study-parameter">
              <span>{{ parameter }}</span>
              <input type="number" :aria-label="`${study.label} ${parameter}`"
                :step="study.name === 'BOLL' && index === 1 ? 0.1 : 1"
                :value="studyPreference(study.name).params[index]"
                @change="updateStudyParameter(study.name, index, $event)" />
            </label>
          </div>
        </div>
        <p v-if="studyError" class="study-error" role="alert">{{ studyError }}</p>
      </section>
      <div v-if="drawings.length" class="saved-drawings" aria-label="已保存绘图">
        <span v-for="drawing in drawings" :key="drawing.id" class="saved-drawing">
          {{ drawingLabel(drawing.name) }}
          <button type="button" :aria-label="`删除绘图 ${drawing.id}`" @click="removeDrawing(drawing.id)">×</button>
        </span>
      </div>
    </div>

    <div ref="chartHost" class="kline-chart" role="img" :aria-label="`${symbolLabel} ${interval} K线图`"></div>
    <p v-if="openInterestError" class="history-error" role="alert">
      {{ openInterestError }}
      <button type="button" class="chart-tool-button" @click="refreshOpenInterest">重试 OI</button>
    </p>
    <p v-if="historyError" class="history-error" role="alert">{{ historyError }}</p>
    <p v-if="chartStorageWarning" class="history-error" role="alert">{{ chartStorageWarning }}</p>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { dispose, init, registerIndicator, type Chart, type DataLoader, type KLineData } from 'klinecharts'
import { api, type ATRIndicator, type Candle, type OpenInterestSample } from '../services/api'
import { KlineHistory, toKlineChartPeriod } from '../services/klineHistory'
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

const studyOptions: Array<{ name: ChartStudyName; label: string; parameters: string[] }> = [
  { name: 'MA', label: 'MA', parameters: ['周期 1', '周期 2', '周期 3', '周期 4'] },
  { name: 'EMA', label: 'EMA', parameters: ['周期 1', '周期 2', '周期 3'] },
  { name: 'BOLL', label: '布林带', parameters: ['周期', '标准差'] },
  { name: 'MACD', label: 'MACD', parameters: ['快线', '慢线', '信号线'] },
  { name: 'OPEN_INTEREST', label: 'OI', parameters: [] }
]
const drawingTools = [
  { name: 'horizontalStraightLine', label: '水平线' },
  { name: 'segment', label: '趋势线' },
  { name: 'rayLine', label: '射线' },
  { name: 'parallelStraightLine', label: '平行通道' },
  { name: 'fibonacciLine', label: '斐波那契' }
] as const
type DrawingToolName = typeof drawingTools[number]['name']

interface OpenInterestIndicatorExtension {
  samples: OpenInterestSample[]
  interval: string
}

registerIndicator<{ value: number | null }, number, OpenInterestIndicatorExtension>({
  name: 'OPEN_INTEREST',
  shortName: 'OI',
  series: 'normal',
  precision: 2,
  calcParams: [],
  shouldOhlc: false,
  shouldFormatBigNumber: true,
  visible: true,
  zLevel: 0,
  extendData: { samples: [], interval: '' },
  figures: [{ key: 'value', title: 'OI Value', type: 'line' }],
  minValue: null,
  maxValue: null,
  styles: null,
  shouldUpdate: (previous, current) => previous.extendData.interval !== current.extendData.interval ||
    previous.extendData.samples !== current.extendData.samples,
  regenerateFigures: null,
  createTooltipDataSource: null,
  draw: null,
  calc: (dataList, indicator) => mapOpenInterestToBars(
    dataList,
    indicator.extendData?.samples ?? [],
    indicator.extendData?.interval ?? ''
  )
})

const props = withDefaults(defineProps<{
  candles: Candle[]
  symbol?: string
  interval?: string
  hasMoreBefore?: boolean
  atr?: ATRIndicator
}>(), {
  symbol: 'ETHUSDT',
  interval: '1d',
  hasMoreBefore: true
})

const chartHost = ref<HTMLElement | null>(null)
const historyError = ref('')
const chartStorageWarning = ref('')
const studyError = ref('')
const openInterestError = ref('')
const showStudySettings = ref(false)
const studies = ref<Record<ChartStudyName, ChartStudyPreference>>(loadStudyPreferences(props.symbol))
const drawings = ref<SavedChartDrawing[]>(getChartPreferences(props.symbol).drawings)
const activeDrawingTool = ref<DrawingToolName | null>(null)
const symbolLabel = computed(() => props.symbol.replace('USDT', '/USDT'))
const latestCandle = computed(() => props.candles[props.candles.length - 1])
const previousCandle = computed(() => props.candles[props.candles.length - 2])
const currentPrice = computed(() => latestCandle.value?.close ?? 0)
const priceChange = computed(() => {
  const previous = previousCandle.value?.close ?? 0
  return previous ? ((currentPrice.value - previous) / previous) * 100 : 0
})
const highPrice = computed(() => props.candles.length ? Math.max(...props.candles.map(candle => candle.high)) : 0)
const lowPrice = computed(() => props.candles.length ? Math.min(...props.candles.map(candle => candle.low)) : 0)
const totalVolume = computed(() => props.candles.reduce((sum, candle) => sum + candle.volume, 0))

const history = new KlineHistory((request, signal) =>
  api.getKlineData(request.symbol, request.interval, request.limit, request.endTime, signal)
)
let chart: Chart | null = null
let resizeObserver: ResizeObserver | null = null
let liveBarCallback: ((bar: KLineData) => void) | null = null
let lastPublishedCandle: Candle | undefined
let openInterestController: AbortController | null = null
let openInterestRequestVersion = 0
let openInterestRefreshTimer: ReturnType<typeof setInterval> | null = null
let drawingPersistTimer: ReturnType<typeof setTimeout> | null = null
let activeChartSymbol = props.symbol

function loadStudyPreferences(symbol: string): Record<ChartStudyName, ChartStudyPreference> {
  const saved = getChartPreferences(symbol).studies
  return Object.fromEntries(Object.entries(DEFAULT_STUDY_PREFERENCES).map(([name, fallback]) => {
    const value = saved[name as ChartStudyName]
    return [name, value ? { enabled: value.enabled, params: [...value.params] } : { enabled: fallback.enabled, params: [...fallback.params] }]
  })) as Record<ChartStudyName, ChartStudyPreference>
}

function studyPreference(name: ChartStudyName): ChartStudyPreference {
  return studies.value[name]
}

function persistStudies() {
  const preferences = getChartPreferences(props.symbol)
  const saved = saveChartPreferences(props.symbol, { ...preferences, studies: studies.value })
  chartStorageWarning.value = saved ? '' : '图表设置未能保存到浏览器，刷新后可能丢失'
}

function cancelOpenInterestRequest() {
  openInterestRequestVersion++
  openInterestController?.abort()
  openInterestController = null
}

function drawingLabel(name: string): string {
  return drawingTools.find(tool => tool.name === name)?.label ?? name
}

function completeOverlayPoints(points: Array<Partial<{ timestamp: number; value: number }>>): SavedChartDrawing['points'] {
  return points.flatMap(point => Number.isFinite(point.timestamp) && Number.isFinite(point.value)
    ? [{ timestamp: point.timestamp!, value: point.value! }]
    : [])
}

function saveDrawingsNow() {
  if (drawingPersistTimer !== null) clearTimeout(drawingPersistTimer)
  drawingPersistTimer = null
  if (!chart) return

  const saved = chart.getOverlays()
    .filter(overlay => drawingTools.some(tool => tool.name === overlay.name))
    .flatMap(overlay => {
      const points = completeOverlayPoints(overlay.points)
      const requiredPoints = overlay.totalStep > 0 ? overlay.totalStep - 1 : 1
      if (points.length < requiredPoints) return []
      return [{
        id: overlay.id,
        name: overlay.name,
        points,
        ...(overlay.styles && typeof overlay.styles === 'object' ? { styles: overlay.styles as Record<string, unknown> } : {})
      } satisfies SavedChartDrawing]
    })
  drawings.value = saved
  const preferences = getChartPreferences(activeChartSymbol)
  const stored = saveChartPreferences(activeChartSymbol, { ...preferences, drawings: saved })
  chartStorageWarning.value = stored ? '' : '图表设置未能保存到浏览器，刷新后可能丢失'
}

function scheduleDrawingsSave() {
  if (drawingPersistTimer !== null) clearTimeout(drawingPersistTimer)
  drawingPersistTimer = setTimeout(saveDrawingsNow, 120)
}

function drawingCallbacks() {
  return {
    onDrawEnd: scheduleDrawingsSave,
    onPressedMoveEnd: scheduleDrawingsSave,
    onRemoved: scheduleDrawingsSave
  }
}

function restoreDrawings() {
  if (!chart) return
  drawings.value = getChartPreferences(activeChartSymbol).drawings
  for (const drawing of drawings.value) {
    chart.createOverlay({
      id: drawing.id,
      name: drawing.name,
      points: drawing.points.map(point => ({ ...point })),
      ...(drawing.styles ? { styles: drawing.styles as never } : {}),
      ...drawingCallbacks()
    })
  }
}

function startDrawing(name: DrawingToolName) {
  if (!chart) return
  activeDrawingTool.value = name
  chart.createOverlay({ name, ...drawingCallbacks() })
}

function removeDrawing(id: string) {
  chart?.removeOverlay({ id })
  scheduleDrawingsSave()
}

async function refreshOpenInterest() {
  cancelOpenInterestRequest()
  openInterestError.value = ''
  if (!chart || !studies.value.OPEN_INTEREST.enabled) return

  const interval = props.interval
  const intervalMs = openInterestIntervalMilliseconds(interval)
  const candles = history.candles.length ? history.candles : props.candles
  if (intervalMs === null || candles.length === 0) {
    chart.overrideIndicator({ name: 'OPEN_INTEREST', extendData: { samples: [], interval } })
    return
  }

  const symbol = props.symbol
  const version = openInterestRequestVersion
  const controller = new AbortController()
  openInterestController = controller
  try {
    const result = await api.getOpenInterestData(
      symbol, interval, 500, candles[0]!.timestamp, candles[candles.length - 1]!.timestamp + intervalMs, controller.signal
    )
    if (controller.signal.aborted || version !== openInterestRequestVersion || symbol !== props.symbol || interval !== props.interval) return
    chart?.overrideIndicator({ name: 'OPEN_INTEREST', extendData: { samples: result.data, interval } })
  } catch (error: unknown) {
    if (controller.signal.aborted || version !== openInterestRequestVersion || isAbortError(error)) return
    openInterestError.value = `OI 加载失败：${api.errorMessage(error, '请求失败')}，可重试`
  } finally {
    if (openInterestController === controller) openInterestController = null
  }
}

function applyStudies() {
  if (!chart) return
  for (const study of studyOptions) chart.removeIndicator({ name: study.name })
  for (const study of studyOptions) {
    const preference = studies.value[study.name]
    if (!preference.enabled) continue
    if (study.name === 'MA' || study.name === 'EMA' || study.name === 'BOLL') {
      chart.createIndicator({ name: study.name, calcParams: [...preference.params], paneId: 'candle_pane' }, true)
    } else if (study.name === 'OPEN_INTEREST') {
      chart.createIndicator({ name: study.name, extendData: { samples: [], interval: props.interval } })
    } else {
      chart.createIndicator({ name: study.name, calcParams: [...preference.params] })
    }
  }
}

function toggleStudy(name: ChartStudyName, enabled: boolean) {
  studyError.value = ''
  studies.value = { ...studies.value, [name]: { ...studies.value[name], enabled } }
  persistStudies()
  applyStudies()
  if (name === 'OPEN_INTEREST') void refreshOpenInterest()
}

function onStudyToggle(name: ChartStudyName, event: Event) {
  toggleStudy(name, (event.target as HTMLInputElement).checked)
}

function updateStudyParameter(name: ChartStudyName, index: number, event: Event) {
  const input = event.target as HTMLInputElement
  const current = studies.value[name]
  const params = [...current.params]
  params[index] = Number(input.value)
  const validation = validateChartStudyParameters(name, params)
  if (!validation.valid) {
    studyError.value = validation.error || '指标参数无效'
    input.value = String(current.params[index] ?? '')
    return
  }
  studyError.value = ''
  studies.value = { ...studies.value, [name]: { ...current, params } }
  persistStudies()
  applyStudies()
}

function toKlineData(candle: Candle): KLineData {
  return {
    timestamp: candle.timestamp,
    open: candle.open,
    high: candle.high,
    low: candle.low,
    close: candle.close,
    volume: candle.volume
  }
}

function sameCandle(left: Candle | undefined, right: Candle | undefined): boolean {
  return left?.timestamp === right?.timestamp && left?.open === right?.open && left?.high === right?.high &&
    left?.low === right?.low && left?.close === right?.close && left?.volume === right?.volume
}

function isAbortError(error: unknown): boolean {
  return error instanceof DOMException && error.name === 'AbortError' ||
    typeof error === 'object' && error !== null &&
    ('name' in error) && ((error as { name?: unknown }).name === 'CanceledError')
}

const dataLoader: DataLoader = {
  getBars: async ({ type, symbol, callback }) => {
    if (type === 'init') {
      history.seed(symbol.ticker, props.interval, props.candles, props.hasMoreBefore)
      historyError.value = ''
      callback(history.candles.map(toKlineData), { forward: history.hasMoreBefore, backward: false })
      return
    }
    if (type === 'forward') {
      try {
        const candles = await history.loadOlder(symbol.ticker, props.interval, 100)
        historyError.value = ''
        callback(candles.map(toKlineData), { forward: history.hasMoreBefore, backward: false })
        if (studies.value.OPEN_INTEREST.enabled) void refreshOpenInterest()
      } catch (error: unknown) {
        if (isAbortError(error)) return
        historyError.value = api.errorMessage(error, '加载更早 K 线失败，可继续拖动重试')
        // Resolve the chart's in-flight request without advancing the cursor.
        // KLineChart unlocks its loader and a subsequent drag can retry.
        callback([], { forward: history.hasMoreBefore, backward: false })
      }
      return
    }
    callback(history.candles.map(toKlineData), { forward: history.hasMoreBefore, backward: false })
  },
  subscribeBar: ({ callback }) => {
    liveBarCallback = callback
    lastPublishedCandle = props.candles[props.candles.length - 1]
  },
  unsubscribeBar: () => {
    liveBarCallback = null
  }
}

watch(() => props.candles, candles => {
  const latest = candles[candles.length - 1]
  if (latest && liveBarCallback && !sameCandle(lastPublishedCandle, latest)) {
    liveBarCallback(toKlineData(latest))
  }
  lastPublishedCandle = latest
}, { deep: true })

watch([() => props.symbol, () => props.interval], ([symbol, interval]) => {
  saveDrawingsNow()
  const symbolChanged = symbol !== activeChartSymbol
  cancelOpenInterestRequest()
  activeChartSymbol = symbol
  studies.value = loadStudyPreferences(symbol)
  history.seed(symbol, interval, props.candles, props.hasMoreBefore)
  chart?.setSymbol({ ticker: symbol, pricePrecision: 8, volumePrecision: 2 })
  chart?.setPeriod(toKlineChartPeriod(interval))
  chart?.resetData()
  applyStudies()
  if (chart && symbolChanged) {
    chart.removeOverlay()
    activeDrawingTool.value = null
    restoreDrawings()
  }
  void refreshOpenInterest()
})

onMounted(() => {
  const container = chartHost.value
  if (!container) return

  history.seed(props.symbol, props.interval, props.candles, props.hasMoreBefore)
  chart = init(container, { styles: 'dark', timezone: 'Asia/Shanghai' })
  if (!chart) return
  chart.setSymbol({ ticker: props.symbol, pricePrecision: 8, volumePrecision: 2 })
  chart.setPeriod(toKlineChartPeriod(props.interval))
  chart.setScrollEnabled(true)
  chart.setZoomEnabled(true)
  chart.setDataLoader(dataLoader)
  applyStudies()
  restoreDrawings()
  void refreshOpenInterest()
  openInterestRefreshTimer = setInterval(() => {
    if (studies.value.OPEN_INTEREST.enabled) void refreshOpenInterest()
  }, 60_000)

  if (typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(() => chart?.resize())
    resizeObserver.observe(container)
  }
})

onBeforeUnmount(() => {
  saveDrawingsNow()
  if (drawingPersistTimer !== null) clearTimeout(drawingPersistTimer)
  drawingPersistTimer = null
  cancelOpenInterestRequest()
  if (openInterestRefreshTimer !== null) clearInterval(openInterestRefreshTimer)
  openInterestRefreshTimer = null
  liveBarCallback = null
  resizeObserver?.disconnect()
  resizeObserver = null
  history.reset(props.symbol, props.interval)
  if (chart) dispose(chart)
  chart = null
})
</script>

<style scoped>
.chart-card {
  overflow: hidden;
  padding: 18px;
  border: 1px solid #333;
  border-radius: 12px;
  background: #171717;
  color: #eee;
}

.chart-header { display: flex; justify-content: space-between; align-items: center; gap: 16px; margin-bottom: 14px; }
.chart-header h3 { display: inline; margin: 0; font-size: 18px; }
.chart-interval { margin-left: 10px; color: #888; font-size: 12px; }
.current-price { display: flex; align-items: baseline; gap: 8px; }
.label { color: #999; font-size: 12px; }
.price, .change { font-size: 14px; }
.positive { color: #26a69a; }
.negative { color: #ef5350; }
.chart-summary { display: flex; gap: 22px; flex-wrap: wrap; margin: 12px 0; }
.info-item { display: flex; gap: 6px; align-items: baseline; font-size: 12px; }
.info-item strong { color: #ddd; font-weight: 500; }
.chart-toolbar { display: flex; align-items: flex-start; gap: 10px; flex-wrap: wrap; margin: 12px 0; }
.drawing-tools { display: flex; flex-wrap: wrap; gap: 6px; }
.chart-tool-button[aria-pressed="true"] { border-color: #d8a900; color: #f0c330; }
.saved-drawings { display: flex; flex-wrap: wrap; gap: 6px; width: 100%; }
.saved-drawing { display: inline-flex; align-items: center; gap: 6px; padding: 3px 7px; border: 1px solid #414141; border-radius: 12px; color: #aaa; font-size: 11px; }
.saved-drawing button { padding: 0; border: 0; background: transparent; color: #aaa; font-size: 15px; cursor: pointer; }
.chart-tool-button { padding: 6px 10px; border: 1px solid #414141; border-radius: 6px; background: #202020; color: #ddd; cursor: pointer; }
.study-settings { display: grid; grid-template-columns: repeat(4, minmax(140px, 1fr)); gap: 12px; width: 100%; padding: 12px; border: 1px solid #383838; border-radius: 8px; background: #1d1d1d; }
.study-setting { min-width: 0; }
.study-toggle { display: flex; align-items: center; gap: 7px; margin-bottom: 8px; color: #ddd; font-size: 12px; }
.study-parameters { display: flex; flex-wrap: wrap; gap: 6px; }
.study-parameter { display: grid; gap: 4px; color: #999; font-size: 10px; }
.study-parameter input { width: 58px; padding: 4px; border: 1px solid #414141; border-radius: 4px; background: #141414; color: #eee; }
.study-error { grid-column: 1 / -1; margin: 0; color: #ff8a80; font-size: 12px; }
.kline-chart { width: 100%; height: 480px; min-width: 0; }
.history-error { margin: 8px 0 0; color: #ff8a80; font-size: 12px; }

@media (max-width: 600px) {
  .chart-card { padding: 12px; }
  .chart-header { align-items: flex-start; flex-direction: column; }
  .chart-summary { gap: 10px 16px; }
  .study-settings { grid-template-columns: repeat(2, minmax(130px, 1fr)); }
  .kline-chart { height: 380px; }
}
</style>
