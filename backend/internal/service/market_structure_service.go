package service

import (
	"math"
	"sort"

	"github.com/kudaompq/ai_trending/backend/internal/indicator"
	"github.com/kudaompq/ai_trending/backend/internal/model"
)

// MarketStructureService handles comprehensive market structure analysis
type MarketStructureService struct{}

// NewMarketStructureService creates a new market structure service
func NewMarketStructureService() *MarketStructureService {
	return &MarketStructureService{}
}

// levelInfo represents a price level with its associated factor and weight
type levelInfo struct {
	price  float64
	factor string
	weight float64
}

// AnalyzeStructure performs comprehensive market structure analysis using all available indicators
func (s *MarketStructureService) AnalyzeStructure(
	candles []model.Candle,
	trend string,
	indicators model.Indicators,
	srLevels model.SRLevels,
	patterns []model.CandlestickPattern,
) model.MarketStructure {
	if len(candles) < 20 {
		return s.getDefaultStructure()
	}

	// Basic structure analysis
	swingHighs := s.findSwingHighs(candles, 5)
	swingLows := s.findSwingLows(candles, 5)

	higherHigh := false
	higherLow := false
	lowerHigh := false
	lowerLow := false

	if len(swingHighs) >= 2 {
		if swingHighs[len(swingHighs)-1] > swingHighs[len(swingHighs)-2] {
			higherHigh = true
		} else {
			lowerHigh = true
		}
	}

	if len(swingLows) >= 2 {
		if swingLows[len(swingLows)-1] > swingLows[len(swingLows)-2] {
			higherLow = true
		} else {
			lowerLow = true
		}
	}

	structureBreak := false
	if trend == "上升" && lowerLow {
		structureBreak = true
	} else if trend == "下降" && higherHigh {
		structureBreak = true
	}

	riskLevel := s.calculateRiskLevel(trend, higherHigh, higherLow, lowerHigh, lowerLow, structureBreak)

	// Enhanced multi-indicator analysis
	currentPrice := candles[len(candles)-1].Close
	trendConfirmation := s.analyzeTrendConfirmation(candles, indicators, trend)
	volatilityProfile := s.analyzeVolatilityProfile(candles, indicators.ATR, currentPrice)
	keyLevelConfluence := s.analyzeKeyLevelConfluence(currentPrice, srLevels, indicators.Fibonacci, indicators.EMA)
	patternSignals := s.analyzePatternSignals(patterns)
	marketQuality := s.calculateMarketQuality(
		trendConfirmation,
		volatilityProfile,
		keyLevelConfluence,
		patternSignals,
		structureBreak,
		trend,
	)

	return model.MarketStructure{
		HigherHigh:         higherHigh,
		HigherLow:          higherLow,
		StructureBreak:     structureBreak,
		RiskLevel:          riskLevel,
		TrendConfirmation:  trendConfirmation,
		VolatilityProfile:  volatilityProfile,
		KeyLevelConfluence: keyLevelConfluence,
		PatternSignals:     patternSignals,
		MarketQuality:      marketQuality,
	}
}

