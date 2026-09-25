package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kudaompq/ai_trending/backend/internal/service"
)

// KlineHandler handles K-line related requests
type KlineHandler struct {
	klineService *service.KlineService
	validator    *service.SymbolValidator
}

// NewKlineHandler creates a new K-line handler
func NewKlineHandler(validator *service.SymbolValidator) *KlineHandler {
	return &KlineHandler{
		klineService: service.NewKlineService(),
		validator:    validator,
	}
}

// GetKline handles GET /api/kline
func (h *KlineHandler) GetKline(c *gin.Context) {
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

	data, err := h.klineService.GetKlineData(symbol, interval, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, data)
}
