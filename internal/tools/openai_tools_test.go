package tools

import (
	"testing"
)

func TestToOpenAITools(t *testing.T) {
	r := NewRegistry()
	r.Register(&StockInfoTool{})
	r.Register(&FinancialsTool{})

	tools := r.ToOpenAITools()
	if len(tools) != 2 {
		t.Fatalf("Expected 2 tools, got %d", len(tools))
	}

	names := map[string]bool{}
	for _, tool := range tools {
		if tool.OfFunction == nil {
			t.Error("Expected OfFunction to be set")
			continue
		}
		fn := tool.OfFunction.Function
		names[fn.Name] = true

		if !fn.Description.Valid() || fn.Description.Value == "" {
			t.Errorf("Tool %q: expected non-empty description", fn.Name)
		}

		if fn.Parameters == nil {
			t.Errorf("Tool %q: expected non-nil parameters", fn.Name)
		}
		if _, ok := fn.Parameters["properties"]; !ok {
			t.Errorf("Tool %q: expected 'properties' in parameters", fn.Name)
		}
	}

	if !names["get_stock_info"] {
		t.Error("Expected get_stock_info tool")
	}
	if !names["get_financials"] {
		t.Error("Expected get_financials tool")
	}
}

func TestToOpenAIToolsEmpty(t *testing.T) {
	r := NewRegistry()
	tools := r.ToOpenAITools()
	if len(tools) != 0 {
		t.Fatalf("Expected 0 tools, got %d", len(tools))
	}
}
