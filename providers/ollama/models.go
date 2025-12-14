package ollama

import gollmx "github.com/onlyhyde/gollm-x"

// defaultModels lists common Ollama models
// Note: Actual available models depend on what's installed locally
var defaultModels = []gollmx.Model{
	// Llama 3 series
	{
		ID:            "llama3.2",
		Name:          "Llama 3.2",
		Description:   "Meta's Llama 3.2 - Latest lightweight model",
		ContextWindow: 128000,
		MaxOutput:     4096,
		InputPrice:    0,
		OutputPrice:   0,
	},
	{
		ID:            "llama3.1",
		Name:          "Llama 3.1",
		Description:   "Meta's Llama 3.1 model",
		ContextWindow: 128000,
		MaxOutput:     4096,
		InputPrice:    0,
		OutputPrice:   0,
	},
	{
		ID:            "llama3",
		Name:          "Llama 3",
		Description:   "Meta's Llama 3 8B model",
		ContextWindow: 8192,
		MaxOutput:     4096,
		InputPrice:    0,
		OutputPrice:   0,
	},

	// Mistral series
	{
		ID:            "mistral",
		Name:          "Mistral 7B",
		Description:   "Mistral AI's 7B parameter model",
		ContextWindow: 32768,
		MaxOutput:     4096,
		InputPrice:    0,
		OutputPrice:   0,
	},
	{
		ID:            "mixtral",
		Name:          "Mixtral 8x7B",
		Description:   "Mistral AI's Mixture of Experts model",
		ContextWindow: 32768,
		MaxOutput:     4096,
		InputPrice:    0,
		OutputPrice:   0,
	},

	// Code models
	{
		ID:            "codellama",
		Name:          "Code Llama",
		Description:   "Meta's Code Llama for programming",
		ContextWindow: 16384,
		MaxOutput:     4096,
		InputPrice:    0,
		OutputPrice:   0,
	},
	{
		ID:            "deepseek-coder",
		Name:          "DeepSeek Coder",
		Description:   "DeepSeek's code generation model",
		ContextWindow: 16384,
		MaxOutput:     4096,
		InputPrice:    0,
		OutputPrice:   0,
	},
	{
		ID:            "qwen2.5-coder",
		Name:          "Qwen 2.5 Coder",
		Description:   "Alibaba's Qwen 2.5 coding model",
		ContextWindow: 32768,
		MaxOutput:     4096,
		InputPrice:    0,
		OutputPrice:   0,
	},

	// Vision models
	{
		ID:            "llava",
		Name:          "LLaVA",
		Description:   "Large Language and Vision Assistant",
		ContextWindow: 4096,
		MaxOutput:     4096,
		InputPrice:    0,
		OutputPrice:   0,
	},
	{
		ID:            "llama3.2-vision",
		Name:          "Llama 3.2 Vision",
		Description:   "Meta's Llama 3.2 with vision capabilities",
		ContextWindow: 128000,
		MaxOutput:     4096,
		InputPrice:    0,
		OutputPrice:   0,
	},

	// Embedding models
	{
		ID:            "nomic-embed-text",
		Name:          "Nomic Embed Text",
		Description:   "Nomic AI's text embedding model",
		ContextWindow: 8192,
		MaxOutput:     0,
		InputPrice:    0,
		OutputPrice:   0,
	},
	{
		ID:            "mxbai-embed-large",
		Name:          "MixedBread Embed Large",
		Description:   "MixedBread AI's embedding model",
		ContextWindow: 512,
		MaxOutput:     0,
		InputPrice:    0,
		OutputPrice:   0,
	},

	// Other popular models
	{
		ID:            "phi3",
		Name:          "Phi-3",
		Description:   "Microsoft's Phi-3 small language model",
		ContextWindow: 4096,
		MaxOutput:     4096,
		InputPrice:    0,
		OutputPrice:   0,
	},
	{
		ID:            "gemma2",
		Name:          "Gemma 2",
		Description:   "Google's Gemma 2 open model",
		ContextWindow: 8192,
		MaxOutput:     4096,
		InputPrice:    0,
		OutputPrice:   0,
	},
	{
		ID:            "qwen2.5",
		Name:          "Qwen 2.5",
		Description:   "Alibaba's Qwen 2.5 model",
		ContextWindow: 32768,
		MaxOutput:     4096,
		InputPrice:    0,
		OutputPrice:   0,
	},
}
