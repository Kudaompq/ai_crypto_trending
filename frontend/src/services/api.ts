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

export interface Indicators {
  macd: MACDIndicator
  kdj: KDJIndicator
  rsi: RSIIndicator
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

export interface MarketStructure {
  higher_high: boolean
  higher_low: boolean
  structure_break: boolean
  risk_level: string
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

  async healthCheck(): Promise<{ status: string; message: string }> {
    const response = await axios.get(`${API_BASE_URL}/health`)
    return response.data
  }
}
