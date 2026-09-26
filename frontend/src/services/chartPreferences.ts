export const CHART_PREFERENCES_STORAGE_KEY = 'crypto-trending-chart-preferences-v2'
const LEGACY_CHART_PREFERENCES_STORAGE_KEY = 'crypto-trending-chart-preferences-v1'

export type ChartStudyName = 'MA' | 'EMA' | 'BOLL' | 'MACD' | 'OPEN_INTEREST'
export interface ChartStudyPreference {
  enabled: boolean
  params: number[]
}
export interface SavedChartDrawing {
  id: string
  name: string
  points: Array<{ timestamp: number; value: number }>
  styles?: Record<string, unknown>
  visible?: boolean
  lock?: boolean
  mode?: 'normal' | 'weak_magnet' | 'strong_magnet'
}
export interface SymbolChartPreferences {
  studies: Partial<Record<ChartStudyName, ChartStudyPreference>>
  drawings: SavedChartDrawing[]
}

export const DEFAULT_STUDY_PREFERENCES: Record<ChartStudyName, ChartStudyPreference> = {
  MA: { enabled: false, params: [5, 10, 30, 60] },
  EMA: { enabled: false, params: [6, 12, 20] },
  BOLL: { enabled: false, params: [20, 2] },
  MACD: { enabled: false, params: [12, 26, 9] },
  OPEN_INTEREST: { enabled: false, params: [] }
}

const studyNames: ChartStudyName[] = ['MA', 'EMA', 'BOLL', 'MACD', 'OPEN_INTEREST']
const drawingNames = new Set([
  'horizontalStraightLine', 'horizontalRayLine', 'horizontalSegment', 'verticalStraightLine', 'verticalRayLine',
  'verticalSegment', 'straightLine', 'rayLine', 'segment', 'arrow', 'priceLine', 'priceChannelLine',
  'parallelStraightLine', 'circle', 'rect', 'parallelogram', 'triangle', 'fibonacciLine', 'fibonacciSegment',
  'fibonacciCircle', 'fibonacciSpiral', 'fibonacciSpeedResistanceFan', 'fibonacciExtension', 'gannBox',
  'xabcd', 'abcd', 'threeWaves', 'fiveWaves', 'eightWaves', 'anyWaves'
])

interface ChartPreferencesDocument {
  version: 2
  studies: Partial<Record<ChartStudyName, ChartStudyPreference>>
  symbols: Record<string, { drawings: SavedChartDrawing[] }>
}

