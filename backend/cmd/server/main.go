package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/kudaompq/ai_trending/backend/internal/database"
	"github.com/kudaompq/ai_trending/backend/internal/handler"
	"github.com/kudaompq/ai_trending/backend/internal/repository"
	"github.com/kudaompq/ai_trending/backend/internal/service"
)

func main() {
	if err := database.RemoveLegacyOpportunityDatabase(filepath.Join("data", "opportunities.db")); err != nil {
		log.Fatalf("Failed to remove legacy trading-opportunity database: %v", err)
	}

	startupContext, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	db, err := database.OpenPostgres(startupContext, os.Getenv("WATCHLIST_DATABASE_URL"))
	cancel()
	if err != nil {
		log.Fatalf("Failed to initialize PostgreSQL: %v", err)
	}
	defer db.Close()

	validator := service.NewSymbolValidator()
	watchlist := service.NewWatchlistService(repository.NewPostgresWatchlistRepository(db), validator)
	r := newRouter(watchlist)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	address := ":" + port
	log.Println("🚀 Server starting on " + address)
	log.Println("📊 ETH K-line Analysis API")
	log.Println("Endpoints:")
	log.Println("  GET /api/health")
	log.Println("  GET /api/kline?symbol=ETHUSDT&interval=1d&limit=100")
	log.Println("  GET /api/analysis?symbol=ETHUSDT&interval=1d&limit=100")
	log.Println("  GET /api/stream?symbol=ETHUSDT&interval=1h")
	log.Println("  GET /api/watchlist/prices?symbols=BTCUSDT,ETHUSDT")
	log.Println("  GET /api/watchlist/stream?symbols=BTCUSDT,ETHUSDT")

	if err := r.Run(address); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func newRouter(watchlists ...*service.WatchlistService) *gin.Engine {
	// Create Gin router
	r := gin.Default()

	// CORS middleware - Allow all origins for development
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false, // Must be false when AllowAllOrigins is true
	}))

	// Initialize handlers
	validator := service.NewSymbolValidator()
	klineHandler := handler.NewKlineHandler(validator)
	analysisHandler := handler.NewAnalysisHandler(validator)
	streamHandler := handler.NewStreamHandler(service.NewMarketStreamService(), validator)
	priceHandler := handler.NewWatchlistPriceHandler(service.NewMarketPriceService(), validator)
	openInterestHandler := handler.NewOpenInterestHandler(validator)

	// API routes
	api := r.Group("/api")
	{
		if len(watchlists) > 0 && watchlists[0] != nil {
			handler.RegisterWatchlistRoutes(api, watchlists[0])
		}
		api.GET("/symbols/validate", handler.ValidateSymbol(validator))
		// K-line data endpoint
		api.GET("/kline", klineHandler.GetKline)
		api.GET("/open-interest", openInterestHandler.GetOpenInterest)

		// Analysis endpoint
		api.GET("/analysis", analysisHandler.GetAnalysis)

		// Real-time Binance kline stream (Server-Sent Events)
		api.GET("/stream", streamHandler.GetMarketStream)
		api.GET("/watchlist/prices", priceHandler.GetPrices)
		api.GET("/watchlist/stream", priceHandler.GetStream)

		// Health check
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":  "ok",
				"message": "ETH Analysis API is running",
			})
		})
	}

	return r
}