// analyzeTrendConfirmation analyzes trend using EMA alignment and MACD
func (s *MarketStructureService) analyzeTrendConfirmation(
	candles []model.Candle,
	indicators model.Indicators,
	trend string,
) model.TrendConfirmation {
	currentPrice := candles[len(candles)-1].Close
	ema := indicators.EMA
	macd := indicators.MACD

	// EMA Alignment Analysis
	emaAlignment := "NEUTRAL"
	emaScore := 50.0

	if indicator.IsBullishAlignment(ema.EMA9, ema.EMA21, ema.EMA50, ema.EMA200) {
		emaAlignment = "BULLISH"
		emaScore = 85.0
	} else if indicator.IsBearishAlignment(ema.EMA9, ema.EMA21, ema.EMA50, ema.EMA200) {
		emaAlignment = "BEARISH"
		emaScore = 85.0
	} else {
		// Partial alignment
		if ema.EMA9 > ema.EMA21 && ema.EMA21 > ema.EMA50 {
			emaAlignment = "BULLISH"
			emaScore = 65.0
		} else if ema.EMA9 < ema.EMA21 && ema.EMA21 < ema.EMA50 {
			emaAlignment = "BEARISH"
			emaScore = 65.0
		}
	}

	// MACD Signal Analysis
	macdSignal := "NEUTRAL"
	macdScore := 50.0

	if macd.Histogram > 0 && macd.DIF > macd.DEA {
		macdSignal = "BULLISH"
		macdScore = 70.0 + math.Min(macd.Histogram*10, 30.0)
	} else if macd.Histogram < 0 && macd.DIF < macd.DEA {
		macdSignal = "BEARISH"
		macdScore = 70.0 + math.Min(math.Abs(macd.Histogram)*10, 30.0)
	}

	// Price vs EMA Analysis
	priceVsEMA := "NEUTRAL"
	priceScore := 50.0

	if currentPrice > ema.EMA9 && currentPrice > ema.EMA21 {
		priceVsEMA = "ABOVE_KEY_EMAS"
		priceScore = 75.0
	} else if currentPrice < ema.EMA9 && currentPrice < ema.EMA21 {
		priceVsEMA = "BELOW_KEY_EMAS"
		priceScore = 75.0
	} else if currentPrice > ema.EMA50 {
		priceVsEMA = "ABOVE_MEDIUM_EMA"
		priceScore = 60.0
	} else if currentPrice < ema.EMA50 {
		priceVsEMA = "BELOW_MEDIUM_EMA"
		priceScore = 60.0
	}

	// Calculate overall confirmation score
	confirmationScore := (emaScore*0.4 + macdScore*0.35 + priceScore*0.25)

	// Determine strength
	strength := "WEAK"
	if confirmationScore >= 75 {
		strength = "STRONG"
	} else if confirmationScore >= 60 {
		strength = "MODERATE"
	}

	return model.TrendConfirmation{
		EMAAlignment:      emaAlignment,
		MACDSignal:        macdSignal,
		PriceVsEMA:        priceVsEMA,
		ConfirmationScore: confirmationScore,
		Strength:          strength,
	}
}

// analyzeVolatilityProfile analyzes market volatility using ATR
func (s *MarketStructureService) analyzeVolatilityProfile(
	candles []model.Candle,
	atr model.ATRIndicator,
	currentPrice float64,
) model.VolatilityProfile {
	atrPercentage := (atr.Value / currentPrice) * 100

	// Determine volatility level
	volatilityLevel := "NORMAL"
	if atrPercentage > 5.0 {
		volatilityLevel = "HIGH"
	} else if atrPercentage < 2.0 {
		volatilityLevel = "LOW"
	}

	// Check if volatility is expanding
	isExpanding := false
	if len(candles) >= 28 {
		// Compare recent ATR with older ATR
		recentATR := indicator.CalculateATR(candles[len(candles)-14:], 14)
		olderATR := indicator.CalculateATR(candles[len(candles)-28:len(candles)-14], 14)
		if recentATR.GetCurrentATR() > olderATR.GetCurrentATR()*1.2 {
			isExpanding = true
		}
	}

	// Risk adjustment recommendation
	riskAdjustment := "NORMAL"
	if volatilityLevel == "HIGH" {
		riskAdjustment = "REDUCE_POSITION_SIZE"
	} else if volatilityLevel == "LOW" {
		riskAdjustment = "CONSIDER_WIDER_STOPS"
	}

	return model.VolatilityProfile{
		CurrentATR:      atr.Value,
		ATRPercentage:   atrPercentage,
		VolatilityLevel: volatilityLevel,
		IsExpanding:     isExpanding,
		RiskAdjustment:  riskAdjustment,
	}
}

// analyzeKeyLevelConfluence identifies confluence zones from multiple indicators
func (s *MarketStructureService) analyzeKeyLevelConfluence(
	currentPrice float64,
	srLevels model.SRLevels,
	fibonacci *model.FibonacciLevels,
	ema model.EMAIndicator,
) model.KeyLevelConfluence {
	// Collect all significant levels
	var allLevels []levelInfo

	// Add SR levels
	for _, level := range srLevels.Support {
		allLevels = append(allLevels, levelInfo{
			price:  level.Price,
			factor: "Support Level",
			weight: level.Strength,
		})
	}
	for _, level := range srLevels.Resistance {
		allLevels = append(allLevels, levelInfo{
			price:  level.Price,
			factor: "Resistance Level",
			weight: level.Strength,
		})
	}

	// Add Fibonacci levels
	if fibonacci != nil {
		for label, price := range fibonacci.Retracement {
			if label == "38.2%" || label == "50%" || label == "61.8%" {
				allLevels = append(allLevels, levelInfo{
					price:  price,
					factor: "Fibonacci " + label,
					weight: 0.8,
				})
			}
		}
	}

	// Add key EMAs
	emaLevels := []struct {
		price float64
		name  string
	}{
		{ema.EMA21, "EMA21"},
		{ema.EMA50, "EMA50"},
		{ema.EMA200, "EMA200"},
	}
	for _, emaLevel := range emaLevels {
		if emaLevel.price > 0 {
			allLevels = append(allLevels, levelInfo{
				price:  emaLevel.price,
				factor: emaLevel.name,
				weight: 0.7,
			})
		}
	}

	// Find confluence zones (levels within 0.5% of each other)
	confluenceZones := s.findConfluenceZones(allLevels, currentPrice, 0.005)

	// Find nearest support and resistance
	nearestSupport := s.findNearestLevel(allLevels, currentPrice, true)
	nearestResistance := s.findNearestLevel(allLevels, currentPrice, false)

	return model.KeyLevelConfluence{
		NearestSupport:    nearestSupport,
		NearestResistance: nearestResistance,
		ConfluenceZones:   confluenceZones,
	}
}

