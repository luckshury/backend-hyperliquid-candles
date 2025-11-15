package main

import (
	"sync"
	"time"
)

// Cache provides thread-safe access to candle data
type Cache struct {
	mu          sync.RWMutex
	data        map[string]CacheEntry
	symbols     []string
	lastUpdate  time.Time
	symbolUpdate time.Time
}

// NewCache creates a new cache instance
func NewCache() *Cache {
	return &Cache{
		data:    make(map[string]CacheEntry),
		symbols: []string{},
	}
}

// Set stores candle data for a symbol
func (c *Cache) Set(symbol string, candles []Candle) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.data[symbol] = CacheEntry{
		Symbol:     symbol,
		Candles:    candles,
		LastUpdate: time.Now(),
	}
	c.lastUpdate = time.Now()
}

// Get retrieves candle data for a specific symbol
func (c *Cache) Get(symbol string) (CacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	entry, exists := c.data[symbol]
	return entry, exists
}

// GetAll returns all cached data
func (c *Cache) GetAll() map[string]CacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	result := make(map[string]CacheEntry, len(c.data))
	for k, v := range c.data {
		result[k] = v
	}
	return result
}

// SetSymbols updates the active symbol list
func (c *Cache) SetSymbols(symbols []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.symbols = symbols
	c.symbolUpdate = time.Now()
}

// GetSymbols returns the active symbol list
func (c *Cache) GetSymbols() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Return a copy
	result := make([]string, len(c.symbols))
	copy(result, c.symbols)
	return result
}

// GetLastUpdate returns the time of the last candle update
func (c *Cache) GetLastUpdate() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastUpdate
}

// GetSymbolUpdate returns the time of the last symbol list update
func (c *Cache) GetSymbolUpdate() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.symbolUpdate
}

