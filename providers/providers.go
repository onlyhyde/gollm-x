// Package providers imports all built-in LLM providers.
// Import this package to register all providers at once:
//
//	import _ "github.com/onlyhyde/gollm-x/providers"
package providers

import (
	// Import all providers to trigger their init() functions
	_ "github.com/onlyhyde/gollm-x/providers/anthropic"
	_ "github.com/onlyhyde/gollm-x/providers/google"
	_ "github.com/onlyhyde/gollm-x/providers/openai"
)
