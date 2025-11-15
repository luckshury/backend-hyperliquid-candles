package main

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/anthdm/hollywood/actor"
)

var (
	cache             *Cache
	engine            *actor.Engine
	symbolFetcherPID  *actor.PID
	candleFetcherPID  *actor.PID
)

// Config holds application configuration
type Config struct {
	Port                      string
	HydromancerAPIKey         string
	CandleInterval            string
	CandleDays                int
	RefreshIntervalMin        int
	SymbolRefreshIntervalMin  int
}

func loadConfig() *Config {
	return &Config{
		Port:                      getEnv("PORT", "3000"),
		HydromancerAPIKey:         getEnv("HYDROMANCER_API_KEY", "sk_nNhuLkdGdW5sxnYec33C2FBPzLjXBnEd"),
		CandleInterval:            getEnv("CANDLE_INTERVAL", "1h"),
		CandleDays:                getEnvInt("CANDLE_DAYS", 7),
		RefreshIntervalMin:        getEnvInt("REFRESH_INTERVAL_MIN", 5),
		SymbolRefreshIntervalMin:  getEnvInt("SYMBOL_REFRESH_INTERVAL_MIN", 60),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	config := loadConfig()
	
	// Initialize cache
	cache = NewCache()
	
	// Initialize API clients
	hydromancerClient := NewHydromancerClient(config.HydromancerAPIKey)
	hyperliquidClient := NewHyperliquidClient()
	
	// Initialize Hollywood actor engine
	var err error
	engine, err = actor.NewEngine(actor.EngineConfig{})
	if err != nil {
		log.Fatalf("Failed to create actor engine: %v", err)
	}
	
	// Spawn symbol fetcher actor
	symbolFetcherPID = engine.Spawn(
		func() actor.Receiver {
			return NewSymbolFetcherActor(
				cache,
				hydromancerClient,
				time.Duration(config.SymbolRefreshIntervalMin)*time.Minute,
			)
		},
		"symbolFetcher",
	)
	
	// Spawn candle fetcher actor
	candleFetcherPID = engine.Spawn(
		func() actor.Receiver {
			return NewCandleFetcherActor(
				cache,
				hyperliquidClient,
				time.Duration(config.RefreshIntervalMin)*time.Minute,
				config.CandleInterval,
				config.CandleDays,
			)
		},
		"candleFetcher",
	)
	
	// Setup HTTP server
	mux := http.NewServeMux()
	
	// API endpoints
	mux.HandleFunc("/api/candles", logRequest(gzipHandler(handleGetAllCandles)))
	mux.HandleFunc("/api/candles/", logRequest(gzipHandler(handleGetSymbolCandles)))
	mux.HandleFunc("/api/symbols", logRequest(gzipHandler(handleGetSymbols)))
	mux.HandleFunc("/health", logRequest(handleHealth))
	
	// Wrap with CORS
	handler := corsMiddleware(mux)
	
	// Start server
	server := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		
		log.Println("Shutting down gracefully...")
		
		// Stop actors
		engine.Poison(symbolFetcherPID)
		engine.Poison(candleFetcherPID)
		
		// Shutdown HTTP server
		if err := server.Close(); err != nil {
			log.Printf("Error shutting down server: %v", err)
		}
		
		os.Exit(0)
	}()
	
	log.Printf("Server started on port %s", config.Port)
	log.Printf("Candle interval: %s, History: %d days", config.CandleInterval, config.CandleDays)
	log.Printf("Refresh intervals - Candles: %dm, Symbols: %dm", config.RefreshIntervalMin, config.SymbolRefreshIntervalMin)
	
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}

// HTTP Handlers

func handleGetAllCandles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	allCandles := cache.GetAll()
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("ETag", generateETag(cache.GetLastUpdate()))
	
	if err := json.NewEncoder(w).Encode(allCandles); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func handleGetSymbolCandles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Extract symbol from path: /api/candles/BTC -> BTC
	path := strings.TrimPrefix(r.URL.Path, "/api/candles/")
	symbol := strings.ToUpper(path)
	
	if symbol == "" {
		http.Error(w, "Symbol required", http.StatusBadRequest)
		return
	}
	
	entry, exists := cache.Get(symbol)
	if !exists {
		http.Error(w, "Symbol not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("ETag", generateETag(entry.LastUpdate))
	
	if err := json.NewEncoder(w).Encode(entry); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func handleGetSymbols(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	symbols := cache.GetSymbols()
	
	w.Header().Set("Content-Type", "application/json")
	
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"symbols": symbols,
		"count":   len(symbols),
	}); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	symbols := cache.GetSymbols()
	lastUpdate := cache.GetLastUpdate()
	symbolUpdate := cache.GetSymbolUpdate()
	
	health := HealthResponse{
		Status:       "healthy",
		SymbolCount:  len(symbols),
		LastUpdate:   lastUpdate,
		SymbolUpdate: symbolUpdate,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// Middleware

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func logRequest(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next(wrapped, r)
		
		duration := time.Since(start)
		log.Printf("%s %s - %d - %v", r.Method, r.URL.Path, wrapped.statusCode, duration)
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func gzipHandler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next(w, r)
			return
		}
		
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		
		gzw := gzipResponseWriter{Writer: gz, ResponseWriter: w}
		next(gzw, r)
	}
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// Utilities

func generateETag(t time.Time) string {
	return `"` + strconv.FormatInt(t.Unix(), 10) + `"`
}