interface LegacyChartPreferencesDocument {
  studiesBySymbol: Record<string, Partial<Record<ChartStudyName, ChartStudyPreference>>>
  symbols: ChartPreferencesDocument['symbols']
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function normalizedSymbol(symbol: string): string {
  return symbol.trim().toUpperCase()
}

function emptyPreferences(): SymbolChartPreferences {
  return { studies: {}, drawings: [] }
}

function validDrawing(value: unknown): value is SavedChartDrawing {
  if (!isRecord(value) || typeof value.id !== 'string' || typeof value.name !== 'string' || !drawingNames.has(value.name)) return false
  if (value.visible !== undefined && typeof value.visible !== 'boolean') return false
  if (value.lock !== undefined && typeof value.lock !== 'boolean') return false
  if (value.mode !== undefined && value.mode !== 'normal' && value.mode !== 'weak_magnet' && value.mode !== 'strong_magnet') return false
  if (!Array.isArray(value.points) || value.points.length === 0) return false
  return value.points.every(point => isRecord(point) && typeof point.timestamp === 'number' && Number.isFinite(point.timestamp) &&
    typeof point.value === 'number' && Number.isFinite(point.value))
}

function copyStudies(value: Partial<Record<ChartStudyName, ChartStudyPreference>>): Partial<Record<ChartStudyName, ChartStudyPreference>> {
  return Object.fromEntries(Object.entries(value).map(([name, preference]) => [
    name,
    preference ? { enabled: preference.enabled, params: [...preference.params] } : preference
  ])) as Partial<Record<ChartStudyName, ChartStudyPreference>>
}

function parseStudies(value: unknown): Partial<Record<ChartStudyName, ChartStudyPreference>> {
  const studies: Partial<Record<ChartStudyName, ChartStudyPreference>> = {}
  if (!isRecord(value)) return studies
  for (const name of studyNames) {
    const study = value[name]
    if (!isRecord(study) || typeof study.enabled !== 'boolean' || !Array.isArray(study.params)) continue
    const params = study.params.filter((param): param is number => typeof param === 'number' && Number.isFinite(param))
    if (params.length !== study.params.length || !validateChartStudyParameters(name, params).valid) continue
    studies[name] = { enabled: study.enabled, params }
  }
  return studies
}

function parseDrawings(value: unknown): SavedChartDrawing[] {
  if (!Array.isArray(value)) return []
  return value.filter(validDrawing).map(drawing => ({
    ...drawing,
    points: drawing.points.map(point => ({ ...point })),
    ...(drawing.styles ? { styles: { ...drawing.styles } } : {})
  }))
}

function emptyDocument(): ChartPreferencesDocument {
  return { version: 2, studies: {}, symbols: {} }
}

function writeDocument(document: ChartPreferencesDocument): boolean {
  try {
    localStorage.setItem(CHART_PREFERENCES_STORAGE_KEY, JSON.stringify(document))
    return true
  } catch {
    return false
  }
}

function readLegacyDocument(): LegacyChartPreferencesDocument | null {
  try {
    const parsed: unknown = JSON.parse(localStorage.getItem(LEGACY_CHART_PREFERENCES_STORAGE_KEY) || 'null')
    if (!isRecord(parsed) || parsed.version !== 1 || !isRecord(parsed.symbols)) return null
    const symbols: ChartPreferencesDocument['symbols'] = {}
    const studiesBySymbol: Record<string, Partial<Record<ChartStudyName, ChartStudyPreference>>> = {}
    for (const [symbol, raw] of Object.entries(parsed.symbols)) {
      if (!/^[A-Z0-9]{5,30}$/.test(symbol) || !isRecord(raw)) continue
      studiesBySymbol[symbol] = parseStudies(raw.studies)
      symbols[symbol] = { drawings: parseDrawings(raw.drawings) }
    }
    return { studiesBySymbol, symbols }
  } catch {
    return null
  }
}

function readDocument(preferredSymbol = ''): ChartPreferencesDocument {
  try {
    const parsed: unknown = JSON.parse(localStorage.getItem(CHART_PREFERENCES_STORAGE_KEY) || 'null')
    if (isRecord(parsed) && parsed.version === 2 && isRecord(parsed.symbols)) {
      const symbols: ChartPreferencesDocument['symbols'] = {}
      for (const [symbol, raw] of Object.entries(parsed.symbols)) {
        if (!/^[A-Z0-9]{5,30}$/.test(symbol) || !isRecord(raw)) continue
        symbols[symbol] = { drawings: parseDrawings(raw.drawings) }
      }
      return { version: 2, studies: parseStudies(parsed.studies), symbols }
    }
  } catch {
    // Fall through to legacy data or defaults.
  }

  const legacy = readLegacyDocument()
  if (!legacy) return emptyDocument()

  const selectedStudies = legacy.studiesBySymbol[normalizedSymbol(preferredSymbol)]
  const fallbackStudies = Object.values(legacy.studiesBySymbol).find(studies => Object.keys(studies).length > 0)
  const document: ChartPreferencesDocument = {
    version: 2,
    studies: copyStudies(selectedStudies && Object.keys(selectedStudies).length > 0 ? selectedStudies : (fallbackStudies || {})),
    symbols: legacy.symbols
  }
  if (writeDocument(document)) {
    try { localStorage.removeItem(LEGACY_CHART_PREFERENCES_STORAGE_KEY) } catch { /* Keep migrated settings usable in memory. */ }
  }
  return document
}

function normalizePreferences(value: SymbolChartPreferences): SymbolChartPreferences | null {
  if (!isRecord(value) || !isRecord(value.studies) || !Array.isArray(value.drawings)) return null
  const preferences = emptyPreferences()
  for (const name of studyNames) {
    const study = value.studies[name]
    if (study === undefined) continue
    if (!isRecord(study) || typeof study.enabled !== 'boolean' || !Array.isArray(study.params) ||
      !study.params.every(param => typeof param === 'number') || !validateChartStudyParameters(name, study.params as number[]).valid) return null
    preferences.studies[name] = { enabled: study.enabled, params: [...study.params] as number[] }
  }
  if (!value.drawings.every(validDrawing)) return null
  preferences.drawings = value.drawings.map(drawing => ({
    ...drawing,
    points: drawing.points.map(point => ({ ...point })),
    ...(drawing.styles ? { styles: { ...drawing.styles } } : {})
  }))
  return preferences
}

export function getChartPreferences(symbol: string): SymbolChartPreferences {
  const key = normalizedSymbol(symbol)
  if (!/^[A-Z0-9]{5,30}$/.test(key)) return emptyPreferences()
  const document = readDocument(key)
  const preferences = document.symbols[key] || { drawings: [] }
  return { studies: copyStudies(document.studies), drawings: parseDrawings(preferences.drawings) }
}

export function saveChartPreferences(symbol: string, value: SymbolChartPreferences): boolean {
  const key = normalizedSymbol(symbol)
  const preferences = normalizePreferences(value)
  if (!/^[A-Z0-9]{5,30}$/.test(key) || !preferences) return false
  const document = readDocument(key)
  document.studies = preferences.studies
  document.symbols[key] = { drawings: preferences.drawings }
  return writeDocument(document)
}

export function validateChartStudyParameters(study: ChartStudyName, params: number[]): { valid: boolean; error?: string } {
  const invalid = (error: string) => ({ valid: false, error })
  if (!Array.isArray(params) || params.some(param => !Number.isFinite(param))) return invalid('指标参数必须为有限数值')
  const positivePeriod = (param: number) => Number.isInteger(param) && param > 0
  if (study === 'OPEN_INTEREST') {
    if (params.length !== 0) return invalid('OI 无需参数')
  } else if (study === 'MA' || study === 'EMA') {
    if (params.length < 1 || params.length > 6 || !params.every(positivePeriod)) return invalid('均线周期必须是 1 到 6 个正整数')
  } else if (study === 'BOLL') {
    if (params.length !== 2 || !positivePeriod(params[0]!) || params[1]! <= 0) return invalid('布林带需要正整数周期和正数标准差')
  } else if (study === 'MACD') {
    if (params.length !== 3 || !params.every(positivePeriod) || params[0]! >= params[1]!) return invalid('MACD 快线周期必须小于慢线周期，且信号周期为正整数')
  } else {
    return invalid('不支持的指标')
  }
  return { valid: true }
}
