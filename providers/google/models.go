package google

import gollmx "github.com/onlyhyde/gollm-x"

// GeminiModels contains all known Google Gemini models
var GeminiModels = []gollmx.Model{
	// Gemini 2.0 series
	{
		ID:            "gemini-2.0-flash-exp",
		Name:          "Gemini 2.0 Flash (Experimental)",
		Provider:      ProviderID,
		Description:   "Next generation model with improved capabilities",
		ContextWindow: 1048576,
		MaxOutput:     8192,
		InputPrice:    0.0,  // Free during experimental period
		OutputPrice:   0.0,
		Features: []gollmx.Feature{
			gollmx.FeatureChat,
			gollmx.FeatureStreaming,
			gollmx.FeatureVision,
			gollmx.FeatureTools,
			gollmx.FeatureJSON,
			gollmx.FeatureSystemPrompt,
		},
		ReleaseDate: "2024-12-11",
	},
	// Gemini 1.5 series
	{
		ID:            "gemini-1.5-pro",
		Name:          "Gemini 1.5 Pro",
		Provider:      ProviderID,
		Description:   "Best for complex reasoning tasks with 2M context",
		ContextWindow: 2097152,
		MaxOutput:     8192,
		InputPrice:    1.25,  // Per 1M tokens (under 128K)
		OutputPrice:   5.00,
		Features: []gollmx.Feature{
			gollmx.FeatureChat,
			gollmx.FeatureStreaming,
			gollmx.FeatureVision,
			gollmx.FeatureTools,
			gollmx.FeatureJSON,
			gollmx.FeatureSystemPrompt,
		},
		ReleaseDate: "2024-05-14",
	},
	{
		ID:            "gemini-1.5-flash",
		Name:          "Gemini 1.5 Flash",
		Provider:      ProviderID,
		Description:   "Fast and versatile model for most tasks",
		ContextWindow: 1048576,
		MaxOutput:     8192,
		InputPrice:    0.075,
		OutputPrice:   0.30,
		Features: []gollmx.Feature{
			gollmx.FeatureChat,
			gollmx.FeatureStreaming,
			gollmx.FeatureVision,
			gollmx.FeatureTools,
			gollmx.FeatureJSON,
			gollmx.FeatureSystemPrompt,
		},
		ReleaseDate: "2024-05-14",
	},
	{
		ID:            "gemini-1.5-flash-8b",
		Name:          "Gemini 1.5 Flash-8B",
		Provider:      ProviderID,
		Description:   "Smallest and fastest model for high-volume tasks",
		ContextWindow: 1048576,
		MaxOutput:     8192,
		InputPrice:    0.0375,
		OutputPrice:   0.15,
		Features: []gollmx.Feature{
			gollmx.FeatureChat,
			gollmx.FeatureStreaming,
			gollmx.FeatureVision,
			gollmx.FeatureTools,
			gollmx.FeatureJSON,
			gollmx.FeatureSystemPrompt,
		},
		ReleaseDate: "2024-10-03",
	},
	// Gemini 1.0 series (legacy)
	{
		ID:            "gemini-1.0-pro",
		Name:          "Gemini 1.0 Pro",
		Provider:      ProviderID,
		Description:   "Previous generation model for text tasks",
		ContextWindow: 32768,
		MaxOutput:     8192,
		InputPrice:    0.50,
		OutputPrice:   1.50,
		Features: []gollmx.Feature{
			gollmx.FeatureChat,
			gollmx.FeatureStreaming,
			gollmx.FeatureTools,
			gollmx.FeatureJSON,
			gollmx.FeatureSystemPrompt,
		},
		Deprecated:  true,
		ReleaseDate: "2023-12-06",
	},
	// Embedding models
	{
		ID:            "text-embedding-004",
		Name:          "Text Embedding 004",
		Provider:      ProviderID,
		Description:   "Latest embedding model with 768 dimensions",
		ContextWindow: 2048,
		InputPrice:    0.00,  // Free
		Features: []gollmx.Feature{
			gollmx.FeatureEmbedding,
		},
		ReleaseDate: "2024-05-14",
	},
	{
		ID:            "embedding-001",
		Name:          "Embedding 001",
		Provider:      ProviderID,
		Description:   "Previous generation embedding model",
		ContextWindow: 2048,
		InputPrice:    0.00,
		Features: []gollmx.Feature{
			gollmx.FeatureEmbedding,
		},
		Deprecated:  true,
		ReleaseDate: "2023-12-06",
	},
}
