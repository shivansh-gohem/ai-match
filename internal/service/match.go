package service

import (
	"fmt"
	"github.com/shiva/ai-match/pkg/logger"
	"sort"
	"strings"

	"github.com/shiva/ai-match/internal/models"
	"github.com/shiva/ai-match/internal/repository"
)

// MatchService handles developer matchmaking logic using the RAG pipeline.
type MatchService struct {
	fakeDB    *repository.FakeDB
	pgDB      *repository.PostgresDB
	gemini    *GeminiService
	embedder  *EmbeddingService
	usePostgres bool
}

// NewMatchService creates a new match service.
func NewMatchService(fakeDB *repository.FakeDB, pgDB *repository.PostgresDB, gemini *GeminiService, embedder *EmbeddingService) *MatchService {
	return &MatchService{
		fakeDB:    fakeDB,
		pgDB:      pgDB,
		gemini:    gemini,
		embedder:  embedder,
		usePostgres: pgDB != nil,
	}
}

// ═══════════════════════════════════════════════
// RAG PIPELINE: Embed → Retrieve → Augment → Generate
// ═══════════════════════════════════════════════

// FindMatches finds the best developer matches for a given user using the full RAG pipeline.
func (m *MatchService) FindMatches(userID string) ([]models.MatchResult, error) {
	// ─── Step 0: Get the user ───
	var user models.User
	var found bool

	if m.usePostgres {
		var err error
		user, found, err = m.pgDB.GetUserByID(userID)
		if err != nil {
			return nil, err
		}
	} else {
		user, found = m.fakeDB.GetUserByID(userID)
	}

	if !found {
		return nil, nil
	}

	// ─── RAG STEP 1: EMBED — Convert user profile to vector ───
	if m.usePostgres && m.embedder.IsAvailable() {
		return m.ragMatchUser(user)
	}

	// ─── Fallback: Skill-overlap matching (when no PostgreSQL or no embeddings) ───
	return m.fallbackMatchUser(user)
}

// ragMatchUser implements the full RAG pipeline for user matching.
func (m *MatchService) ragMatchUser(user models.User) ([]models.MatchResult, error) {
	logger.Printf("🧠 RAG Pipeline: Starting match for user %s", user.Username)

	// ─── STEP 1: EMBED the query user's profile ───
	queryVector, _, err := m.embedder.EmbedUserProfile(user)
	if err != nil {
		logger.Printf("⚠️  Embedding failed for user %s: %v — falling back", user.Username, err)
		return m.fallbackMatchUser(user)
	}
	logger.Printf("📐 Generated embedding vector of %d dimensions", len(queryVector))

	// ─── STEP 2: RETRIEVE — Use pgvector to find similar developers ───
	similarResults, err := m.pgDB.SearchSimilar(queryVector, "user", user.ID, 10)
	if err != nil {
		logger.Printf("⚠️  pgvector search failed: %v — falling back", err)
		return m.fallbackMatchUser(user)
	}
	logger.Printf("🔍 Retrieved %d similar developers from pgvector", len(similarResults))

	if len(similarResults) == 0 {
		return []models.MatchResult{}, nil
	}

	// Fetch full user profiles for the retrieved IDs
	var candidates []models.User
	var scores []float64
	for _, r := range similarResults {
		if candidate, found, err := m.pgDB.GetUserByID(r.SourceID); err == nil && found {
			candidates = append(candidates, candidate)
			scores = append(scores, r.Score)
		}
	}

	// ─── STEP 3: AUGMENT + GENERATE — Use Gemini LLM with retrieved context ───
	if m.gemini.IsAvailable() && len(candidates) > 0 {
		results, err := m.gemini.GenerateMatchSuggestion(user, candidates)
		if err == nil && len(results) > 0 {
			// Merge pgvector scores with LLM reasoning
			for i := range results {
				if i < len(scores) {
					// Blend: 60% vector similarity + 40% LLM score
					results[i].Score = 0.6*scores[i] + 0.4*results[i].Score
				}
			}
			sort.Slice(results, func(i, j int) bool {
				return results[i].Score > results[j].Score
			})
			if len(results) > 5 {
				results = results[:5]
			}
			logger.Printf("✅ RAG Pipeline complete — %d matches generated", len(results))
			return results, nil
		}
	}

	// If LLM fails, use pgvector scores directly
	var results []models.MatchResult
	for i, c := range candidates {
		score := 0.0
		if i < len(scores) {
			score = scores[i]
		}
		results = append(results, models.MatchResult{
			User:   c,
			Score:  score,
			Reason: fmt.Sprintf("%.0f%% profile similarity based on skills and interests", score*100),
		})
	}

	if len(results) > 5 {
		results = results[:5]
	}
	return results, nil
}

