package main

import (
	"time"
)

// Candle represents OHLCV data for a specific timeframe
type Candle struct {
	Timestamp int64   `json:"timestamp"` // Unix timestamp in milliseconds
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    float64 `json:"volume"`
}

// CacheEntry holds candle data for a single symbol
type CacheEntry struct {
	Symbol     string    `json:"symbol"`
	Candles    []Candle  `json:"candles"`
	LastUpdate time.Time `json:"last_update"`
}

// SymbolList holds the list of active perpetual symbols
type SymbolList struct {
	Symbols   []string  `json:"symbols"`
	LastFetch time.Time `json:"last_fetch"`
}

// HyperliquidCandleRequest represents the request to fetch candles
type HyperliquidCandleRequest struct {
	Type string `json:"type"`
	Req  struct {
		Coin      string `json:"coin"`
		Interval  string `json:"interval"`
		StartTime int64  `json:"startTime"`
		EndTime   int64  `json:"endTime"`
	} `json:"req"`
}

// HyperliquidCandle represents the raw response from Hyperliquid
type HyperliquidCandle struct {
	T int64   `json:"t"` // Timestamp
	O float64 `json:"o,string"` // Open
	H float64 `json:"h,string"` // High
	L float64 `json:"l,string"` // Low
	C float64 `json:"c,string"` // Close
	V float64 `json:"v,string"` // Volume
	N int     `json:"n"` // Number of trades
}

// HydromancerRequest represents the request to Hydromancer API
type HydromancerRequest struct {
	Type string `json:"type"`
}

// HydromancerPerpDeployResponse represents the perpDeployAuctionStatus response
type HydromancerPerpDeployResponse struct {
	Universe []struct {
		Name string `json:"name"`
	} `json:"universe"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status       string    `json:"status"`
	SymbolCount  int       `json:"symbol_count"`
	LastUpdate   time.Time `json:"last_update,omitempty"`
	SymbolUpdate time.Time `json:"symbol_update,omitempty"`
}

// Actor Messages
type FetchSymbolsMsg struct{}
type FetchCandlesMsg struct{}
type GetCacheMsg struct {
	ResponseChan chan map[string]CacheEntry
}
type GetSymbolsMsg struct {
	ResponseChan chan []string
}

