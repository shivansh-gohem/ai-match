package models

// Embedding represents a vector embedding for a user or project profile.
type Embedding struct {
	ID        string    `json:"id"`
	SourceID  string    `json:"source_id"`  // User ID or Project ID
	SourceType string   `json:"source_type"` // "user" or "project"
	Text      string    `json:"text"`        // Original text that was embedded
	Vector    []float32 `json:"vector"`      // The embedding vector (768 dimensions for text-embedding-004)
}

// SimilarityResult represents a vector similarity search result.
type SimilarityResult struct {
	SourceID   string  `json:"source_id"`
	SourceType string  `json:"source_type"`
	Score      float64 `json:"score"` // Cosine similarity score (0.0 to 1.0)
}

// ConnectionSuggestion represents an auto-detected connection between developers
// working on related projects/tech.
type ConnectionSuggestion struct {
	User1      User    `json:"user1"`
	User2      User    `json:"user2"`
	Reason     string  `json:"reason"`
	Similarity float64 `json:"similarity"`
	SharedTech []string `json:"shared_tech"`
}
