package service

import (
	"fmt"
	"math"
	"time"

	"github.com/kudaompq/ai_trending/backend/internal/model"
	"github.com/kudaompq/ai_trending/backend/internal/repository"
)

// OpportunityService detects trading opportunities
type OpportunityService struct {
	repository *repository.OpportunityRepository
}

// NewOpportunityService creates a new opportunity service
func NewOpportunityService() *OpportunityService {
	return &OpportunityService{
		repository: repository.NewOpportunityRepository(),
	}
}

// DetectOpportunities detects trading opportunities based on analysis
func (s *OpportunityService) DetectOpportunities(
	candles []model.Candle,
	analysis *model.AnalysisResult,
	minRiskReward float64,
) []model.TradingOpportunity {
	// First, update expired opportunities
	s.repository.UpdateExpiredOpportunities()

	// Get existing active opportunities for this symbol
	existingOpps, _ := s.repository.FindBySymbol(analysis.Symbol, "ACTIVE")

	// Detect new opportunities
	newlyDetected := []model.TradingOpportunity{}

	// Try support bounce strategy
	if opp := s.detectSupportBounce(candles, analysis); opp != nil {
		if opp.RiskReward.Ratio >= minRiskReward {
			// Save to database
			s.repository.Save(opp)
			newlyDetected = append(newlyDetected, *opp)
		}
	}

	// Try breakout retest strategy
	if opp := s.detectBreakoutRetest(candles, analysis); opp != nil {
		if opp.RiskReward.Ratio >= minRiskReward {
			s.repository.Save(opp)
			newlyDetected = append(newlyDetected, *opp)
		}
	}

	// Try trend continuation strategy
	if opp := s.detectTrendContinuation(candles, analysis); opp != nil {
		if opp.RiskReward.Ratio >= minRiskReward {
			s.repository.Save(opp)
			newlyDetected = append(newlyDetected, *opp)
		}
	}

	// Combine existing and newly detected (deduplicate by ID)
	allOpportunities := make(map[string]model.TradingOpportunity)

	// Add existing
	for _, opp := range existingOpps {
		allOpportunities[opp.ID] = opp
	}

	// Add/update with newly detected
	for _, opp := range newlyDetected {
		allOpportunities[opp.ID] = opp
	}

	// Convert map to slice
	result := make([]model.TradingOpportunity, 0, len(allOpportunities))
	for _, opp := range allOpportunities {
		result = append(result, opp)
	}

	return result
}

// detectSupportBounce detects support bounce opportunities
func (s *OpportunityService) detectSupportBounce(
	candles []model.Candle,
	analysis *model.AnalysisResult,
) *model.TradingOpportunity {
	if len(candles) < 50 || len(analysis.SRLevels.Support) == 0 {
		return nil
	}

	currentPrice := candles[len(candles)-1].Close
	atr := analysis.Indicators.ATR.Value

	// Find strongest support near current price
	var strongestSupport *model.SRLevel
	minDistance := math.MaxFloat64

	for i := range analysis.SRLevels.Support {
		support := &analysis.SRLevels.Support[i]
		distance := math.Abs(currentPrice - support.Price)
		distancePct := distance / currentPrice * 100

		// Support should be within 1% below current price and have good strength
		if support.Price < currentPrice && distancePct < 1.0 && support.Strength > 0.7 {
			if distance < minDistance {
				minDistance = distance
				strongestSupport = support
			}
		}
	}

	if strongestSupport == nil {
		return nil
	}

	// Check for bullish candlestick pattern
	hasBullishPattern := false
	var patternName string
	for _, pattern := range analysis.CandlestickPatterns {
		if pattern.Direction == "看涨" && pattern.Reliability > 0.7 {
			hasBullishPattern = true
			patternName = pattern.Pattern
			break
		}
	}

	if !hasBullishPattern {
		return nil
	}

	// Calculate entry, stop-loss, and targets
	supportPrice := strongestSupport.Price
	entryPrice := supportPrice * 1.002 // 0.2% above support

	// Stop-loss: 1.5% below support or 1x ATR
	stopLossDistance := math.Max(supportPrice*0.015, atr)
	stopLossPrice := supportPrice - stopLossDistance

	// Find resistance targets
	targets := s.findResistanceTargets(currentPrice, analysis.SRLevels.Resistance, analysis.Indicators.Fibonacci)

	if len(targets) == 0 {
		return nil
	}

	// Calculate risk-reward
	riskAmount := entryPrice - stopLossPrice
	rewardAmount := targets[0].Price - entryPrice
	rrRatio := rewardAmount / riskAmount

	if rrRatio < 2.0 {
		return nil
	}

	// Build reasons
	reasons := []string{
		fmt.Sprintf("Strong support at $%.2f (strength %.2f)", supportPrice, strongestSupport.Strength),
		fmt.Sprintf("%s pattern (reliability %.1f)", patternName, 0.8),
	}

	// Check for EMA support
	if analysis.Indicators.EMA.EMA50 > 0 && math.Abs(analysis.Indicators.EMA.EMA50-supportPrice) < supportPrice*0.01 {
		reasons = append(reasons, fmt.Sprintf("EMA(50) support at $%.2f", analysis.Indicators.EMA.EMA50))
	}

	// Check for Fibonacci support
	if analysis.Indicators.Fibonacci != nil {
		for level, price := range analysis.Indicators.Fibonacci.Retracement {
			if math.Abs(price-supportPrice) < supportPrice*0.01 {
				reasons = append(reasons, fmt.Sprintf("Fibonacci %s retracement at $%.2f", level, price))
				break
			}
		}
	}

	// Calculate confidence score
	confidence := s.calculateConfidence(reasons, hasBullishPattern, strongestSupport.Strength, rrRatio)

	// Create opportunity
	opportunity := &model.TradingOpportunity{
		ID:        fmt.Sprintf("opp_%d", time.Now().Unix()),
		Symbol:    analysis.Symbol,
		Type:      "LONG",
		Strategy:  "SUPPORT_BOUNCE",
		Timestamp: analysis.Timestamp,
		Entry: model.EntryPoint{
			Price:   entryPrice,
			Reasons: reasons,
		},
		StopLoss: model.StopLossInfo{
			Price:       stopLossPrice,
			DistancePct: (entryPrice - stopLossPrice) / entryPrice * 100,
			Method:      "TECHNICAL_LEVEL",
		},
		TakeProfit: targets,
		RiskReward: model.RiskRewardInfo{
			Ratio:        rrRatio,
			RiskAmount:   riskAmount,
			RewardAmount: rewardAmount,
			RiskPct:      (entryPrice - stopLossPrice) / entryPrice * 100,
			RewardPct:    (targets[0].Price - entryPrice) / entryPrice * 100,
		},
		Confidence: confidence,
		Validity: model.ValidityInfo{
			ExpiresAt: time.Now().Add(4*time.Hour).Unix() * 1000,
			Status:    "ACTIVE",
		},
	}

	return opportunity
}