// findConfluenceZones identifies areas where multiple levels cluster
func (s *MarketStructureService) findConfluenceZones(
	levels []levelInfo,
	currentPrice float64,
	threshold float64,
) []model.ConfluenceZone {
	if len(levels) == 0 {
		return []model.ConfluenceZone{}
	}

	// Sort levels by price
	sort.Slice(levels, func(i, j int) bool {
		return levels[i].price < levels[j].price
	})

	var zones []model.ConfluenceZone
	i := 0

	for i < len(levels) {
		zoneStart := levels[i].price
		zoneEnd := levels[i].price
		var factors []string
		totalStrength := levels[i].weight

		factors = append(factors, levels[i].factor)

		// Find all levels within threshold
		j := i + 1
		for j < len(levels) {
			if math.Abs(levels[j].price-zoneStart)/currentPrice <= threshold {
				zoneEnd = levels[j].price
				factors = append(factors, levels[j].factor)
				totalStrength += levels[j].weight
				j++
			} else {
				break
			}
		}

		// Only create zone if multiple factors align
		if len(factors) >= 2 {
			significance := "MODERATE"
			if len(factors) >= 4 {
				significance = "CRITICAL"
			} else if len(factors) >= 3 {
				significance = "IMPORTANT"
			}

			zones = append(zones, model.ConfluenceZone{
				PriceRange:   [2]float64{zoneStart, zoneEnd},
				Factors:      factors,
				Strength:     math.Min(totalStrength*20, 100),
				Significance: significance,
			})
		}

		i = j
	}

	// Sort zones by strength
	sort.Slice(zones, func(i, j int) bool {
		return zones[i].Strength > zones[j].Strength
	})

	// Return top 5 zones
	if len(zones) > 5 {
		zones = zones[:5]
	}

	return zones
}

// findNearestLevel finds the nearest support or resistance level
func (s *MarketStructureService) findNearestLevel(
	levels []levelInfo,
	currentPrice float64,
	isSupport bool,
) *model.ConfluenceLevel {
	var bestLevel *model.ConfluenceLevel
	minDistance := math.MaxFloat64

	// Group nearby levels
	grouped := make(map[float64][]levelInfo)
	for _, level := range levels {
		if isSupport && level.price >= currentPrice {
			continue
		}
		if !isSupport && level.price <= currentPrice {
			continue
		}

		// Find if this level is close to any existing group
		found := false
		for groupPrice := range grouped {
			if math.Abs(level.price-groupPrice)/currentPrice < 0.005 {
				grouped[groupPrice] = append(grouped[groupPrice], level)
				found = true
				break
			}
		}
		if !found {
			grouped[level.price] = []levelInfo{level}
		}
	}

	// Find nearest group
	for groupPrice, groupLevels := range grouped {
		distance := math.Abs(groupPrice - currentPrice)
		if distance < minDistance {
			minDistance = distance

			var factors []string
			totalStrength := 0.0
			for _, l := range groupLevels {
				factors = append(factors, l.factor)
				totalStrength += l.weight
			}

			levelType := "RESISTANCE"
			if isSupport {
				levelType = "SUPPORT"
			}

			bestLevel = &model.ConfluenceLevel{
				Price:    groupPrice,
				Distance: (distance / currentPrice) * 100,
				Factors:  factors,
				Strength: math.Min(totalStrength*20, 100),
				Type:     levelType,
			}
		}
	}

	return bestLevel
}

