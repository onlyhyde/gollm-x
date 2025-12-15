package cohere

import gollmx "github.com/onlyhyde/gollm-x"

// CohereModels contains all known Cohere models
var CohereModels = []gollmx.Model{
	// Command R+ series
	{
		ID:            "command-r-plus",
		Name:          "Command R+",
		Provider:      ProviderID,
		Description:   "Most powerful Cohere model for complex tasks",
		ContextWindow: 128000,
		MaxOutput:     4096,
		InputPrice:    2.50,
		OutputPrice:   10.00,
		Features: []gollmx.Feature{
			gollmx.FeatureChat,
			gollmx.FeatureStreaming,
			gollmx.FeatureTools,
			gollmx.FeatureSystemPrompt,
		},
		ReleaseDate: "2024-04-04",
	},
	{
		ID:            "command-r",
		Name:          "Command R",
		Provider:      ProviderID,
		Description:   "Balanced model for various tasks",
		ContextWindow: 128000,
		MaxOutput:     4096,
		InputPrice:    0.15,
		OutputPrice:   0.60,
		Features: []gollmx.Feature{
			gollmx.FeatureChat,
			gollmx.FeatureStreaming,
			gollmx.FeatureTools,
			gollmx.FeatureSystemPrompt,
		},
		ReleaseDate: "2024-03-11",
	},
	// Command series
	{
		ID:            "command",
		Name:          "Command",
		Provider:      ProviderID,
		Description:   "Instruction-following model",
		ContextWindow: 4096,
		MaxOutput:     4096,
		InputPrice:    1.00,
		OutputPrice:   2.00,
		Features: []gollmx.Feature{
			gollmx.FeatureChat,
			gollmx.FeatureStreaming,
			gollmx.FeatureSystemPrompt,
		},
		ReleaseDate: "2023-03-01",
	},
	{
		ID:            "command-light",
		Name:          "Command Light",
		Provider:      ProviderID,
		Description:   "Faster, lighter instruction-following model",
		ContextWindow: 4096,
		MaxOutput:     4096,
		InputPrice:    0.30,
		OutputPrice:   0.60,
		Features: []gollmx.Feature{
			gollmx.FeatureChat,
			gollmx.FeatureStreaming,
			gollmx.FeatureSystemPrompt,
		},
		ReleaseDate: "2023-03-01",
	},
	// Embedding models
	{
		ID:            "embed-english-v3.0",
		Name:          "Embed English v3.0",
		Provider:      ProviderID,
		Description:   "State-of-the-art English embedding model",
		ContextWindow: 512,
		InputPrice:    0.10,
		Features: []gollmx.Feature{
			gollmx.FeatureEmbedding,
		},
		ReleaseDate: "2023-11-02",
	},
	{
		ID:            "embed-multilingual-v3.0",
		Name:          "Embed Multilingual v3.0",
		Provider:      ProviderID,
		Description:   "Multilingual embedding model for 100+ languages",
		ContextWindow: 512,
		InputPrice:    0.10,
		Features: []gollmx.Feature{
			gollmx.FeatureEmbedding,
		},
		ReleaseDate: "2023-11-02",
	},
	{
		ID:            "embed-english-light-v3.0",
		Name:          "Embed English Light v3.0",
		Provider:      ProviderID,
		Description:   "Lighter English embedding model",
		ContextWindow: 512,
		InputPrice:    0.10,
		Features: []gollmx.Feature{
			gollmx.FeatureEmbedding,
		},
		ReleaseDate: "2023-11-02",
	},
	{
		ID:            "embed-multilingual-light-v3.0",
		Name:          "Embed Multilingual Light v3.0",
		Provider:      ProviderID,
		Description:   "Lighter multilingual embedding model",
		ContextWindow: 512,
		InputPrice:    0.10,
		Features: []gollmx.Feature{
			gollmx.FeatureEmbedding,
		},
		ReleaseDate: "2023-11-02",
	},
}
