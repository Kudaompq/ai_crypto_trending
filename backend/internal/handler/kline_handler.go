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

// NewKlineHandlerWithService creates a handler with an injected K-line service.
func NewKlineHandlerWithService(klineService *service.KlineService, validator *service.SymbolValidator) *KlineHandler {
	return &KlineHandler{klineService: klineService, validator: validator}
}

// GetKline handles GET /api/kline
func (h *KlineHandler) GetKline(c *gin.Context) {
	symbol, ok := validatedSymbol(c, h.validator)
	if !ok {
		return
	}
	interval := c.DefaultQuery("interval", "1d")
	if !service.IsSupportedKlineInterval(interval) {
		c.JSON(http.StatusBadRequest, gin.H{"code": "unsupported_interval", "error": "不支持的时间周期"})
		return
	}
	limitStr := c.DefaultQuery("limit", "100")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 500 {
		limit = 100
	}
	var endTime *int64
	if rawEndTime, present := c.GetQuery("endTime"); present {
		parsed, err := strconv.ParseInt(rawEndTime, 10, 64)
		if err != nil || parsed <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_end_time", "error": "endTime 必须是正整数毫秒时间戳"})
			return
		}
		endTime = &parsed
	}

	data, err := h.klineService.GetKlineData(symbol, interval, limit, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, data)
}
