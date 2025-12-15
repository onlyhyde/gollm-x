package mistral

import gollmx "github.com/onlyhyde/gollm-x"

// MistralModels contains all known Mistral models
var MistralModels = []gollmx.Model{
	// Premier models
	{
		ID:            "mistral-large-latest",
		Name:          "Mistral Large",
		Provider:      ProviderID,
		Description:   "Most powerful Mistral model for complex tasks",
		ContextWindow: 128000,
		MaxOutput:     8192,
		InputPrice:    2.00,
		OutputPrice:   6.00,
		Features: []gollmx.Feature{
			gollmx.FeatureChat,
			gollmx.FeatureStreaming,
			gollmx.FeatureTools,
			gollmx.FeatureJSON,
			gollmx.FeatureSystemPrompt,
		},
		ReleaseDate: "2024-11-18",
	},
	{
		ID:            "mistral-small-latest",
		Name:          "Mistral Small",
		Provider:      ProviderID,
		Description:   "Cost-efficient model for everyday tasks",
		ContextWindow: 32000,
		MaxOutput:     8192,
		InputPrice:    0.20,
		OutputPrice:   0.60,
		Features: []gollmx.Feature{
			gollmx.FeatureChat,
			gollmx.FeatureStreaming,
			gollmx.FeatureTools,
			gollmx.FeatureJSON,
			gollmx.FeatureSystemPrompt,
		},
		ReleaseDate: "2024-09-18",
	},
	// Codestral
	{
		ID:            "codestral-latest",
		Name:          "Codestral",
		Provider:      ProviderID,
		Description:   "Specialized model for code generation",
		ContextWindow: 32000,
		MaxOutput:     8192,
		InputPrice:    0.20,
		OutputPrice:   0.60,
		Features: []gollmx.Feature{
			gollmx.FeatureChat,
			gollmx.FeatureStreaming,
			gollmx.FeatureSystemPrompt,
		},
		ReleaseDate: "2024-05-29",
	},
	// Ministral
	{
		ID:            "ministral-8b-latest",
		Name:          "Ministral 8B",
		Provider:      ProviderID,
		Description:   "Compact model for edge deployment",
		ContextWindow: 128000,
		MaxOutput:     8192,
		InputPrice:    0.10,
		OutputPrice:   0.10,
		Features: []gollmx.Feature{
			gollmx.FeatureChat,
			gollmx.FeatureStreaming,
			gollmx.FeatureSystemPrompt,
		},
		ReleaseDate: "2024-10-16",
	},
	{
		ID:            "ministral-3b-latest",
		Name:          "Ministral 3B",
		Provider:      ProviderID,
		Description:   "Ultra-compact model for simple tasks",
		ContextWindow: 128000,
		MaxOutput:     8192,
		InputPrice:    0.04,
		OutputPrice:   0.04,
		Features: []gollmx.Feature{
			gollmx.FeatureChat,
			gollmx.FeatureStreaming,
			gollmx.FeatureSystemPrompt,
		},
		ReleaseDate: "2024-10-16",
	},
	// Free models
	{
		ID:            "pixtral-12b-2409",
		Name:          "Pixtral 12B",
		Provider:      ProviderID,
		Description:   "Multimodal model with vision capabilities",
		ContextWindow: 128000,
		MaxOutput:     8192,
		InputPrice:    0.15,
		OutputPrice:   0.15,
		Features: []gollmx.Feature{
			gollmx.FeatureChat,
			gollmx.FeatureStreaming,
			gollmx.FeatureVision,
			gollmx.FeatureSystemPrompt,
		},
		ReleaseDate: "2024-09-12",
	},
	{
		ID:            "open-mistral-nemo",
		Name:          "Mistral Nemo",
		Provider:      ProviderID,
		Description:   "Open-weight model for research",
		ContextWindow: 128000,
		MaxOutput:     8192,
		InputPrice:    0.15,
		OutputPrice:   0.15,
		Features: []gollmx.Feature{
			gollmx.FeatureChat,
			gollmx.FeatureStreaming,
			gollmx.FeatureTools,
			gollmx.FeatureSystemPrompt,
		},
		ReleaseDate: "2024-07-18",
	},
	// Embedding model
	{
		ID:            "mistral-embed",
		Name:          "Mistral Embed",
		Provider:      ProviderID,
		Description:   "Text embedding model",
		ContextWindow: 8192,
		InputPrice:    0.10,
		Features: []gollmx.Feature{
			gollmx.FeatureEmbedding,
		},
		ReleaseDate: "2024-01-15",
	},
}