// analyzePatternSignals aggregates candlestick pattern insights
func (s *MarketStructureService) analyzePatternSignals(patterns []model.CandlestickPattern) model.PatternSignals {
	if len(patterns) == 0 {
		return model.PatternSignals{
			RecentPatterns:     []string{},
			BullishCount:       0,
			BearishCount:       0,
			DominantSignal:     "NEUTRAL",
			PatternReliability: 0,
		}
	}

	var recentPatterns []string
	bullishCount := 0
	bearishCount := 0
	totalReliability := 0.0

	// Analyze recent patterns (last 5)
	count := len(patterns)
	if count > 5 {
		count = 5
	}

	for i := 0; i < count; i++ {
		pattern := patterns[len(patterns)-1-i]
		recentPatterns = append(recentPatterns, pattern.Pattern)

		if pattern.Direction == "看涨" {
			bullishCount++
		} else if pattern.Direction == "看跌" {
			bearishCount++
		}

		totalReliability += pattern.Reliability
	}

	// Determine dominant signal
	dominantSignal := "NEUTRAL"
	if bullishCount > bearishCount {
		dominantSignal = "BULLISH"
	} else if bearishCount > bullishCount {
		dominantSignal = "BEARISH"
	}

	patternReliability := 0.0
	if count > 0 {
		patternReliability = (totalReliability / float64(count)) * 100
	}

	return model.PatternSignals{
		RecentPatterns:     recentPatterns,
		BullishCount:       bullishCount,
		BearishCount:       bearishCount,
		DominantSignal:     dominantSignal,
		PatternReliability: patternReliability,
	}
}

// calculateMarketQuality provides overall market structure quality assessment
func (s *MarketStructureService) calculateMarketQuality(
	trendConfirmation model.TrendConfirmation,
	volatilityProfile model.VolatilityProfile,
	keyLevelConfluence model.KeyLevelConfluence,
	patternSignals model.PatternSignals,
	structureBreak bool,
	trend string,
) model.MarketQuality {
	scoreBreakdown := make(map[string]float64)

	// Trend Confirmation Score (30%)
	trendScore := trendConfirmation.ConfirmationScore
	scoreBreakdown["trend_confirmation"] = trendScore

	// Volatility Score (20%)
	volatilityScore := 50.0
	if volatilityProfile.VolatilityLevel == "NORMAL" {
		volatilityScore = 80.0
	} else if volatilityProfile.VolatilityLevel == "LOW" {
		volatilityScore = 60.0
	} else {
		volatilityScore = 40.0
	}
	scoreBreakdown["volatility"] = volatilityScore

	// Confluence Score (25%)
	confluenceScore := 50.0
	if keyLevelConfluence.NearestSupport != nil && keyLevelConfluence.NearestResistance != nil {
		avgStrength := (keyLevelConfluence.NearestSupport.Strength + keyLevelConfluence.NearestResistance.Strength) / 2
		confluenceScore = avgStrength
	}
	if len(keyLevelConfluence.ConfluenceZones) > 0 {
		confluenceScore = math.Min(confluenceScore+10, 100)
	}
	scoreBreakdown["key_levels"] = confluenceScore

	// Pattern Score (15%)
	patternScore := patternSignals.PatternReliability
	scoreBreakdown["patterns"] = patternScore

	// Structure Integrity Score (10%)
	structureScore := 80.0
	if structureBreak {
		structureScore = 30.0
	}
	scoreBreakdown["structure_integrity"] = structureScore

	// Calculate overall score
	overallScore := (trendScore*0.30 +
		volatilityScore*0.20 +
		confluenceScore*0.25 +
		patternScore*0.15 +
		structureScore*0.10)

	// Determine grade
	grade := "F"
	if overallScore >= 90 {
		grade = "A"
	} else if overallScore >= 80 {
		grade = "B"
	} else if overallScore >= 70 {
		grade = "C"
	} else if overallScore >= 60 {
		grade = "D"
	}

	// Determine trading condition
	tradingCondition := "POOR"
	if overallScore >= 85 {
		tradingCondition = "EXCELLENT"
	} else if overallScore >= 70 {
		tradingCondition = "GOOD"
	} else if overallScore >= 55 {
		tradingCondition = "FAIR"
	}

	// Identify strengths and weaknesses
	var strengths, weaknesses []string

	if trendScore >= 75 {
		strengths = append(strengths, "Strong trend confirmation")
	} else if trendScore < 50 {
		weaknesses = append(weaknesses, "Weak trend confirmation")
	}

	if volatilityProfile.VolatilityLevel == "NORMAL" {
		strengths = append(strengths, "Healthy volatility")
	} else if volatilityProfile.VolatilityLevel == "HIGH" {
		weaknesses = append(weaknesses, "High volatility increases risk")
	}

	if confluenceScore >= 70 {
		strengths = append(strengths, "Clear key levels identified")
	} else {
		weaknesses = append(weaknesses, "Unclear support/resistance")
	}

	if patternScore >= 70 {
		strengths = append(strengths, "Reliable pattern signals")
	}

	if structureBreak {
		weaknesses = append(weaknesses, "Market structure break detected")
	}

	// Generate recommendation
	recommendation := s.generateRecommendation(overallScore, trend, trendConfirmation, structureBreak)

	return model.MarketQuality{
		OverallScore:     overallScore,
		Grade:            grade,
		TradingCondition: tradingCondition,
		Strengths:        strengths,
		Weaknesses:       weaknesses,
		Recommendation:   recommendation,
		ScoreBreakdown:   scoreBreakdown,
	}
}

