package main

import (
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/kudaompq/ai_trending/backend/internal/database"
	"github.com/kudaompq/ai_trending/backend/internal/handler"
	"github.com/kudaompq/ai_trending/backend/internal/service"
)

func main() {
	// Initialize database
	if err := database.InitDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer database.CloseDB()

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
	klineHandler := handler.NewKlineHandler()
	analysisHandler := handler.NewAnalysisHandler()
	opportunityHandler := handler.NewOpportunityHandler()
	streamHandler := handler.NewStreamHandler(service.NewMarketStreamService())

	// API routes
	api := r.Group("/api")
	{
		// K-line data endpoint
		api.GET("/kline", klineHandler.GetKline)

		// Analysis endpoint
		api.GET("/analysis", analysisHandler.GetAnalysis)

		// Opportunities endpoint
		api.GET("/opportunities", opportunityHandler.GetOpportunities)

		// Real-time Binance kline stream (Server-Sent Events)
		api.GET("/stream", streamHandler.GetMarketStream)

		// Health check
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":  "ok",
				"message": "ETH Analysis API is running",
			})
		})
	}

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
	log.Println("  GET /api/opportunities?symbol=ETHUSDT&interval=1h&min_rr=3.0")
	log.Println("  GET /api/stream?symbol=ETHUSDT&interval=1h")

	if err := r.Run(address); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
