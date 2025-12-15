// Package main demonstrates embedding generation and similarity search using gollm-x
package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"sort"
	"time"

	gollmx "github.com/onlyhyde/gollm-x"
	_ "github.com/onlyhyde/gollm-x/providers" // Import all providers
)

// Document represents a text document with its embedding
type Document struct {
	ID        int
	Text      string
	Embedding []float64
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// SearchResult represents a search result with similarity score
type SearchResult struct {
	Document   Document
	Similarity float64
}

func main() {
	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY not set, using Ollama for local embeddings")
		runOllamaExample()
		return
	}

	runOpenAIExample(apiKey)
}

func runOpenAIExample(apiKey string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Create OpenAI client
	client, err := gollmx.New("openai", gollmx.WithAPIKey(apiKey))
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		return
	}

	// Wrap with retry logic
	client = gollmx.WithRetry(client,
		gollmx.WithRetryMaxRetries(3),
		gollmx.WithRetryInitialDelay(1*time.Second),
	)

	fmt.Println("=== OpenAI Embeddings Example ===")
	fmt.Printf("Provider: %s\n\n", client.Name())

	// Sample documents for semantic search
	documents := []string{
		"The quick brown fox jumps over the lazy dog.",
		"Machine learning is a subset of artificial intelligence.",
		"Python is a popular programming language for data science.",
		"Natural language processing enables computers to understand human language.",
		"Deep learning models require large amounts of training data.",
		"The capital of France is Paris.",
		"Climate change is affecting global weather patterns.",
		"Quantum computing uses quantum mechanics for computation.",
	}

	// Generate embeddings for all documents
	fmt.Println("Generating embeddings for documents...")
	docs, err := embedDocuments(ctx, client, documents, "text-embedding-3-small")
	if err != nil {
		fmt.Printf("Failed to generate embeddings: %v\n", err)
		return
	}
	fmt.Printf("Generated embeddings for %d documents\n\n", len(docs))

	// Search queries
	queries := []string{
		"AI and machine learning",
		"Programming languages",
		"European capitals",
	}

	for _, query := range queries {
		fmt.Printf("Query: \"%s\"\n", query)
		fmt.Println("---")

		results, err := semanticSearch(ctx, client, query, docs, "text-embedding-3-small", 3)
		if err != nil {
			fmt.Printf("Search failed: %v\n", err)
			continue
		}

		for i, result := range results {
			fmt.Printf("%d. [%.4f] %s\n", i+1, result.Similarity, result.Document.Text)
		}
		fmt.Println()
	}

	// Demonstrate batch embeddings
	fmt.Println("=== Batch Embedding Example ===")
	batchTexts := []string{
		"Hello, world!",
		"How are you today?",
		"The weather is nice.",
	}

	resp, err := client.Embed(ctx, &gollmx.EmbedRequest{
		Model: "text-embedding-3-small",
		Input: batchTexts,
	})
	if err != nil {
		fmt.Printf("Batch embedding failed: %v\n", err)
		return
	}

	fmt.Printf("Generated %d embeddings in batch\n", len(resp.Embeddings))
	fmt.Printf("Embedding dimensions: %d\n", len(resp.Embeddings[0].Vector))
	fmt.Printf("Usage - Total tokens: %d\n", resp.Usage.TotalTokens)
}

func runOllamaExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Create Ollama client
	client, err := gollmx.New("ollama")
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		return
	}

	fmt.Println("=== Ollama Embeddings Example ===")
	fmt.Printf("Provider: %s\n", client.Name())
	fmt.Println("Note: Ollama embeddings work best with 'nomic-embed-text' model")
	fmt.Println("Install with: ollama pull nomic-embed-text")
	fmt.Println()

	// Check if embedding is supported
	if !client.HasFeature(gollmx.FeatureEmbedding) {
		fmt.Println("Embedding feature not supported")
		return
	}

	// Simple embedding example
	text := "Hello, this is a test sentence for embedding."
	fmt.Printf("Embedding text: \"%s\"\n", text)

	resp, err := client.Embed(ctx, &gollmx.EmbedRequest{
		Model: "nomic-embed-text",
		Input: []string{text},
	})
	if err != nil {
		fmt.Printf("Embedding failed: %v\n", err)
		fmt.Println("Make sure you have nomic-embed-text model installed")
		return
	}

	fmt.Printf("Embedding dimensions: %d\n", len(resp.Embeddings[0].Vector))
	fmt.Printf("First 5 values: %v\n", resp.Embeddings[0].Vector[:5])

	// Similarity comparison
	fmt.Println("\n=== Similarity Comparison ===")
	texts := []string{
		"Machine learning is fascinating",
		"Deep learning is a type of machine learning",
		"I love pizza",
	}

	embeddings := make([][]float64, len(texts))
	for i, t := range texts {
		resp, err := client.Embed(ctx, &gollmx.EmbedRequest{
			Model: "nomic-embed-text",
			Input: []string{t},
		})
		if err != nil {
			fmt.Printf("Failed to embed text %d: %v\n", i, err)
			return
		}
		embeddings[i] = resp.Embeddings[0].Vector
	}

	fmt.Println("Text similarities:")
	for i := 0; i < len(texts); i++ {
		for j := i + 1; j < len(texts); j++ {
			sim := cosineSimilarity(embeddings[i], embeddings[j])
			fmt.Printf("  \"%s\" <-> \"%s\": %.4f\n", texts[i], texts[j], sim)
		}
	}
}

// embedDocuments generates embeddings for a list of documents
func embedDocuments(ctx context.Context, client gollmx.LLM, texts []string, model string) ([]Document, error) {
	resp, err := client.Embed(ctx, &gollmx.EmbedRequest{
		Model: model,
		Input: texts,
	})
	if err != nil {
		return nil, err
	}

	docs := make([]Document, len(texts))
	for i, text := range texts {
		docs[i] = Document{
			ID:        i,
			Text:      text,
			Embedding: resp.Embeddings[i].Vector,
		}
	}

	return docs, nil
}

// semanticSearch finds the most similar documents to a query
func semanticSearch(ctx context.Context, client gollmx.LLM, query string, docs []Document, model string, topK int) ([]SearchResult, error) {
	// Get embedding for query
	resp, err := client.Embed(ctx, &gollmx.EmbedRequest{
		Model: model,
		Input: []string{query},
	})
	if err != nil {
		return nil, err
	}

	queryEmbedding := resp.Embeddings[0].Vector

	// Calculate similarity with all documents
	results := make([]SearchResult, len(docs))
	for i, doc := range docs {
		results[i] = SearchResult{
			Document:   doc,
			Similarity: cosineSimilarity(queryEmbedding, doc.Embedding),
		}
	}

	// Sort by similarity (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	// Return top K results
	if topK > len(results) {
		topK = len(results)
	}

	return results[:topK], nil
}
