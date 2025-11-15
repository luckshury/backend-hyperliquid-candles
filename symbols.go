package main

import (
	"log"
	"time"

	"github.com/anthdm/hollywood/actor"
)

// SymbolFetcherActor periodically fetches the list of perpetual symbols
type SymbolFetcherActor struct {
	cache              *Cache
	hydromancerClient  *HydromancerClient
	refreshInterval    time.Duration
	cachedSymbols      []string // Fallback cache
}

// NewSymbolFetcherActor creates a new symbol fetcher actor
func NewSymbolFetcherActor(cache *Cache, hydromancerClient *HydromancerClient, refreshInterval time.Duration) *SymbolFetcherActor {
	return &SymbolFetcherActor{
		cache:             cache,
		hydromancerClient: hydromancerClient,
		refreshInterval:   refreshInterval,
		cachedSymbols:     []string{},
	}
}

func (a *SymbolFetcherActor) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		log.Println("[SymbolFetcher] Actor started")
		// Fetch symbols immediately on start
		a.fetchSymbols()
		// Schedule periodic fetches
		ctx.SendRepeat(ctx.PID(), FetchSymbolsMsg{}, a.refreshInterval)
		
	case FetchSymbolsMsg:
		a.fetchSymbols()
		
	case GetSymbolsMsg:
		symbols := a.cache.GetSymbols()
		msg.ResponseChan <- symbols
		
	case actor.Stopped:
		log.Println("[SymbolFetcher] Actor stopped")
	}
}

func (a *SymbolFetcherActor) fetchSymbols() {
	log.Println("[SymbolFetcher] Fetching perpetual symbols from Hyperliquid...")
	
	symbols, err := a.hydromancerClient.FetchPerpetualSymbols()
	if err != nil {
		log.Printf("[SymbolFetcher] ERROR: Failed to fetch symbols: %v", err)
		// Use cached symbols if API fails
		if len(a.cachedSymbols) > 0 {
			log.Printf("[SymbolFetcher] Using cached symbol list (%d symbols)", len(a.cachedSymbols))
			a.cache.SetSymbols(a.cachedSymbols)
		}
		return
	}
	
	if len(symbols) == 0 {
		log.Println("[SymbolFetcher] WARNING: Received empty symbol list")
		return
	}
	
	log.Printf("[SymbolFetcher] Discovered %d symbols from Hyperliquid", len(symbols))
	
	// Update cache and fallback
	a.cache.SetSymbols(symbols)
	a.cachedSymbols = symbols
}

