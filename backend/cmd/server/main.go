package main

import (
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/kudaompq/ai_trending/backend/internal/handler"
)

func main() {
	// Create Gin router
	r := gin.Default()

	// CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Initialize handlers
	klineHandler := handler.NewKlineHandler()
	analysisHandler := handler.NewAnalysisHandler()

	// API routes
	api := r.Group("/api")
	{
		// K-line data endpoint
		api.GET("/kline", klineHandler.GetKline)

		// Analysis endpoint
		api.GET("/analysis", analysisHandler.GetAnalysis)

		// Health check
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status": "ok",
				"message": "ETH Analysis API is running",
			})
		})
	}

	// Start server
	log.Println("🚀 Server starting on :8080")
	log.Println("📊 ETH K-line Analysis API")
	log.Println("Endpoints:")
	log.Println("  GET /api/health")
	log.Println("  GET /api/kline?symbol=ETHUSDT&interval=1d&limit=100")
	log.Println("  GET /api/analysis?symbol=ETHUSDT&interval=1d&limit=100")

	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
