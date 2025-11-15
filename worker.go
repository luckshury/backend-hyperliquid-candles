package main

import (
	"log"
	"time"

	"github.com/anthdm/hollywood/actor"
)

// CandleFetcherActor periodically fetches candle data for all symbols
type CandleFetcherActor struct {
	cache             *Cache
	hyperliquidClient *HyperliquidClient
	refreshInterval   time.Duration
	candleInterval    string
	candleDays        int
	batchSize         int
	batchDelay        time.Duration
}

// NewCandleFetcherActor creates a new candle fetcher actor
func NewCandleFetcherActor(
	cache *Cache,
	hyperliquidClient *HyperliquidClient,
	refreshInterval time.Duration,
	candleInterval string,
	candleDays int,
) *CandleFetcherActor {
	return &CandleFetcherActor{
		cache:             cache,
		hyperliquidClient: hyperliquidClient,
		refreshInterval:   refreshInterval,
		candleInterval:    candleInterval,
		candleDays:        candleDays,
		batchSize:         10,
		batchDelay:        200 * time.Millisecond,
	}
}

func (a *CandleFetcherActor) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		log.Println("[CandleFetcher] Actor started")
		// Fetch candles immediately on start
		a.fetchAllCandles()
		// Schedule periodic fetches
		ctx.SendRepeat(ctx.PID(), FetchCandlesMsg{}, a.refreshInterval)
		
	case FetchCandlesMsg:
		a.fetchAllCandles()
		
	case GetCacheMsg:
		msg.ResponseChan <- a.cache.GetAll()
		
	case actor.Stopped:
		log.Println("[CandleFetcher] Actor stopped")
	}
}

func (a *CandleFetcherActor) fetchAllCandles() {
	symbols := a.cache.GetSymbols()
	
	if len(symbols) == 0 {
		log.Println("[CandleFetcher] No symbols available yet, skipping fetch")
		return
	}
	
	log.Printf("[CandleFetcher] Found %d symbols, starting candle fetch...", len(symbols))
	
	// Calculate time range
	endTime := time.Now().UnixMilli()
	startTime := time.Now().AddDate(0, 0, -a.candleDays).UnixMilli()
	
	totalBatches := (len(symbols) + a.batchSize - 1) / a.batchSize
	successCount := 0
	
	for batchIdx := 0; batchIdx < len(symbols); batchIdx += a.batchSize {
		end := batchIdx + a.batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		
		batch := symbols[batchIdx:end]
		currentBatch := (batchIdx / a.batchSize) + 1
		
		log.Printf("[CandleFetcher] Fetching batch %d/%d (%d symbols)...", currentBatch, totalBatches, len(batch))
		
		// Fetch batch concurrently
		type result struct {
			symbol  string
			candles []Candle
			err     error
		}
		
		results := make(chan result, len(batch))
		
		for _, symbol := range batch {
			go func(sym string) {
				candles, err := a.hyperliquidClient.FetchCandlesWithRetry(
					sym,
					a.candleInterval,
					startTime,
					endTime,
					3, // max retries
				)
				results <- result{symbol: sym, candles: candles, err: err}
			}(symbol)
		}
		
		// Collect results
		for i := 0; i < len(batch); i++ {
			res := <-results
			if res.err != nil {
				log.Printf("[CandleFetcher] ERROR: Failed to fetch %s: %v", res.symbol, res.err)
				// Store empty array for failed symbols
				a.cache.Set(res.symbol, []Candle{})
			} else {
				a.cache.Set(res.symbol, res.candles)
				successCount++
			}
		}
		
		// Delay between batches to avoid rate limiting
		if currentBatch < totalBatches {
			time.Sleep(a.batchDelay)
		}
	}
	
	log.Printf("[CandleFetcher] Batch %d/%d complete (%d symbols cached successfully)", totalBatches, totalBatches, successCount)
	log.Printf("[CandleFetcher] ✓ Cached %d/%d symbols", successCount, len(symbols))
}

