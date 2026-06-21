package tools

import (
	"encoding/json"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
)

// ToOpenAITools exports all registered tools as OpenAI function-calling tool
// parameters suitable for passing to ChatCompletionNewParams.Tools.
func (r *Registry) ToOpenAITools() []openai.ChatCompletionToolUnionParam {
	list := r.List()
	tools := make([]openai.ChatCompletionToolUnionParam, 0, len(list))
	for _, t := range list {
		var params shared.FunctionParameters
		if err := json.Unmarshal([]byte(t.Parameters()), &params); err != nil {
			params = shared.FunctionParameters{}
		}
		tools = append(tools, openai.ChatCompletionToolUnionParam{
			OfFunction: &openai.ChatCompletionFunctionToolParam{
				Function: shared.FunctionDefinitionParam{
					Name:        t.Name(),
					Description: openai.String(t.Description()),
					Parameters:  params,
				},
			},
		})
	}
	return tools
}
