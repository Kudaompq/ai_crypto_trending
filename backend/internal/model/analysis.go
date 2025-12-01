package model

// TrendAnalysis represents the overall trend analysis result
type TrendAnalysis struct {
	Direction         string  `json:"direction"`          // "上升" / "下降" / "盘整"
	Strength          float64 `json:"strength"`           // 0-1, 趋势强度
	ChangeProbability float64 `json:"change_probability"` // 趋势反转概率
}

// MACDIndicator represents MACD indicator values
type MACDIndicator struct {
	DIF       float64 `json:"dif"`
	DEA       float64 `json:"dea"`
	Histogram float64 `json:"histogram"`
}

// KDJIndicator represents KDJ indicator values
type KDJIndicator struct {
	K float64 `json:"k"`
	D float64 `json:"d"`
	J float64 `json:"j"`
}

// RSIIndicator represents RSI indicator values
type RSIIndicator struct {
	RSI6  float64 `json:"rsi6"`
	RSI14 float64 `json:"rsi14"`
}

// ATRIndicator represents ATR (Average True Range) indicator
type ATRIndicator struct {
	Value  float64 `json:"value"`  // Current ATR value
	Period int     `json:"period"` // Period used (typically 14)
}

// EMAIndicator represents EMA values for multiple periods
type EMAIndicator struct {
	EMA9   float64 `json:"ema9"`   // 9-period EMA
	EMA21  float64 `json:"ema21"`  // 21-period EMA
	EMA50  float64 `json:"ema50"`  // 50-period EMA
	EMA200 float64 `json:"ema200"` // 200-period EMA
}

// FibonacciLevels represents Fibonacci retracement and extension levels
type FibonacciLevels struct {
	High        float64            `json:"high"`        // Swing high
	Low         float64            `json:"low"`         // Swing low
	Retracement map[string]float64 `json:"retracement"` // Retracement levels
	Extension   map[string]float64 `json:"extension"`   // Extension levels
	Direction   string             `json:"direction"`   // "UPTREND" or "DOWNTREND"
}

// Indicators contains all technical indicators
type Indicators struct {
	MACD      MACDIndicator    `json:"macd"`
	KDJ       KDJIndicator     `json:"kdj"`
	RSI       RSIIndicator     `json:"rsi"`
	ATR       ATRIndicator     `json:"atr"`
	EMA       EMAIndicator     `json:"ema"`
	Fibonacci *FibonacciLevels `json:"fibonacci,omitempty"`
}

// SRLevel represents a support or resistance level
type SRLevel struct {
	Price    float64 `json:"price"`
	Strength float64 `json:"strength"` // 0-1, 强度
}

// SRLevels contains support and resistance levels
type SRLevels struct {
	Resistance []SRLevel `json:"resistance"`
	Support    []SRLevel `json:"support"`
}

// CandlestickPattern represents an identified candlestick pattern
type CandlestickPattern struct {
	Pattern     string  `json:"pattern"`     // 形态名称
	Type        string  `json:"type"`        // "反转" or "持续"
	Direction   string  `json:"direction"`   // "看涨" or "看跌"
	Position    int     `json:"position"`    // 相对当前K线的位置
	Reliability float64 `json:"reliability"` // 0-1, 可靠性
	Description string  `json:"description"` // 描述
}

// MarketStructure represents comprehensive market structure analysis
type MarketStructure struct {
	// Basic Structure
	HigherHigh     bool   `json:"higher_high"`     // 是否创新高
	HigherLow      bool   `json:"higher_low"`      // 是否创新低
	StructureBreak bool   `json:"structure_break"` // 结构是否被破坏
	RiskLevel      string `json:"risk_level"`      // "低" / "中" / "高"

	// Enhanced Multi-Indicator Analysis
	TrendConfirmation  TrendConfirmation  `json:"trend_confirmation"`
	VolatilityProfile  VolatilityProfile  `json:"volatility_profile"`
	KeyLevelConfluence KeyLevelConfluence `json:"key_level_confluence"`
	PatternSignals     PatternSignals     `json:"pattern_signals"`
	MarketQuality      MarketQuality      `json:"market_quality"`
}

// TrendConfirmation analyzes trend strength using multiple indicators
type TrendConfirmation struct {
	EMAAlignment      string  `json:"ema_alignment"`      // "BULLISH", "BEARISH", "NEUTRAL"
	MACDSignal        string  `json:"macd_signal"`        // "BULLISH", "BEARISH", "NEUTRAL"
	PriceVsEMA        string  `json:"price_vs_ema"`       // Price position relative to key EMAs
	ConfirmationScore float64 `json:"confirmation_score"` // 0-100
	Strength          string  `json:"strength"`           // "STRONG", "MODERATE", "WEAK"
}

// VolatilityProfile assesses market volatility for risk management
type VolatilityProfile struct {
	CurrentATR      float64 `json:"current_atr"`
	ATRPercentage   float64 `json:"atr_percentage"`   // ATR as % of price
	VolatilityLevel string  `json:"volatility_level"` // "HIGH", "NORMAL", "LOW"
	IsExpanding     bool    `json:"is_expanding"`     // Volatility trend
	RiskAdjustment  string  `json:"risk_adjustment"`  // Suggested position sizing
}

// KeyLevelConfluence identifies confluence zones
type KeyLevelConfluence struct {
	NearestSupport    *ConfluenceLevel `json:"nearest_support"`
	NearestResistance *ConfluenceLevel `json:"nearest_resistance"`
	ConfluenceZones   []ConfluenceZone `json:"confluence_zones"`
}

