package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"github.com/shiva/ai-match/internal/models"
)

// EmbeddingService handles vector embedding generation via Gemini.
type EmbeddingService struct {
	client *genai.Client
	model  *genai.EmbeddingModel
}

// NewEmbeddingService creates a new embedding service using Gemini's text-embedding-004.
func NewEmbeddingService() *EmbeddingService {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" || apiKey == "your-gemini-api-key-here" {
		log.Println("⚠️  GEMINI_API_KEY not set or is placeholder — embedding features will be disabled")
		return &EmbeddingService{}
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Printf("⚠️  Failed to create Gemini client for embeddings: %v", err)
		return &EmbeddingService{}
	}

	model := client.EmbeddingModel("text-embedding-004")

	return &EmbeddingService{
		client: client,
		model:  model,
	}
}

// Close cleans up the embedding client.
func (e *EmbeddingService) Close() {
	if e.client != nil {
		e.client.Close()
	}
}

// IsAvailable returns whether the embedding service is ready.
func (e *EmbeddingService) IsAvailable() bool {
	return e.client != nil && e.model != nil
}

// GenerateEmbedding converts text into a 768-dimensional vector using Gemini.
func (e *EmbeddingService) GenerateEmbedding(text string) ([]float32, error) {
	if !e.IsAvailable() {
		return nil, fmt.Errorf("embedding service not available")
	}

	ctx := context.Background()
	res, err := e.model.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	if res.Embedding == nil {
		return nil, fmt.Errorf("empty embedding response")
	}

	return res.Embedding.Values, nil
}

// EmbedUserProfile creates a rich text representation of a user and embeds it.
func (e *EmbeddingService) EmbedUserProfile(user models.User) ([]float32, string, error) {
	// Build a rich text representation for embedding
	text := fmt.Sprintf(
		"Developer: %s. Bio: %s. Skills: %s. Interests: %s. Location: %s.",
		user.Username,
		user.Bio,
		strings.Join(user.Skills, ", "),
		strings.Join(user.Interests, ", "),
		user.Location,
	)

	vector, err := e.GenerateEmbedding(text)
	if err != nil {
		return nil, text, err
	}

	return vector, text, nil
}

// EmbedProjectProfile creates a rich text representation of a project and embeds it.
func (e *EmbeddingService) EmbedProjectProfile(project models.Project) ([]float32, string, error) {
	text := fmt.Sprintf(
		"Project: %s. Description: %s. Tech Stack: %s. Status: %s.",
		project.Title,
		project.Description,
		strings.Join(project.TechStack, ", "),
		project.Status,
	)

	vector, err := e.GenerateEmbedding(text)
	if err != nil {
		return nil, text, err
	}

	return vector, text, nil
}

// EmbedQuery embeds a free-text search query.
func (e *EmbeddingService) EmbedQuery(query string) ([]float32, error) {
	return e.GenerateEmbedding(query)
}
