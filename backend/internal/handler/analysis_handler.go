package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kudaompq/ai_trending/backend/internal/service"
)

// AnalysisHandler handles analysis related requests
type AnalysisHandler struct {
	analysisService *service.AnalysisService
	validator       *service.SymbolValidator
}

// NewAnalysisHandler creates a new analysis handler
func NewAnalysisHandler(validator *service.SymbolValidator) *AnalysisHandler {
	return &AnalysisHandler{
		analysisService: service.NewAnalysisService(),
		validator:       validator,
	}
}

// GetAnalysis handles GET /api/analysis
func (h *AnalysisHandler) GetAnalysis(c *gin.Context) {
	symbol, ok := validatedSymbol(c, h.validator)
	if !ok {
		return
	}
	interval := c.DefaultQuery("interval", "1d")
	limitStr := c.DefaultQuery("limit", "100")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 500 {
		limit = 100
	}

	result, err := h.analysisService.PerformAnalysis(symbol, interval, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}
