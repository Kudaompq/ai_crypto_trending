import axios from 'axios'

const API_BASE_URL = 'http://localhost:8080/api'

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

// Trading Opportunity Types
export interface EntryPoint {
  price: number
  reasons: string[]
}

export interface StopLossInfo {
  price: number
  distance_pct: number
  method: string
}

export interface TakeProfitLevel {
  level: number
  price: number
  distance_pct: number
  target: string
  position_close_pct: number
}

export interface RiskRewardInfo {
  ratio: number
  risk_amount: number
  reward_amount: number
  risk_pct: number
  reward_pct: number
}

export interface ConfidenceInfo {
  score: number
  level: string
  factors: string[]
}

export interface ValidityInfo {
  expires_at: number
  status: string
}

export interface TradingOpportunity {
  id: string
  symbol: string
  type: string
  strategy: string
  timestamp: number
  entry: EntryPoint
  stop_loss: StopLossInfo
  take_profit: TakeProfitLevel[]
  risk_reward: RiskRewardInfo
  confidence: ConfidenceInfo
  validity: ValidityInfo
}

export interface OpportunitySummary {
  total_opportunities: number
  avg_risk_reward: number
  high_confidence_count: number
}

export interface OpportunitiesResponse {
  opportunities: TradingOpportunity[]
  summary: OpportunitySummary
}

export const api = {
  async getKlineData(symbol: string, interval: string, limit: number): Promise<KlineData> {
    const response = await axios.get(`${API_BASE_URL}/kline`, {
      params: { symbol, interval, limit }
    })
    return response.data
  },

  async getAnalysis(symbol: string, interval: string, limit: number): Promise<AnalysisResult> {
    const response = await axios.get(`${API_BASE_URL}/analysis`, {
      params: { symbol, interval, limit }
    })
    return response.data
  },

  async getOpportunities(symbol: string, interval: string, minRR: number = 3.0): Promise<OpportunitiesResponse> {
    const response = await axios.get(`${API_BASE_URL}/opportunities`, {
      params: { symbol, interval, min_rr: minRR, limit: 100 }
    })
    return response.data
  },

  async healthCheck(): Promise<{ status: string; message: string }> {
    const response = await axios.get(`${API_BASE_URL}/health`)
    return response.data
  }
}
