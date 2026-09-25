import axios from 'axios'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api'

export interface Candle {
  timestamp: number
  open: number
  high: number
  low: number
  close: number
  volume: number
}

export interface KlineData {
  symbol: string
  interval: string
  data: Candle[]
}

export interface MarketEvent {
  type: 'kline'
  symbol: string
  interval: string
  candle: Candle
  is_final: boolean
  event_time: number
}

export interface StreamStatus {
  symbol: string
  interval: string
  state: 'connecting' | 'reconnecting'
  message: string
}

export interface PriceQuote {
  symbol: string
  price: number
  event_time: number
}

export interface PriceStreamStatus {
  state: 'connecting' | 'live' | 'reconnecting'
  message: string
  unavailable_symbols?: string[]
}

export interface PriceSnapshot {
  prices: PriceQuote[]
  unavailable_symbols: string[]
}

export interface TrendAnalysis {
  direction: string
  strength: number
  change_probability: number
}

export interface MACDIndicator {
  dif: number
  dea: number
  histogram: number
}

export interface KDJIndicator {
  k: number
  d: number
  j: number
}

export interface RSIIndicator {
  rsi6: number
  rsi14: number
}

export interface ATRIndicator {
  value: number
  period: number
}

export interface EMAIndicator {
  ema9: number
  ema21: number
  ema50: number
  ema200: number
}

export interface FibonacciLevels {
  high: number
  low: number
  retracement: Record<string, number>
  extension: Record<string, number>
  direction: string
}

export interface Indicators {
  macd: MACDIndicator
  kdj: KDJIndicator
  rsi: RSIIndicator
  atr: ATRIndicator
  ema: EMAIndicator
  fibonacci?: FibonacciLevels
}

export interface SRLevel {
  price: number
  strength: number
}

export interface SRLevels {
  resistance: SRLevel[]
  support: SRLevel[]
}

export interface CandlestickPattern {
  pattern: string
  type: string
  direction: string
  position: number
  reliability: number
  description: string
}

export interface TrendConfirmation {
  ema_alignment: string
  macd_signal: string
  price_vs_ema: string
  confirmation_score: number
  strength: string
}

export interface VolatilityProfile {
  current_atr: number
  atr_percentage: number
  volatility_level: string
  is_expanding: boolean
  risk_adjustment: string
}

export interface ConfluenceLevel {
  price: number
  distance: number
  factors: string[]
  strength: number
  type: string
}

export interface ConfluenceZone {
  price_range: [number, number]
  factors: string[]
  strength: number
  significance: string
}

export interface KeyLevelConfluence {
  nearest_support: ConfluenceLevel | null
  nearest_resistance: ConfluenceLevel | null
  confluence_zones: ConfluenceZone[]
}

export interface PatternSignals {
  recent_patterns: string[]
  bullish_count: number
  bearish_count: number
  dominant_signal: string
  pattern_reliability: number
}

export interface MarketQuality {
  overall_score: number
  grade: string
  trading_condition: string
  strengths: string[]
  weaknesses: string[]
  recommendation: string
  score_breakdown: Record<string, number>
}

export interface MarketStructure {
  higher_high: boolean
  higher_low: boolean
  structure_break: boolean
  risk_level: string
  trend_confirmation: TrendConfirmation
  volatility_profile: VolatilityProfile
  key_level_confluence: KeyLevelConfluence
  pattern_signals: PatternSignals
  market_quality: MarketQuality
}

export interface AnalysisResult {
  symbol: string
  interval: string
  timestamp: number
  trend: TrendAnalysis
  indicators: Indicators
  sr_levels: SRLevels
  candlestick_patterns: CandlestickPattern[]
  market_structure: MarketStructure
}

export const api = {
  invalidSelection(err: unknown): boolean {
    return axios.isAxiosError(err) && (err.response?.status === 400 || err.response?.status === 404)
  },
  errorMessage(err: unknown, fallback: string): string {
    if (axios.isAxiosError<{ error?: string }>(err)) {
      return err.response?.data?.error || fallback
    }
    return err instanceof Error ? err.message : fallback
  },

  async validateSymbol(symbol: string): Promise<string> {
    const response = await axios.get<{ symbol: string }>(`${API_BASE_URL}/symbols/validate`, {
      params: { symbol }
    })
    return response.data.symbol
  },

  async getKlineData(symbol: string, interval: string, limit: number): Promise<KlineData> {
    const response = await axios.get(`${API_BASE_URL}/kline`, {
      params: { symbol, interval, limit }
    })
    return response.data
  },

  async getWatchlistPrices(symbols: string[]): Promise<PriceSnapshot> {
    const response = await axios.get<PriceSnapshot>(`${API_BASE_URL}/watchlist/prices`, {
      params: { symbols: symbols.join(',') }
    })
    return response.data
  },

  async getAnalysis(symbol: string, interval: string, limit: number): Promise<AnalysisResult> {
    const response = await axios.get(`${API_BASE_URL}/analysis`, {
      params: { symbol, interval, limit }
    })
    return response.data
  },

  async healthCheck(): Promise<{ status: string; message: string }> {
    const response = await axios.get(`${API_BASE_URL}/health`)
    return response.data
  },

  createMarketStream(
    symbol: string,
    interval: string,
    onEvent: (event: MarketEvent) => void,
    onStatus?: (status: StreamStatus) => void
  ): EventSource {
    const params = new URLSearchParams({ symbol, interval })
    const source = new EventSource(`${API_BASE_URL}/stream?${params.toString()}`)
    source.addEventListener('kline', (message) => {
      onEvent(JSON.parse((message as MessageEvent).data) as MarketEvent)
    })
    source.addEventListener('status', (message) => {
      onStatus?.(JSON.parse((message as MessageEvent).data) as StreamStatus)
    })
    return source
  },

  createWatchlistPriceStream(
    symbols: string[],
    onPrice: (event: PriceQuote) => void,
    onStatus?: (status: PriceStreamStatus) => void
  ): EventSource {
    const params = new URLSearchParams({ symbols: symbols.join(',') })
    const source = new EventSource(`${API_BASE_URL}/watchlist/stream?${params.toString()}`)
    source.addEventListener('ready', (message) => {
      const ready = JSON.parse((message as MessageEvent).data) as {
        symbols?: string[]
        unavailable_symbols?: string[]
      }
      onStatus?.({ state: 'connecting', message: '正在连接实时行情', unavailable_symbols: ready.unavailable_symbols || [] })
      if (Array.isArray(ready.symbols) && ready.symbols.length === 0) source.close()
    })
    source.addEventListener('price', (message) => {
      onPrice(JSON.parse((message as MessageEvent).data) as PriceQuote)
    })
    source.addEventListener('status', (message) => {
      onStatus?.(JSON.parse((message as MessageEvent).data) as PriceStreamStatus)
    })
    return source
  }
}
