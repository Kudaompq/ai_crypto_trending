package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kudaompq/ai_trending/backend/internal/service"
)

type MarketAnalysisChatHandler struct {
	validator *service.SymbolValidator
	chat      *service.MarketAnalysisChatService
}

func NewMarketAnalysisChatHandler(validator *service.SymbolValidator, chat *service.MarketAnalysisChatService) *MarketAnalysisChatHandler {
	return &MarketAnalysisChatHandler{validator: validator, chat: chat}
}

func (h *MarketAnalysisChatHandler) PostMessage(c *gin.Context) {
	if h.validator == nil || h.chat == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "ai_provider_unavailable", "error": "AI 行情分析服务暂不可用"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 512<<10)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	var request service.MarketAnalysisChatRequest
	if err := decoder.Decode(&request); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"code": "request_too_large", "error": "请求内容过长"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_request", "error": "对话请求格式错误"})
		return
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_request", "error": "对话请求格式错误"})
		return
	}

	symbol, symbolErr := h.validator.Validate(c.Request.Context(), request.Symbol)
	if symbolErr != nil {
		c.JSON(symbolErr.Status, gin.H{"code": symbolErr.Code, "error": symbolErr.Message})
		return
	}
	request.Symbol = symbol
	if !service.IsSupportedKlineInterval(strings.TrimSpace(request.Interval)) {
		c.JSON(http.StatusBadRequest, gin.H{"code": "unsupported_interval", "error": "不支持的时间周期"})
		return
	}
	response, err := h.chat.Chat(c.Request.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidChatRequest):
			c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_request", "error": "对话内容无效或超出长度限制"})
		case errors.Is(err, service.ErrChatProviderNotConfigured):
			c.JSON(http.StatusServiceUnavailable, gin.H{"code": "ai_provider_unconfigured", "error": "AI 行情分析服务尚未配置"})
		case errors.Is(err, service.ErrMarketContextUnavailable):
			c.JSON(http.StatusServiceUnavailable, gin.H{"code": "market_context_unavailable", "error": "当前行情上下文暂不可用，请稍后重试"})
		case errors.Is(err, service.ErrChatProviderFailed):
			c.JSON(http.StatusBadGateway, gin.H{"code": "ai_provider_failed", "error": "AI 行情分析暂时失败，请重试"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"code": "ai_analysis_failed", "error": "AI 行情分析暂时失败，请重试"})
		}
		return
	}
	c.JSON(http.StatusOK, response)
}
