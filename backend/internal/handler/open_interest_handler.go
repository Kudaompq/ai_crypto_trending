package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kudaompq/ai_trending/backend/internal/model"
	"github.com/kudaompq/ai_trending/backend/internal/service"
)

type OpenInterestHandler struct {
	service interface {
		GetOpenInterestData(symbol, interval string, limit int, startTime, endTime *int64) (*model.OpenInterestData, error)
	}
	validator *service.SymbolValidator
}

func NewOpenInterestHandler(validator *service.SymbolValidator) *OpenInterestHandler {
	return NewOpenInterestHandlerWithService(service.NewOpenInterestService(), validator)
}

func NewOpenInterestHandlerWithService(openInterestService interface {
	GetOpenInterestData(symbol, interval string, limit int, startTime, endTime *int64) (*model.OpenInterestData, error)
}, validator *service.SymbolValidator) *OpenInterestHandler {
	return &OpenInterestHandler{service: openInterestService, validator: validator}
}

func (h *OpenInterestHandler) GetOpenInterest(c *gin.Context) {
	symbol, ok := validatedSymbol(c, h.validator)
	if !ok {
		return
	}
	interval := c.DefaultQuery("interval", "1d")
	limit := 100
	if rawLimit, present := c.GetQuery("limit"); present {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 || parsed > 500 {
			c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_limit", "error": "limit 必须是 1 到 500 之间的整数"})
			return
		}
		limit = parsed
	}
	startTime, ok := parseOpenInterestTime(c, "startTime")
	if !ok {
		return
	}
	endTime, ok := parseOpenInterestTime(c, "endTime")
	if !ok {
		return
	}
	if startTime != nil && endTime != nil && *startTime > *endTime {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_time_range", "error": "startTime 不能晚于 endTime"})
		return
	}

	data, err := h.service.GetOpenInterestData(symbol, interval, limit, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "open_interest_source_unavailable", "error": "OI 数据源暂不可用"})
		return
	}
	c.JSON(http.StatusOK, data)
}

func parseOpenInterestTime(c *gin.Context, key string) (*int64, bool) {
	raw, present := c.GetQuery(key)
	if !present {
		return nil, true
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || parsed <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_" + key, "error": key + " 必须是正整数毫秒时间戳"})
		return nil, false
	}
	return &parsed, true
}
