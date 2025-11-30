package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kudaompq/ai_trending/backend/internal/repository"
	"github.com/kudaompq/ai_trending/backend/internal/service"
)

// OpportunityHandler handles opportunity-related requests
type OpportunityHandler struct {
	binanceRepo        *repository.BinanceRepository
	analysisService    *service.AnalysisService
	opportunityService *service.OpportunityService
}

// NewOpportunityHandler creates a new opportunity handler
func NewOpportunityHandler() *OpportunityHandler {
	return &OpportunityHandler{
		binanceRepo:        repository.NewBinanceRepository(),
		analysisService:    service.NewAnalysisService(),
		opportunityService: service.NewOpportunityService(),
	}
}

// GetOpportunities handles GET /api/opportunities
func (h *OpportunityHandler) GetOpportunities(c *gin.Context) {
	// Parse parameters
	symbol := c.DefaultQuery("symbol", "ETHUSDT")
	interval := c.DefaultQuery("interval", "1h")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	minRR, _ := strconv.ParseFloat(c.DefaultQuery("min_rr", "3.0"), 64)

	// Get candles
	candles, err := h.binanceRepo.GetKlines(symbol, interval, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch kline data: " + err.Error(),
		})
		return
	}

	// Perform analysis
	analysis, err := h.analysisService.PerformAnalysis(symbol, interval, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to perform analysis: " + err.Error(),
		})
		return
	}

	// Detect opportunities
	opportunities := h.opportunityService.DetectOpportunities(candles, analysis, minRR)

	// Calculate summary
	totalCount := len(opportunities)
	avgRR := 0.0
	highConfCount := 0

	for _, opp := range opportunities {
		avgRR += opp.RiskReward.Ratio
		if opp.Confidence.Level == "HIGH" {
			highConfCount++
		}
	}

	if totalCount > 0 {
		avgRR = avgRR / float64(totalCount)
	}

	// Build response
	c.JSON(http.StatusOK, gin.H{
		"opportunities": opportunities,
		"summary": gin.H{
			"total_opportunities":   totalCount,
			"avg_risk_reward":       avgRR,
			"high_confidence_count": highConfCount,
		},
	})
}
