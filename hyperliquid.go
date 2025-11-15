package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	hyperliquidURL = "https://api.hyperliquid.xyz/info"
)

// HyperliquidClient handles API calls to Hyperliquid
type HyperliquidClient struct {
	httpClient *http.Client
}

// NewHyperliquidClient creates a new Hyperliquid client
func NewHyperliquidClient() *HyperliquidClient {
	return &HyperliquidClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// FetchCandles fetches candle data for a specific symbol
func (c *HyperliquidClient) FetchCandles(symbol, interval string, startTime, endTime int64) ([]Candle, error) {
	reqBody := map[string]interface{}{
		"type": "candleSnapshot",
		"req": map[string]interface{}{
			"coin":      symbol,
			"interval":  interval,
			"startTime": startTime,
			"endTime":   endTime,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", hyperliquidURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var rawCandles []HyperliquidCandle
	if err := json.Unmarshal(body, &rawCandles); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to our Candle format
	candles := make([]Candle, len(rawCandles))
	for i, rc := range rawCandles {
		candles[i] = Candle{
			Timestamp: rc.T,
			Open:      rc.O,
			High:      rc.H,
			Low:       rc.L,
			Close:     rc.C,
			Volume:    rc.V,
		}
	}

	return candles, nil
}

// FetchCandlesWithRetry fetches candles with exponential backoff retry
func (c *HyperliquidClient) FetchCandlesWithRetry(symbol, interval string, startTime, endTime int64, maxRetries int) ([]Candle, error) {
	var lastErr error
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		candles, err := c.FetchCandles(symbol, interval, startTime, endTime)
		if err == nil {
			return candles, nil
		}
		
		lastErr = err
		if attempt < maxRetries-1 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			time.Sleep(backoff)
		}
	}
	
	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

