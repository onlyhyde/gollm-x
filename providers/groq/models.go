package groq

import gollmx "github.com/onlyhyde/gollm-x"

// GroqModels contains all known Groq models
var GroqModels = []gollmx.Model{
	// Llama 3.3
	{
		ID:            "llama-3.3-70b-versatile",
		Name:          "Llama 3.3 70B Versatile",
		Provider:      ProviderID,
		Description:   "Latest Llama model, great for diverse tasks",
		ContextWindow: 128000,
		MaxOutput:     32768,
		InputPrice:    0.59,
		OutputPrice:   0.79,
		Features: []gollmx.Feature{
			gollmx.FeatureChat,
			gollmx.FeatureStreaming,
			gollmx.FeatureTools,
			gollmx.FeatureJSON,
			gollmx.FeatureSystemPrompt,
		},
		ReleaseDate: "2024-12-06",
	},
	// Llama 3.1 series
	{
		ID:            "llama-3.1-70b-versatile",
		Name:          "Llama 3.1 70B Versatile",
		Provider:      ProviderID,
		Description:   "Versatile model for complex reasoning tasks",
		ContextWindow: 128000,
		MaxOutput:     32768,
		InputPrice:    0.59,
		OutputPrice:   0.79,
		Features: []gollmx.Feature{
			gollmx.FeatureChat,
			gollmx.FeatureStreaming,
			gollmx.FeatureTools,
			gollmx.FeatureJSON,
			gollmx.FeatureSystemPrompt,
		},
		ReleaseDate: "2024-07-23",
	},
	{
		ID:            "llama-3.1-8b-instant",
		Name:          "Llama 3.1 8B Instant",
		Provider:      ProviderID,
		Description:   "Fast model for quick responses",
		ContextWindow: 128000,
		MaxOutput:     8192,
		InputPrice:    0.05,
		OutputPrice:   0.08,
		Features: []gollmx.Feature{
			gollmx.FeatureChat,
			gollmx.FeatureStreaming,
			gollmx.FeatureTools,
			gollmx.FeatureJSON,
			gollmx.FeatureSystemPrompt,
		},
		ReleaseDate: "2024-07-23",
	},
	// Llama 3 series
	{
		ID:            "llama3-70b-8192",
		Name:          "Llama 3 70B",
		Provider:      ProviderID,
		Description:   "Large model with 8K context",
		ContextWindow: 8192,
		MaxOutput:     8192,
		InputPrice:    0.59,
		OutputPrice:   0.79,
		Features: []gollmx.Feature{
			gollmx.FeatureChat,
			gollmx.FeatureStreaming,
			gollmx.FeatureSystemPrompt,
		},
		ReleaseDate: "2024-04-18",
	},
	{
		ID:            "llama3-8b-8192",
		Name:          "Llama 3 8B",
		Provider:      ProviderID,
		Description:   "Compact model with 8K context",
		ContextWindow: 8192,
		MaxOutput:     8192,
		InputPrice:    0.05,
		OutputPrice:   0.08,
		Features: []gollmx.Feature{
			gollmx.FeatureChat,
			gollmx.FeatureStreaming,
			gollmx.FeatureSystemPrompt,
		},
		ReleaseDate: "2024-04-18",
	},
	// Mixtral
	{
		ID:            "mixtral-8x7b-32768",
		Name:          "Mixtral 8x7B",
		Provider:      ProviderID,
		Description:   "MoE model with 32K context",
		ContextWindow: 32768,
		MaxOutput:     32768,
		InputPrice:    0.24,
		OutputPrice:   0.24,
		Features: []gollmx.Feature{
			gollmx.FeatureChat,
			gollmx.FeatureStreaming,
			gollmx.FeatureSystemPrompt,
		},
		ReleaseDate: "2023-12-11",
	},
	// Gemma
	{
		ID:            "gemma2-9b-it",
		Name:          "Gemma 2 9B",
		Provider:      ProviderID,
		Description:   "Google's Gemma 2 instruction-tuned model",
		ContextWindow: 8192,
		MaxOutput:     8192,
		InputPrice:    0.20,
		OutputPrice:   0.20,
		Features: []gollmx.Feature{
			gollmx.FeatureChat,
			gollmx.FeatureStreaming,
			gollmx.FeatureSystemPrompt,
		},
		ReleaseDate: "2024-06-27",
	},
}
