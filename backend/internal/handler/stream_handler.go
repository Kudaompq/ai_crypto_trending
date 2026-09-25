package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kudaompq/ai_trending/backend/internal/service"
)

var allowedIntervals = map[string]struct{}{
	"1m": {}, "5m": {}, "15m": {}, "1h": {}, "4h": {}, "1d": {},
}

type StreamHandler struct {
	streamService *service.MarketStreamService
	validator     *service.SymbolValidator
}

func NewStreamHandler(streamService *service.MarketStreamService, validator *service.SymbolValidator) *StreamHandler {
	return &StreamHandler{streamService: streamService, validator: validator}
}

// GetMarketStream handles GET /api/stream using Server-Sent Events.
func (h *StreamHandler) GetMarketStream(c *gin.Context) {
	symbol, ok := validatedSymbol(c, h.validator)
	if !ok {
		return
	}
	interval := c.DefaultQuery("interval", "1d")

	if _, ok := allowedIntervals[interval]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"code": "unsupported_interval", "error": "不支持的时间周期"})
		return
	}

	events, unsubscribe := h.streamService.Subscribe(symbol, interval)
	defer unsubscribe()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	c.SSEvent("ready", gin.H{"symbol": symbol, "interval": interval})
	c.Writer.Flush()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			if event.Type == "status" {
				c.SSEvent("status", gin.H{"symbol": event.Symbol, "interval": event.Interval, "state": event.State, "message": event.Message})
			} else {
				c.SSEvent("kline", event)
			}
			c.Writer.Flush()
		}
	}
}
