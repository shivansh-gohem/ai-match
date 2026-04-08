package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/shiva/ai-match/internal/models"
	"github.com/shiva/ai-match/pkg/logger"
)

// GroqService handles AI chat via Groq's OpenAI-compatible API.
type GroqService struct {
	apiKey    string
	available bool
}

// groqRequest is the request body for Groq chat completions.
type groqRequest struct {
	Model    string        `json:"model"`
	Messages []groqMessage `json:"messages"`
}

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// groqResponse is the response from Groq API.
type groqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// NewGroqService creates a new Groq service.
func NewGroqService() *GroqService {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		logger.Println("⚠️  GROQ_API_KEY not set — AI Assistant will use fallback mode")
		return &GroqService{available: false}
	}

	logger.Println("✅ Groq AI service initialized successfully")
	return &GroqService{
		apiKey:    apiKey,
		available: true,
	}
}

// IsAvailable returns whether the Groq API is configured.
func (g *GroqService) IsAvailable() bool {
	return g.available
}

// buildSystemPrompt creates a system prompt with all DevConnect context.
func (g *GroqService) buildSystemPrompt(users []models.User, projects []models.Project) string {
	var sb strings.Builder

	sb.WriteString(`You are the DevConnect AI Assistant — an intelligent helper embedded in the DevConnect developer collaboration platform.

IMPORTANT RULES:
1. You ONLY answer questions related to DevConnect, its developers, and its projects.
2. If someone asks about things NOT related to DevConnect (e.g., general knowledge, math, other topics), politely redirect them:
   "I'm the DevConnect AI Assistant! I can only help with questions about developers and projects on our platform. Try asking me things like: 'Who works with Golang?', 'What projects use Kubernetes?', or 'Tell me about arjun_dev's skills.'"
3. Be friendly, concise, and use emojis when appropriate.
4. Format lists nicely with bullet points.
5. When mentioning developers, include their skills and location.

Here is the complete data about all developers and projects on the DevConnect platform:

=== DEVELOPERS ===
`)

	for _, u := range users {
		sb.WriteString(fmt.Sprintf("• **%s** (ID: %s)\n  Location: %s\n  Skills: %s\n  Interests: %s\n  Bio: %s\n  GitHub: %s\n\n",
			u.Username, u.ID, u.Location,
			strings.Join(u.Skills, ", "),
			strings.Join(u.Interests, ", "),
			u.Bio, u.GithubURL))
	}

	sb.WriteString("\n=== PROJECTS ===\n")

	for _, p := range projects {
		members := "None yet"
		if len(p.Members) > 0 {
			members = strings.Join(p.MemberNames, ", ")
		}
		sb.WriteString(fmt.Sprintf("• **%s** (ID: %s, Status: %s)\n  Description: %s\n  Tech Stack: %s\n  Owner: %s\n  Members (%d/%d): %s\n\n",
			p.Title, p.ID, p.Status,
			p.Description,
			strings.Join(p.TechStack, ", "),
			p.OwnerName,
			len(p.Members), p.MaxMembers,
			members))
	}

	return sb.String()
}

// ChatWithContext sends a message to Groq with full DevConnect context.
func (g *GroqService) ChatWithContext(prompt string, users []models.User, projects []models.Project) (string, error) {
	if !g.available {
		return "🤖 AI Assistant is currently in offline mode. Please set GROQ_API_KEY to enable AI features.", nil
	}

	systemPrompt := g.buildSystemPrompt(users, projects)

	reqBody := groqRequest{
		Model: "llama-3.3-70b-versatile",
		Messages: []groqMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("groq API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var groqResp groqResponse
	if err := json.Unmarshal(body, &groqResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if groqResp.Error != nil {
		return "", fmt.Errorf("groq API error: %s", groqResp.Error.Message)
	}

	if len(groqResp.Choices) > 0 {
		return groqResp.Choices[0].Message.Content, nil
	}

	return "I couldn't generate a response. Please try again!", nil
}