// FindMatchesForProject finds developers that match a project using RAG.
func (m *MatchService) FindMatchesForProject(projectID string) ([]models.MatchResult, error) {
	var project models.Project
	var found bool

	if m.usePostgres {
		var err error
		project, found, err = m.pgDB.GetProjectByID(projectID)
		if err != nil {
			return nil, err
		}
	} else {
		project, found = m.fakeDB.GetProjectByID(projectID)
	}

	if !found {
		return nil, nil
	}

	// RAG pipeline for project matching
	if m.usePostgres && m.embedder.IsAvailable() {
		return m.ragMatchProject(project)
	}

	return m.fallbackMatchProject(project)
}

// ragMatchProject implements the RAG pipeline for project-to-developer matching.
func (m *MatchService) ragMatchProject(project models.Project) ([]models.MatchResult, error) {
	logger.Printf("🧠 RAG Pipeline: Finding contributors for project '%s'", project.Title)

	// ─── STEP 1: EMBED the project description ───
	queryVector, _, err := m.embedder.EmbedProjectProfile(project)
	if err != nil {
		logger.Printf("⚠️  Embedding failed for project: %v — falling back", err)
		return m.fallbackMatchProject(project)
	}

	// ─── STEP 2: RETRIEVE — Search for similar user embeddings ───
	similarResults, err := m.pgDB.SearchSimilar(queryVector, "user", project.OwnerID, 10)
	if err != nil {
		return m.fallbackMatchProject(project)
	}

	if len(similarResults) == 0 {
		return []models.MatchResult{}, nil
	}

	var candidates []models.User
	var scores []float64
	for _, r := range similarResults {
		if candidate, found, err := m.pgDB.GetUserByID(r.SourceID); err == nil && found {
			candidates = append(candidates, candidate)
			scores = append(scores, r.Score)
		}
	}

	// ─── STEP 3: AUGMENT + GENERATE ───
	projectUser := models.User{
		Username:  "Project: " + project.Title,
		Skills:    project.TechStack,
		Interests: project.TechStack,
		Bio:       project.Description,
	}

	if m.gemini.IsAvailable() && len(candidates) > 0 {
		results, err := m.gemini.GenerateMatchSuggestion(projectUser, candidates)
		if err == nil && len(results) > 0 {
			for i := range results {
				if i < len(scores) {
					results[i].Score = 0.6*scores[i] + 0.4*results[i].Score
				}
			}
			sort.Slice(results, func(i, j int) bool {
				return results[i].Score > results[j].Score
			})
			if len(results) > 5 {
				results = results[:5]
			}
			return results, nil
		}
	}

	var results []models.MatchResult
	for i, c := range candidates {
		score := 0.0
		if i < len(scores) {
			score = scores[i]
		}
		results = append(results, models.MatchResult{
			User:   c,
			Score:  score,
			Reason: fmt.Sprintf("%.0f%% match for this project's tech requirements", score*100),
		})
	}
	if len(results) > 5 {
		results = results[:5]
	}
	return results, nil
}

// FindConnections discovers developers working on related things.
func (m *MatchService) FindConnections(userID string) ([]models.ConnectionSuggestion, error) {
	if !m.usePostgres {
		return m.fallbackConnections(userID)
	}

	user, found, err := m.pgDB.GetUserByID(userID)
	if err != nil || !found {
		return nil, err
	}

	related, err := m.pgDB.FindRelatedDevelopers(userID, 5)
	if err != nil {
		return m.fallbackConnections(userID)
	}

	var suggestions []models.ConnectionSuggestion
	for _, r := range related {
		otherUser, found, err := m.pgDB.GetUserByID(r.SourceID)
		if err != nil || !found {
			continue
		}

		// Find shared technologies
		shared := findSharedItems(user.Skills, otherUser.Skills)
		sharedInterests := findSharedItems(user.Interests, otherUser.Interests)
		shared = append(shared, sharedInterests...)

		reason := fmt.Sprintf("Both work with %s — %.0f%% profile similarity",
			strings.Join(shared, ", "), r.Score*100)

		suggestions = append(suggestions, models.ConnectionSuggestion{
			User1:      user,
			User2:      otherUser,
			Reason:     reason,
			Similarity: r.Score,
			SharedTech: shared,
		})
	}

	return suggestions, nil
}

