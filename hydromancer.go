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
	hydromancerURL = "https://api.hydromancer.xyz/info"
)

// HydromancerClient handles API calls to Hydromancer and Hyperliquid
type HydromancerClient struct {
	apiKey     string
	httpClient *http.Client
}

// NewHydromancerClient creates a new Hydromancer client
func NewHydromancerClient(apiKey string) *HydromancerClient {
	return &HydromancerClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// MetaResponse represents the response from Hyperliquid's meta endpoint
type MetaResponse struct {
	Universe []struct {
		Name       string `json:"name"`
		IsDelisted bool   `json:"isDelisted,omitempty"`
	} `json:"universe"`
}

// FetchPerpetualSymbols fetches all active perpetual symbols from Hyperliquid
func (c *HydromancerClient) FetchPerpetualSymbols() ([]string, error) {
	// Use Hyperliquid's meta endpoint to get all symbols
	reqBody := map[string]interface{}{
		"type": "meta",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Use Hyperliquid API directly (not Hydromancer for this)
	req, err := http.NewRequest("POST", "https://api.hyperliquid.xyz/info", bytes.NewBuffer(jsonData))
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

	var response MetaResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract symbol names from universe (exclude delisted symbols)
	symbols := make([]string, 0, len(response.Universe))
	for _, item := range response.Universe {
		if item.Name != "" && !item.IsDelisted {
			symbols = append(symbols, item.Name)
		}
	}

	return symbols, nil
}

