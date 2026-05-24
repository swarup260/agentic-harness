// package main

// import (
// 	"context"
// 	"fmt"

// 	"github.com/openai/openai-go/v3"
// 	"github.com/openai/openai-go/v3/option"
// )

// func main() {
// 	// Define URL
// 	llmURL := "http://0.0.0.0:8080"

// 	ctx := context.Background()

// 	client := openai.NewClient(
// 		option.WithBaseURL(llmURL),
// 		option.WithAPIKey("sk-no-key"),
// 	)

// 	// params := openai.ChatCompletionNewParams{
// 	// 	Messages: []openai.ChatCompletionMessageParamUnion{
// 	// 		openai.SystemMessage("You are a senior stock analyst with 20+ years of experience. You always output the final answer in bullet points"),
// 	// 		openai.UserMessage("Write a haiku about computers"),
// 	// 	},
// 	// 	Seed:        openai.Int(0),
// 	// 	Temperature: openai.Float(0),
// 	// }

// 	// zero-shot prompt example

// 	// paramsZeroShot := openai.ChatCompletionNewParams{
// 	// 	Messages: []openai.ChatCompletionMessageParamUnion{
// 	// 		openai.SystemMessage("You are a senior stock analyst with 20+ years of experience. You always output the final answer in bullet points"),
// 	// 		openai.UserMessage("What is the fundamental analysis of Apple stock?"),
// 	// 	},
// 	// 	Seed:        openai.Int(0),
// 	// 	Temperature: openai.Float(0),
// 	// }

// 	// few-shot prompt example
// 	// 	paramsFewShot := openai.ChatCompletionNewParams{
// 	// 		Messages: []openai.ChatCompletionMessageParamUnion{
// 	// 			openai.SystemMessage(
// 	// 				"You are a senior stock analyst with 20+ years of experience. " +
// 	// 					"You always output the final answer in bullet points.",
// 	// 			),

// 	// 			// Example 1
// 	// 			openai.UserMessage("What is the fundamental analysis of Microsoft stock?"),
// 	// 			openai.AssistantMessage(`
// 	// - Strong recurring revenue from Azure and Office 365
// 	// - High operating margins and strong free cash flow
// 	// - Diversified enterprise business reduces risk
// 	// - AI investments are improving long-term growth outlook
// 	// `),

// 	// 			// Example 2
// 	// 			openai.UserMessage("What is the sentiment analysis of Tesla stock?"),
// 	// 			openai.AssistantMessage(`
// 	// - Retail investor sentiment remains highly polarized
// 	// - Analysts are divided on valuation sustainability
// 	// - Strong brand loyalty supports bullish outlook
// 	// - EV competition is increasing market uncertainty
// 	// `),

// 	// 			// Actual query
// 	// 			openai.UserMessage("What is the news analysis of Apple stock?"),
// 	// 		},
// 	// 		Seed:        openai.Int(0),
// 	// 		Temperature: openai.Float(0),
// 	// 	}

// 	// 	stream := client.Chat.Completions.NewStreaming(ctx, paramsFewShot)

// 	paramsChainOfThought := openai.ChatCompletionNewParams{
// 		Messages: []openai.ChatCompletionMessageParamUnion{
// 			openai.SystemMessage(
// 				"Analyze this bug report step by step:\n" +
// 					"1. What patterns do you observe? (timing, scope, triggers)\n" +
// 					"2. What does each clue rule in or rule out?\n" +
// 					"3. What is the most likely root cause?\n" +
// 					"4. What would you check first to confirm?",
// 			),

// 			openai.UserMessage(`
// Users report intermittent 500 errors after deployment.

// Observations:
// - Errors started immediately after release v2.4.1
// - Only affects EU region
// - Requests fail only for authenticated users
// - Logs show database timeout spikes
// - CPU and memory usage remain normal
// - Rolling back fixes the issue
// `),
// 		},

// 		Seed:        openai.Int(0),
// 		Temperature: openai.Float(0),
// 	}

// 	stream := client.Chat.Completions.NewStreaming(ctx, paramsChainOfThought)

// 	for stream.Next() {
// 		event := stream.Current()

// 		if len(event.Choices) > 0 {
// 			fmt.Print(event.Choices[0].Delta.Content)
// 		}
// 	}

// 	if stream.Err() != nil {
// 		panic(stream.Err())
// 	}

// 	fmt.Println()
// }

package main

// These imports will be used later in the tutorial. If you save the file
// now, Go might complain they are unused, but that's fine.
// You may also need to run `go mod tidy` to download bubbletea and its
// dependencies.
import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

type model struct {
	choices  []string         // items on the to-do list
	cursor   int              // which to-do list item our cursor is pointing at
	selected map[int]struct{} // which to-do items are selected
}

func initialModel() model {
	return model{
		// Our to-do list is a grocery list
		choices: []string{"Buy carrots", "Buy celery", "Buy kohlrabi"},

		// A map which indicates which choices are selected. We're using
		// the  map like a mathematical set. The keys refer to the indexes
		// of the `choices` slice, above.
		selected: make(map[int]struct{}),
	}
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Is it a key press?
	case tea.KeyPressMsg:

		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit

		// The "up" and "k" keys move the cursor up
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		// The "down" and "j" keys move the cursor down
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		// The "enter" key and the space bar toggle the selected state
		// for the item that the cursor is pointing at.
		case "enter", "space":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}

func (m model) View() tea.View {
	// The header
	s := "What should we buy at the market?\n\n"

	// Iterate over our choices
	for i, choice := range m.choices {

		// Is the cursor pointing at this choice?
		cursor := " " // no cursor
		if m.cursor == i {
			cursor = ">" // cursor!
		}

		// Is this choice selected?
		checked := " " // not selected
		if _, ok := m.selected[i]; ok {
			checked = "x" // selected!
		}

		// Render the row
		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
	}

	// The footer
	s += "\nPress q to quit.\n"

	// Send the UI for rendering
	return tea.NewView(s)
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
