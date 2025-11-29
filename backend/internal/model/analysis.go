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

// Indicators contains all technical indicators
type Indicators struct {
	MACD MACDIndicator `json:"macd"`
	KDJ  KDJIndicator  `json:"kdj"`
	RSI  RSIIndicator  `json:"rsi"`
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
	Pattern     string  `json:"pattern"`      // 形态名称
	Type        string  `json:"type"`         // "反转" or "持续"
	Direction   string  `json:"direction"`    // "看涨" or "看跌"
	Position    int     `json:"position"`     // 相对当前K线的位置
	Reliability float64 `json:"reliability"`  // 0-1, 可靠性
	Description string  `json:"description"`  // 描述
}

// MarketStructure represents market structure analysis
type MarketStructure struct {
	HigherHigh     bool   `json:"higher_high"`      // 是否创新高
	HigherLow      bool   `json:"higher_low"`       // 是否创新低
	StructureBreak bool   `json:"structure_break"`  // 结构是否被破坏
	RiskLevel      string `json:"risk_level"`       // "低" / "中" / "高"
}

// AnalysisResult represents the complete analysis result
type AnalysisResult struct {
	Symbol            string                `json:"symbol"`
	Interval          string                `json:"interval"`
	Timestamp         int64                 `json:"timestamp"`
	Trend             TrendAnalysis         `json:"trend"`
	Indicators        Indicators            `json:"indicators"`
	SRLevels          SRLevels              `json:"sr_levels"`
	CandlestickPatterns []CandlestickPattern `json:"candlestick_patterns"`
	MarketStructure   MarketStructure       `json:"market_structure"`
}
