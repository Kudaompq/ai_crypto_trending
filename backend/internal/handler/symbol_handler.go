package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/kudaompq/ai_trending/backend/internal/service"
)

func validatedSymbol(c *gin.Context, validator *service.SymbolValidator) (string, bool) {
	symbol, err := validator.Validate(c.Request.Context(), c.DefaultQuery("symbol", "ETHUSDT"))
	if err != nil {
		c.JSON(err.Status, gin.H{"code": err.Code, "error": err.Message})
		return "", false
	}
	return symbol, true
}

func ValidateSymbol(validator *service.SymbolValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		symbol, ok := validatedSymbol(c, validator)
		if ok {
			c.JSON(200, gin.H{"symbol": symbol})
		}
	}
}
