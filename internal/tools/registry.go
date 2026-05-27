package tools

import "context"

// Tool represents a capability the LLM can invoke.
type Tool interface {
	Name() string
	Description() string
	Parameters() string // JSON schema of parameters
	Execute(ctx context.Context, args string) (string, error)
}

// Registry manages the set of available tools.
type Registry struct {
	tools map[string]Tool
}

// NewRegistry initializes an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry.
func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

// Get retrieves a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	t, exists := r.tools[name]
	return t, exists
}

// List returns all registered tools.
func (r *Registry) List() []Tool {
	list := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		list = append(list, t)
	}
	return list
}
