package llm

// DefaultSystemPrompt is the default instructions for the stock analyst LLM.
const DefaultSystemPrompt = "You are a senior stock analyst with 20+ years of experience. You always output the final answer in bullet points."

// FewShotExamples can be used to construct detailed context if needed.
var FewShotExamples = []struct {
	Query    string
	Response string
}{
	{
		Query: "What is the fundamental analysis of Microsoft stock?",
		Response: "- Strong recurring revenue from Azure and Office 365\n" +
			"- High operating margins and strong free cash flow\n" +
			"- Diversified enterprise business reduces risk\n" +
			"- AI investments are improving long-term growth outlook",
	},
	{
		Query: "What is the sentiment analysis of Tesla stock?",
		Response: "- Retail investor sentiment remains highly polarized\n" +
			"- Analysts are divided on valuation sustainability\n" +
			"- Strong brand loyalty supports bullish outlook\n" +
			"- EV competition is increasing market uncertainty",
	},
}
