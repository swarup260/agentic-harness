package tools

import (
	"context"
	"fmt"
)

// FinancialsTool fetches company balance sheet and income statements.
type FinancialsTool struct{}

func (t *FinancialsTool) Name() string {
	return "get_financials"
}

func (t *FinancialsTool) Description() string {
	return "Get the latest SEC filings, balance sheet, and financial metrics for a company."
}

func (t *FinancialsTool) Parameters() string {
	return `{"type":"object","properties":{"symbol":{"type":"string","description":"The stock ticker symbol, e.g., MSFT"}},"required":["symbol"]}`
}

func (t *FinancialsTool) Execute(ctx context.Context, args string) (string, error) {
	// Placeholder execution logic
	return fmt.Sprintf("Financials result for arguments: %s (Simulated API call)", args), nil
}
