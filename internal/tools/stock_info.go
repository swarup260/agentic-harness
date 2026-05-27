package tools

import (
	"context"
	"fmt"
)

// StockInfoTool fetches information about a stock symbol.
type StockInfoTool struct{}

func (t *StockInfoTool) Name() string {
	return "get_stock_info"
}

func (t *StockInfoTool) Description() string {
	return "Get real-time and historical stock data for a given ticker symbol."
}

func (t *StockInfoTool) Parameters() string {
	return `{"type":"object","properties":{"symbol":{"type":"string","description":"The stock ticker symbol, e.g., AAPL"}},"required":["symbol"]}`
}

func (t *StockInfoTool) Execute(ctx context.Context, args string) (string, error) {
	// Placeholder execution logic
	return fmt.Sprintf("Stock info result for arguments: %s (Simulated API call)", args), nil
}
