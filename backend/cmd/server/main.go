package main

import (
	"context"
	"errors"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/kudaompq/ai_trending/backend/internal/database"
	"github.com/kudaompq/ai_trending/backend/internal/handler"
	"github.com/kudaompq/ai_trending/backend/internal/repository"
	"github.com/kudaompq/ai_trending/backend/internal/service"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
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
	chatProviderConfig, chatProviderConfigured := service.OpenAICompatibleConfigFromEnv()
	var chatProvider service.ChatCompletionProvider
	if chatProviderConfigured {
		chatProvider = service.NewOpenAICompatibleProvider(chatProviderConfig)
	}
	chatService := service.NewMarketAnalysisChatService(service.NewMarketContextService(), chatProvider, service.NewMarketAnalysisToolExecutor())
	r := newRouterWithServices(watchlist, chatService)

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
	log.Println("  POST /api/analysis/chat")
	log.Println("  GET /api/stream?symbol=ETHUSDT&interval=1h")
	log.Println("  GET /api/watchlist/prices?symbols=BTCUSDT,ETHUSDT")
	log.Println("  GET /api/watchlist/stream?symbols=BTCUSDT,ETHUSDT")

	processContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var background sync.WaitGroup
	if service.ScheduledWatchlistAnalysisEnabledFromEnv() && chatProviderConfigured {
		analysisRepository := repository.NewPostgresScheduledAnalysisRepository(db)
		notifier, notifierConfigured := service.NewWeComNotifierFromEnv()
		if !notifierConfigured {
			log.Println("scheduled analysis enabled without a WeCom webhook; results will be stored and alerts will remain queued")
		}
		runner := service.NewScheduledWatchlistAnalysisService(watchlist, chatService, analysisRepository, service.ScheduledWatchlistAnalysisConfig{
			MaxConcurrency: envInt("SCHEDULED_ANALYSIS_CONCURRENCY", 4),
			SymbolTimeout:  envDuration("SCHEDULED_ANALYSIS_SYMBOL_TIMEOUT", 2*time.Minute),
			MaxRunDuration: envDuration("SCHEDULED_ANALYSIS_MAX_DURATION", 50*time.Minute),
			Model:          chatProviderConfig.Model,
			AlertSender:    notifier,
		})
		background.Add(1)
		go func() {
			defer background.Done()
			runner.Start(processContext)
		}()
	} else if service.ScheduledWatchlistAnalysisEnabledFromEnv() {
		log.Println("scheduled analysis is enabled but the AI provider is not configured")
	}

	server := &http.Server{Addr: address, Handler: r, ReadHeaderTimeout: 10 * time.Second}
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- server.ListenAndServe() }()
	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server stopped: %v", err)
		}
	case <-processContext.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		if err := server.Shutdown(shutdownContext); err != nil {
			log.Printf("HTTP shutdown failed: %v", err)
			_ = server.Close()
		}
		cancel()
	}
	stop()
	background.Wait()
}

func newRouter(watchlists ...*service.WatchlistService) *gin.Engine {
	var watchlist *service.WatchlistService
	if len(watchlists) > 0 {
		watchlist = watchlists[0]
	}
	chatProviderConfig, chatProviderConfigured := service.OpenAICompatibleConfigFromEnv()
	var chatProvider service.ChatCompletionProvider
	if chatProviderConfigured {
		chatProvider = service.NewOpenAICompatibleProvider(chatProviderConfig)
	}
	chatService := service.NewMarketAnalysisChatService(service.NewMarketContextService(), chatProvider, service.NewMarketAnalysisToolExecutor())
	return newRouterWithServices(watchlist, chatService)
}

func newRouterWithServices(watchlist *service.WatchlistService, chatService *service.MarketAnalysisChatService) *gin.Engine {
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
	if chatService == nil {
		chatProviderConfig, chatProviderConfigured := service.OpenAICompatibleConfigFromEnv()
		var chatProvider service.ChatCompletionProvider
		if chatProviderConfigured {
			chatProvider = service.NewOpenAICompatibleProvider(chatProviderConfig)
		}
		chatService = service.NewMarketAnalysisChatService(service.NewMarketContextService(), chatProvider, service.NewMarketAnalysisToolExecutor())
	}
	chatHandler := handler.NewMarketAnalysisChatHandler(validator, chatService)
	streamHandler := handler.NewStreamHandler(service.NewMarketStreamService(), validator)
	priceHandler := handler.NewWatchlistPriceHandler(service.NewMarketPriceService(), validator)
	openInterestHandler := handler.NewOpenInterestHandler(validator)

	// API routes
	api := r.Group("/api")
	{
		if watchlist != nil {
			handler.RegisterWatchlistRoutes(api, watchlist)
		}
		api.GET("/symbols/validate", handler.ValidateSymbol(validator))
		// K-line data endpoint
		api.GET("/kline", klineHandler.GetKline)
		api.GET("/open-interest", openInterestHandler.GetOpenInterest)

		// Analysis endpoint
		api.GET("/analysis", analysisHandler.GetAnalysis)
		api.POST("/analysis/chat", chatHandler.PostMessage)

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

func envInt(name string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(name)))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func envDuration(name string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(strings.TrimSpace(os.Getenv(name)))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
