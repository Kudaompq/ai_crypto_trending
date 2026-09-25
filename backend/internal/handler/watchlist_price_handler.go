package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kudaompq/ai_trending/backend/internal/service"
)

// WatchlistPriceSource provides REST snapshots and a shared filtered price stream.
type WatchlistPriceSource interface {
	Snapshot(context.Context, []string) ([]service.PriceQuote, error)
	Subscribe([]string) (<-chan service.PriceEvent, func())
}

type WatchlistPriceHandler struct {
	prices    WatchlistPriceSource
	validator *service.SymbolValidator
}

func NewWatchlistPriceHandler(prices WatchlistPriceSource, validator *service.SymbolValidator) *WatchlistPriceHandler {
	return &WatchlistPriceHandler{prices: prices, validator: validator}
}

func (h *WatchlistPriceHandler) GetPrices(c *gin.Context) {
	symbols, unavailableSymbols, ok := h.validatedSymbols(c)
	if !ok {
		return
	}
	quotes := []service.PriceQuote{}
	if len(symbols) > 0 {
		var err error
		quotes, err = h.prices.Snapshot(c.Request.Context(), symbols)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"code": "price_snapshot_failed", "error": "获取实时价格失败，请稍后重试"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"prices": quotes, "unavailable_symbols": unavailableSymbols})
}

func (h *WatchlistPriceHandler) GetStream(c *gin.Context) {
	symbols, unavailableSymbols, ok := h.validatedSymbols(c)
	if !ok {
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	c.SSEvent("ready", gin.H{"symbols": symbols, "unavailable_symbols": unavailableSymbols})
	c.Writer.Flush()
	if len(symbols) == 0 {
		return
	}
	events, unsubscribe := h.prices.Subscribe(symbols)
	defer unsubscribe()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case event, open := <-events:
			if !open {
				return
			}
			if event.Type != "status" && event.Type != "price" {
				continue
			}
			c.SSEvent(event.Type, event)
			c.Writer.Flush()
		}
	}
}

func (h *WatchlistPriceHandler) validatedSymbols(c *gin.Context) ([]string, []string, bool) {
	parts := strings.Split(c.Query("symbols"), ",")
	if len(parts) == 0 || (len(parts) == 1 && strings.TrimSpace(parts[0]) == "") {
		c.JSON(http.StatusBadRequest, gin.H{"code": "symbols_required", "error": "请至少提供一个交易对"})
		return nil, nil, false
	}
	if len(parts) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"code": "too_many_symbols", "error": "一次最多查询 100 个交易对"})
		return nil, nil, false
	}
	symbols := make([]string, 0, len(parts))
	unavailable := make([]string, 0)
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		symbol, validationErr := h.validator.Validate(c.Request.Context(), part)
		if validationErr != nil {
			if validationErr.Status == http.StatusNotFound {
				unavailableSymbol := strings.ToUpper(strings.TrimSpace(part))
				if _, exists := seen[unavailableSymbol]; !exists {
					seen[unavailableSymbol] = struct{}{}
					unavailable = append(unavailable, unavailableSymbol)
				}
				continue
			}
			c.JSON(validationErr.Status, gin.H{"code": validationErr.Code, "error": validationErr.Message})
			return nil, nil, false
		}
		if _, exists := seen[symbol]; exists {
			continue
		}
		seen[symbol] = struct{}{}
		symbols = append(symbols, symbol)
	}
	return symbols, unavailable, true
}
