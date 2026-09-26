package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kudaompq/ai_trending/backend/internal/repository"
	"github.com/kudaompq/ai_trending/backend/internal/service"
)

type watchlistUseCase interface {
	Get(context.Context) (repository.WatchlistSnapshot, error)
	ImportLegacy(context.Context, []string) (repository.WatchlistSnapshot, error)
	Add(context.Context, string) (repository.WatchlistSnapshot, error)
	Remove(context.Context, string) (repository.WatchlistSnapshot, error)
	Reorder(context.Context, int64, []string) (repository.WatchlistSnapshot, error)
}

type watchlistMutationRequest struct {
	Symbol string `json:"symbol"`
}

type watchlistImportRequest struct {
	Symbols []string `json:"symbols"`
}

type watchlistReorderRequest struct {
	Revision int64    `json:"revision"`
	Symbols  []string `json:"symbols"`
}

func RegisterWatchlistRoutes(routes *gin.RouterGroup, svc watchlistUseCase) {
	routes.GET("/watchlist", func(c *gin.Context) {
		snapshot, err := svc.Get(c.Request.Context())
		if err != nil {
			writeWatchlistError(c, snapshot, err)
			return
		}
		c.JSON(http.StatusOK, snapshot)
	})
	routes.POST("/watchlist/import-legacy", func(c *gin.Context) {
		var request watchlistImportRequest
		if err := c.ShouldBindJSON(&request); err != nil || request.Symbols == nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_request", "error": "Watchlist 数据格式错误"})
			return
		}
		snapshot, err := svc.ImportLegacy(c.Request.Context(), request.Symbols)
		if err != nil {
			writeWatchlistError(c, snapshot, err)
			return
		}
		c.JSON(http.StatusOK, snapshot)
	})
	routes.POST("/watchlist/symbols", func(c *gin.Context) {
		var request watchlistMutationRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_request", "error": "交易对数据格式错误"})
			return
		}
		snapshot, err := svc.Add(c.Request.Context(), request.Symbol)
		if err != nil {
			writeWatchlistError(c, snapshot, err)
			return
		}
		c.JSON(http.StatusOK, snapshot)
	})
	routes.DELETE("/watchlist/symbols/:symbol", func(c *gin.Context) {
		snapshot, err := svc.Remove(c.Request.Context(), c.Param("symbol"))
		if err != nil {
			writeWatchlistError(c, snapshot, err)
			return
		}
		c.JSON(http.StatusOK, snapshot)
	})
	routes.PUT("/watchlist/order", func(c *gin.Context) {
		var request watchlistReorderRequest
		if err := c.ShouldBindJSON(&request); err != nil || request.Revision < 1 || request.Symbols == nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_request", "error": "Watchlist 顺序数据格式错误"})
			return
		}
		snapshot, err := svc.Reorder(c.Request.Context(), request.Revision, request.Symbols)
		if err != nil {
			writeWatchlistError(c, snapshot, err)
			return
		}
		c.JSON(http.StatusOK, snapshot)
	})
}

func writeWatchlistError(c *gin.Context, snapshot repository.WatchlistSnapshot, err error) {
	var symbolErr *service.SymbolError
	if errors.As(err, &symbolErr) {
		c.JSON(symbolErr.Status, gin.H{"code": symbolErr.Code, "error": symbolErr.Message})
		return
	}
	switch {
	case errors.Is(err, repository.ErrWatchlistConflict):
		c.JSON(http.StatusConflict, gin.H{
			"code": "watchlist_conflict", "error": "Watchlist 已被其他客户端修改，请先同步最新列表",
			"symbols": snapshot.Symbols, "revision": snapshot.Revision,
			"legacy_import_pending": snapshot.LegacyImportPending,
		})
	case errors.Is(err, repository.ErrWatchlistDuplicate):
		c.JSON(http.StatusConflict, gin.H{"code": "duplicate_symbol", "error": "该交易对已在列表中"})
	case errors.Is(err, repository.ErrWatchlistLastSymbol):
		c.JSON(http.StatusConflict, gin.H{"code": "last_symbol", "error": "至少保留一个自选交易对"})
	case errors.Is(err, repository.ErrWatchlistSymbolNotFound):
		c.JSON(http.StatusNotFound, gin.H{"code": "symbol_not_found", "error": "该交易对不在 Watchlist 中"})
	case errors.Is(err, repository.ErrWatchlistInvalidOrder):
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_watchlist_order", "error": "排序必须包含当前所有交易对且不能重复"})
	case errors.Is(err, repository.ErrWatchlistLimit):
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"code": "watchlist_limit", "error": "Watchlist 最多支持 100 个交易对"})
	default:
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": "watchlist_unavailable", "error": "Watchlist 保存失败，请稍后重试"})
	}
}