// generateRecommendation generates trading recommendation based on analysis
func (s *MarketStructureService) generateRecommendation(
	overallScore float64,
	trend string,
	trendConfirmation model.TrendConfirmation,
	structureBreak bool,
) string {
	if structureBreak {
		return "谨慎交易：市场结构已被破坏，等待新结构形成"
	}

	if overallScore >= 85 {
		if trend == "上升" && trendConfirmation.EMAAlignment == "BULLISH" {
			return "优质做多机会：多项指标确认上升趋势"
		} else if trend == "下降" && trendConfirmation.EMAAlignment == "BEARISH" {
			return "优质做空机会：多项指标确认下降趋势"
		}
		return "市场结构良好，寻找符合趋势的交易机会"
	} else if overallScore >= 70 {
		return "可交易：市场条件良好，注意风险管理"
	} else if overallScore >= 55 {
		return "谨慎交易：市场信号混合，减小仓位"
	} else {
		return "观望为主：市场条件不佳，等待更好机会"
	}
}

// Helper methods from original implementation

func (s *MarketStructureService) findSwingHighs(candles []model.Candle, lookback int) []float64 {
	swingHighs := make([]float64, 0)

	for i := lookback; i < len(candles)-lookback; i++ {
		isSwingHigh := true

		for j := i - lookback; j <= i+lookback; j++ {
			if j != i && candles[j].High >= candles[i].High {
				isSwingHigh = false
				break
			}
		}

		if isSwingHigh {
			swingHighs = append(swingHighs, candles[i].High)
		}
	}

	return swingHighs
}

func (s *MarketStructureService) findSwingLows(candles []model.Candle, lookback int) []float64 {
	swingLows := make([]float64, 0)

	for i := lookback; i < len(candles)-lookback; i++ {
		isSwingLow := true

		for j := i - lookback; j <= i+lookback; j++ {
			if j != i && candles[j].Low <= candles[i].Low {
				isSwingLow = false
				break
			}
		}

		if isSwingLow {
			swingLows = append(swingLows, candles[i].Low)
		}
	}

	return swingLows
}

func (s *MarketStructureService) calculateRiskLevel(trend string, hh, hl, lh, ll, structureBreak bool) string {
	if structureBreak {
		return "高"
	}

	if trend == "上升" {
		if hh && hl {
			return "低" // Healthy uptrend
		}
		return "中"
	} else if trend == "下降" {
		if lh && ll {
			return "低" // Healthy downtrend (low risk for shorts)
		}
		return "中"
	}

	return "中" // Consolidation
}

func (s *MarketStructureService) getDefaultStructure() model.MarketStructure {
	return model.MarketStructure{
		HigherHigh:     false,
		HigherLow:      false,
		StructureBreak: false,
		RiskLevel:      "中",
		TrendConfirmation: model.TrendConfirmation{
			EMAAlignment:      "NEUTRAL",
			MACDSignal:        "NEUTRAL",
			PriceVsEMA:        "NEUTRAL",
			ConfirmationScore: 50,
			Strength:          "WEAK",
		},
		VolatilityProfile: model.VolatilityProfile{
			CurrentATR:      0,
			ATRPercentage:   0,
			VolatilityLevel: "NORMAL",
			IsExpanding:     false,
			RiskAdjustment:  "NORMAL",
		},
		KeyLevelConfluence: model.KeyLevelConfluence{
			NearestSupport:    nil,
			NearestResistance: nil,
			ConfluenceZones:   []model.ConfluenceZone{},
		},
		PatternSignals: model.PatternSignals{
			RecentPatterns:     []string{},
			BullishCount:       0,
			BearishCount:       0,
			DominantSignal:     "NEUTRAL",
			PatternReliability: 0,
		},
		MarketQuality: model.MarketQuality{
			OverallScore:     50,
			Grade:            "D",
			TradingCondition: "POOR",
			Strengths:        []string{},
			Weaknesses:       []string{"数据不足"},
			Recommendation:   "等待更多数据",
			ScoreBreakdown:   make(map[string]float64),
		},
	}
}