// ConfluenceLevel represents a key level with multiple confirmations
type ConfluenceLevel struct {
	Price    float64  `json:"price"`
	Distance float64  `json:"distance"` // Distance from current price (%)
	Factors  []string `json:"factors"`  // What confirms this level
	Strength float64  `json:"strength"` // 0-100
	Type     string   `json:"type"`     // "SUPPORT" or "RESISTANCE"
}

// ConfluenceZone represents an area where multiple factors align
type ConfluenceZone struct {
	PriceRange   [2]float64 `json:"price_range"` // [low, high]
	Factors      []string   `json:"factors"`
	Strength     float64    `json:"strength"`
	Significance string     `json:"significance"` // "CRITICAL", "IMPORTANT", "MODERATE"
}

// PatternSignals aggregates candlestick pattern insights
type PatternSignals struct {
	RecentPatterns     []string `json:"recent_patterns"`
	BullishCount       int      `json:"bullish_count"`
	BearishCount       int      `json:"bearish_count"`
	DominantSignal     string   `json:"dominant_signal"`     // "BULLISH", "BEARISH", "NEUTRAL"
	PatternReliability float64  `json:"pattern_reliability"` // 0-100
}

// MarketQuality provides overall market structure quality assessment
type MarketQuality struct {
	OverallScore     float64            `json:"overall_score"`     // 0-100
	Grade            string             `json:"grade"`             // "A", "B", "C", "D", "F"
	TradingCondition string             `json:"trading_condition"` // "EXCELLENT", "GOOD", "FAIR", "POOR"
	Strengths        []string           `json:"strengths"`
	Weaknesses       []string           `json:"weaknesses"`
	Recommendation   string             `json:"recommendation"`
	ScoreBreakdown   map[string]float64 `json:"score_breakdown"`
}

// RiskReward represents risk-reward analysis for a trading opportunity
type RiskReward struct {
	Ratio        float64 `json:"ratio"`         // Risk-reward ratio (e.g., 3.0 for 3:1)
	RiskAmount   float64 `json:"risk_amount"`   // Amount at risk
	RewardAmount float64 `json:"reward_amount"` // Potential reward
	RiskPct      float64 `json:"risk_pct"`      // Risk as percentage
	RewardPct    float64 `json:"reward_pct"`    // Reward as percentage
}

// AnalysisResult represents the complete analysis result
type AnalysisResult struct {
	Symbol              string               `json:"symbol"`
	Interval            string               `json:"interval"`
	Timestamp           int64                `json:"timestamp"`
	Trend               TrendAnalysis        `json:"trend"`
	Indicators          Indicators           `json:"indicators"`
	SRLevels            SRLevels             `json:"sr_levels"`
	CandlestickPatterns []CandlestickPattern `json:"candlestick_patterns"`
	MarketStructure     MarketStructure      `json:"market_structure"`
}

// EntryPoint represents the entry point for a trade
type EntryPoint struct {
	Price   float64  `json:"price"`
	Reasons []string `json:"reasons"`
}

// StopLossInfo represents stop-loss information
type StopLossInfo struct {
	Price       float64 `json:"price"`
	DistancePct float64 `json:"distance_pct"`
	Method      string  `json:"method"` // "TECHNICAL_LEVEL", "ATR", "PERCENTAGE"
}

// TakeProfitLevel represents a take-profit target
type TakeProfitLevel struct {
	Level            int     `json:"level"`
	Price            float64 `json:"price"`
	DistancePct      float64 `json:"distance_pct"`
	Target           string  `json:"target"`
	PositionClosePct int     `json:"position_close_pct"`
}

// RiskRewardInfo represents risk-reward analysis
type RiskRewardInfo struct {
	Ratio        float64 `json:"ratio"`
	RiskAmount   float64 `json:"risk_amount"`
	RewardAmount float64 `json:"reward_amount"`
	RiskPct      float64 `json:"risk_pct"`
	RewardPct    float64 `json:"reward_pct"`
}

// ConfidenceInfo represents confidence scoring
type ConfidenceInfo struct {
	Score   int      `json:"score"`   // 0-100
	Level   string   `json:"level"`   // "HIGH", "MEDIUM", "LOW"
	Factors []string `json:"factors"` // Reasons for confidence level
}

// ValidityInfo represents opportunity validity
type ValidityInfo struct {
	ExpiresAt int64  `json:"expires_at"`
	Status    string `json:"status"` // "ACTIVE", "EXPIRED"
}

// TradingOpportunity represents a trading opportunity
type TradingOpportunity struct {
	ID         string            `json:"id"`
	Symbol     string            `json:"symbol"`
	Type       string            `json:"type"`     // "LONG" or "SHORT"
	Strategy   string            `json:"strategy"` // "SUPPORT_BOUNCE", "BREAKOUT_RETEST", "TREND_CONTINUATION"
	Timestamp  int64             `json:"timestamp"`
	Entry      EntryPoint        `json:"entry"`
	StopLoss   StopLossInfo      `json:"stop_loss"`
	TakeProfit []TakeProfitLevel `json:"take_profit"`
	RiskReward RiskRewardInfo    `json:"risk_reward"`
	Confidence ConfidenceInfo    `json:"confidence"`
	Validity   ValidityInfo      `json:"validity"`
}

// OpportunitySummary represents a summary of opportunities
type OpportunitySummary struct {
	TotalCount          int     `json:"total_opportunities"`
	AvgRiskReward       float64 `json:"avg_risk_reward"`
	HighConfidenceCount int     `json:"high_confidence_count"`
}

// OpportunitiesResponse represents the API response for opportunities
type OpportunitiesResponse struct {
	Opportunities []TradingOpportunity `json:"opportunities"`
	Summary       OpportunitySummary   `json:"summary"`
}