// EmbedAllProfiles generates and stores embeddings for all users and projects.
// Called during startup to populate the vector database.
func (m *MatchService) EmbedAllProfiles() error {
	if !m.usePostgres || !m.embedder.IsAvailable() {
		logger.Println("⚠️  Skipping embedding — PostgreSQL or embeddings not available")
		return nil
	}

	logger.Println("🧠 Generating embeddings for all profiles...")

	// Embed all users
	users, err := m.pgDB.ListUsers()
	if err != nil {
		return err
	}

	for _, user := range users {
		vector, text, err := m.embedder.EmbedUserProfile(user)
		if err != nil {
			logger.Printf("⚠️  Failed to embed user %s: %v", user.Username, err)
			continue
		}

		err = m.pgDB.StoreEmbedding(models.Embedding{
			SourceID:   user.ID,
			SourceType: "user",
			Text:       text,
			Vector:     vector,
		})
		if err != nil {
			logger.Printf("⚠️  Failed to store embedding for %s: %v", user.Username, err)
			continue
		}
		logger.Printf("  ✅ Embedded: %s", user.Username)
	}

	// Embed all projects
	projects, err := m.pgDB.ListProjects()
	if err != nil {
		return err
	}

	for _, project := range projects {
		vector, text, err := m.embedder.EmbedProjectProfile(project)
		if err != nil {
			logger.Printf("⚠️  Failed to embed project %s: %v", project.Title, err)
			continue
		}

		err = m.pgDB.StoreEmbedding(models.Embedding{
			SourceID:   project.ID,
			SourceType: "project",
			Text:       text,
			Vector:     vector,
		})
		if err != nil {
			logger.Printf("⚠️  Failed to store embedding for project %s: %v", project.Title, err)
			continue
		}
		logger.Printf("  ✅ Embedded: %s", project.Title)
	}

	logger.Printf("🧠 Embedding complete — %d users, %d projects", len(users), len(projects))
	return nil
}

// ═══════════════════ FALLBACK METHODS ═══════════════════

func (m *MatchService) fallbackMatchUser(user models.User) ([]models.MatchResult, error) {
	candidates := m.fakeDB.FindMatchingUsers(user.Skills, user.ID)
	interestCandidates := m.fakeDB.FindMatchingUsers(user.Interests, user.ID)
	
	candidateMap := make(map[string]bool)
	for _, c := range candidates {
		candidateMap[c.ID] = true
	}
	for _, c := range interestCandidates {
		if !candidateMap[c.ID] {
			candidates = append(candidates, c)
		}
	}

	if len(candidates) == 0 {
		return []models.MatchResult{}, nil
	}
	if len(candidates) > 10 {
		candidates = candidates[:10]
	}

	results, err := m.gemini.GenerateMatchSuggestion(user, candidates)
	if err != nil {
		return nil, err
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if len(results) > 5 {
		results = results[:5]
	}
	return results, nil
}

func (m *MatchService) fallbackMatchProject(project models.Project) ([]models.MatchResult, error) {
	candidates := m.fakeDB.FindMatchingUsers(project.TechStack, project.OwnerID)
	if len(candidates) == 0 {
		return []models.MatchResult{}, nil
	}
	if len(candidates) > 10 {
		candidates = candidates[:10]
	}

	projectUser := models.User{
		Username:  "Project: " + project.Title,
		Skills:    project.TechStack,
		Interests: project.TechStack,
		Bio:       project.Description,
	}

	results, err := m.gemini.GenerateMatchSuggestion(projectUser, candidates)
	if err != nil {
		return nil, err
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if len(results) > 5 {
		results = results[:5]
	}
	return results, nil
}

func (m *MatchService) fallbackConnections(userID string) ([]models.ConnectionSuggestion, error) {
	user, found := m.fakeDB.GetUserByID(userID)
	if !found {
		return nil, nil
	}

	candidates := m.fakeDB.FindMatchingUsers(user.Skills, userID)
	var suggestions []models.ConnectionSuggestion
	for _, c := range candidates {
		shared := findSharedItems(user.Skills, c.Skills)
		if len(shared) > 0 {
			suggestions = append(suggestions, models.ConnectionSuggestion{
				User1:      user,
				User2:      c,
				Reason:     fmt.Sprintf("Both work with %s", strings.Join(shared, ", ")),
				Similarity: float64(len(shared)) / float64(len(user.Skills)),
				SharedTech: shared,
			})
		}
	}
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}
	return suggestions, nil
}

// ═══════════════════ HELPERS ═══════════════════

func findSharedItems(a, b []string) []string {
	set := make(map[string]bool)
	for _, s := range a {
		set[strings.ToLower(s)] = true
	}
	var shared []string
	seen := make(map[string]bool)
	for _, s := range b {
		if set[strings.ToLower(s)] && !seen[strings.ToLower(s)] {
			shared = append(shared, s)
			seen[strings.ToLower(s)] = true
		}
	}
	return shared
}
