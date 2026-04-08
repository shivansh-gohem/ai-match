package service

import (
	"context"
	"fmt"
	"github.com/shiva/ai-match/pkg/logger"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"github.com/shiva/ai-match/internal/models"
)

// GeminiService handles all interactions with the Google Gemini API.
type GeminiService struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

// NewGeminiService initializes the Gemini client.
func NewGeminiService() *GeminiService {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" || apiKey == "your-gemini-api-key-here" {
		logger.Println("⚠️  GEMINI_API_KEY not set or is placeholder — AI features will use fallback mode")
		return &GeminiService{}
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		logger.Printf("⚠️  Failed to create Gemini client: %v — using fallback mode", err)
		return &GeminiService{}
	}

	model := client.GenerativeModel("gemini-2.5-flash")
	model.SetTemperature(0.7)
	model.SetTopP(0.9)
	model.SystemInstruction = genai.NewUserContent(genai.Text(
		`You are DevConnect AI, an intelligent assistant embedded in a developer collaboration platform. 
You help developers find the right collaborators, suggest technologies, discuss tech topics, and provide 
expert advice on software engineering. Be concise, friendly, and technically accurate. 
Use markdown formatting in your responses when helpful.`))

	return &GeminiService{
		client: client,
		model:  model,
	}
}

// Close cleans up the Gemini client.
func (g *GeminiService) Close() {
	if g.client != nil {
		g.client.Close()
	}
}

// IsAvailable returns whether the Gemini API is configured.
func (g *GeminiService) IsAvailable() bool {
	return g.client != nil
}

// GenerateMatchSuggestion uses Gemini to generate intelligent match reasoning.
func (g *GeminiService) GenerateMatchSuggestion(user models.User, candidates []models.User) ([]models.MatchResult, error) {
	if !g.IsAvailable() {
		return g.fallbackMatch(user, candidates), nil
	}

	// Build the prompt
	candidateDescriptions := ""
	for i, c := range candidates {
		candidateDescriptions += fmt.Sprintf(
			"%d. **%s** (Location: %s)\n   Skills: %s\n   Interests: %s\n   Bio: %s\n\n",
			i+1, c.Username, c.Location,
			strings.Join(c.Skills, ", "),
			strings.Join(c.Interests, ", "),
			c.Bio,
		)
	}

	prompt := fmt.Sprintf(
		`A developer named "%s" has the following profile:
- Skills: %s
- Interests: %s
- Bio: %s

Here are potential collaborators:
%s

For each candidate, provide:
1. A compatibility score from 0.0 to 1.0
2. A brief reason (one sentence) why they are a good match.

Format your response as:
CANDIDATE_NAME|SCORE|REASON
(one per line, no extra text)`,
		user.Username,
		strings.Join(user.Skills, ", "),
		strings.Join(user.Interests, ", "),
		user.Bio,
		candidateDescriptions,
	)

	ctx := context.Background()
	resp, err := g.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		logger.Printf("Gemini API error: %v — falling back", err)
		return g.fallbackMatch(user, candidates), nil
	}

	// Parse response
	results := g.parseMatchResponse(resp, candidates)
	if len(results) == 0 {
		return g.fallbackMatch(user, candidates), nil
	}
	return results, nil
}

// ChatWithAI sends a free-form prompt to Gemini and returns a response.
func (g *GeminiService) ChatWithAI(prompt string) (string, error) {
	if !g.IsAvailable() {
		return "🤖 AI is currently in offline mode. Please set your GEMINI_API_KEY to enable AI features.", nil
	}

	ctx := context.Background()
	resp, err := g.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("gemini API error: %w", err)
	}

	// Extract text from response
	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		return fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]), nil
	}

	return "I couldn't generate a response. Please try again.", nil
}

// parseMatchResponse parses Gemini's structured match output.
func (g *GeminiService) parseMatchResponse(resp *genai.GenerateContentResponse, candidates []models.User) []models.MatchResult {
	var results []models.MatchResult

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return results
	}

	text := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	lines := strings.Split(text, "\n")

	candidateMap := make(map[string]models.User)
	for _, c := range candidates {
		candidateMap[strings.ToLower(c.Username)] = c
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		scoreStr := strings.TrimSpace(parts[1])
		reason := strings.TrimSpace(parts[2])

		var score float64
		fmt.Sscanf(scoreStr, "%f", &score)

		if user, ok := candidateMap[strings.ToLower(name)]; ok {
			results = append(results, models.MatchResult{
				User:   user,
				Score:  score,
				Reason: reason,
			})
		}
	}

	return results
}

// fallbackMatch provides basic skill-overlap matching when Gemini is unavailable.
func (g *GeminiService) fallbackMatch(user models.User, candidates []models.User) []models.MatchResult {
	var results []models.MatchResult

	userSkills := make(map[string]bool)
	for _, s := range user.Skills {
		userSkills[strings.ToLower(s)] = true
	}
	for _, i := range user.Interests {
		userSkills[strings.ToLower(i)] = true
	}

	for _, c := range candidates {
		overlap := 0
		total := len(userSkills)
		var matched []string

		for _, s := range c.Skills {
			if userSkills[strings.ToLower(s)] {
				overlap++
				matched = append(matched, s)
			}
		}
		for _, i := range c.Interests {
			if userSkills[strings.ToLower(i)] {
				overlap++
				matched = append(matched, i)
			}
		}

		if overlap > 0 && total > 0 {
			score := float64(overlap) / float64(total)
			if score > 1.0 {
				score = 1.0
			}
			reason := fmt.Sprintf("Shares expertise in %s", strings.Join(matched, ", "))
			results = append(results, models.MatchResult{
				User:   c,
				Score:  score,
				Reason: reason,
			})
		}
	}

	return results
}
