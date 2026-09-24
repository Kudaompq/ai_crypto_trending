package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kudaompq/ai_trending/backend/internal/service"
)

var allowedSymbols = map[string]struct{}{
	"BTCUSDT": {}, "ETHUSDT": {}, "BNBUSDT": {}, "SOLUSDT": {},
	"XRPUSDT": {}, "ADAUSDT": {}, "DOGEUSDT": {}, "MATICUSDT": {},
}

var allowedIntervals = map[string]struct{}{
	"1m": {}, "5m": {}, "15m": {}, "1h": {}, "4h": {}, "1d": {},
}

type StreamHandler struct {
	streamService *service.MarketStreamService
}

func NewStreamHandler(streamService *service.MarketStreamService) *StreamHandler {
	return &StreamHandler{streamService: streamService}
}

// GetMarketStream handles GET /api/stream using Server-Sent Events.
func (h *StreamHandler) GetMarketStream(c *gin.Context) {
	symbol := strings.ToUpper(c.DefaultQuery("symbol", "ETHUSDT"))
	interval := c.DefaultQuery("interval", "1d")

	if _, ok := allowedSymbols[symbol]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported symbol"})
		return
	}
	if _, ok := allowedIntervals[interval]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported interval"})
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
			c.SSEvent("kline", event)
			c.Writer.Flush()
		}
	}
}