// detectBreakoutRetest detects breakout retest opportunities
func (s *OpportunityService) detectBreakoutRetest(
	candles []model.Candle,
	analysis *model.AnalysisResult,
) *model.TradingOpportunity {
	// TODO: Implement breakout retest strategy
	return nil
}

// detectTrendContinuation detects trend continuation opportunities
func (s *OpportunityService) detectTrendContinuation(
	candles []model.Candle,
	analysis *model.AnalysisResult,
) *model.TradingOpportunity {
	// TODO: Implement trend continuation strategy
	return nil
}

// findResistanceTargets finds resistance levels for take-profit targets
func (s *OpportunityService) findResistanceTargets(
	currentPrice float64,
	resistanceLevels []model.SRLevel,
	fibonacci *model.FibonacciLevels,
) []model.TakeProfitLevel {
	targets := []model.TakeProfitLevel{}

	// Find resistance levels above current price
	validResistances := []model.SRLevel{}
	for _, r := range resistanceLevels {
		if r.Price > currentPrice && r.Strength > 0.6 {
			validResistances = append(validResistances, r)
		}
	}

	if len(validResistances) == 0 {
		return targets
	}

	// Sort by price (ascending)
	for i := 0; i < len(validResistances)-1; i++ {
		for j := i + 1; j < len(validResistances); j++ {
			if validResistances[i].Price > validResistances[j].Price {
				validResistances[i], validResistances[j] = validResistances[j], validResistances[i]
			}
		}
	}

	// Target 1: First resistance (50% position close)
	if len(validResistances) > 0 {
		targets = append(targets, model.TakeProfitLevel{
			Level:            1,
			Price:            validResistances[0].Price,
			DistancePct:      (validResistances[0].Price - currentPrice) / currentPrice * 100,
			Target:           "SR Level resistance",
			PositionClosePct: 50,
		})
	}

	// Target 2: Second resistance or Fibonacci extension (30% position close)
	if len(validResistances) > 1 {
		targets = append(targets, model.TakeProfitLevel{
			Level:            2,
			Price:            validResistances[1].Price,
			DistancePct:      (validResistances[1].Price - currentPrice) / currentPrice * 100,
			Target:           "SR Level resistance",
			PositionClosePct: 30,
		})
	} else if fibonacci != nil && fibonacci.Extension["1.618"] > currentPrice {
		targets = append(targets, model.TakeProfitLevel{
			Level:            2,
			Price:            fibonacci.Extension["1.618"],
			DistancePct:      (fibonacci.Extension["1.618"] - currentPrice) / currentPrice * 100,
			Target:           "Fibonacci 1.618 extension",
			PositionClosePct: 30,
		})
	}

	return targets
}

// calculateConfidence calculates confidence score for an opportunity
func (s *OpportunityService) calculateConfidence(
	reasons []string,
	hasBullishPattern bool,
	supportStrength float64,
	rrRatio float64,
) model.ConfidenceInfo {
	score := 50 // Base score

	// Add points for multiple support convergence
	if len(reasons) >= 3 {
		score += 15
	} else if len(reasons) >= 2 {
		score += 10
	}

	// Add points for bullish pattern
	if hasBullishPattern {
		score += 15
	}

	// Add points for strong support
	if supportStrength > 0.8 {
		score += 10
	} else if supportStrength > 0.7 {
		score += 5
	}

	// Add points for high R:R ratio
	if rrRatio >= 5.0 {
		score += 10
	} else if rrRatio >= 4.0 {
		score += 7
	} else if rrRatio >= 3.0 {
		score += 5
	}

	// Determine level
	level := "LOW"
	if score >= 80 {
		level = "HIGH"
	} else if score >= 60 {
		level = "MEDIUM"
	}

	factors := []string{}
	if len(reasons) >= 3 {
		factors = append(factors, "Multiple support convergence")
	}
	if hasBullishPattern {
		factors = append(factors, "Strong bullish pattern")
	}
	if supportStrength > 0.8 {
		factors = append(factors, "High support strength")
	}
	if rrRatio >= 4.0 {
		factors = append(factors, "Excellent risk-reward ratio")
	}

	return model.ConfidenceInfo{
		Score:   score,
		Level:   level,
		Factors: factors,
	}
}
