package repository

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shiva/ai-match/internal/models"
)

// PostgresDB is the PostgreSQL + pgvector backed repository.
type PostgresDB struct {
	pool *pgxpool.Pool
}

// NewPostgresDB creates a new PostgreSQL connection pool and initializes the schema.
func NewPostgresDB() (*PostgresDB, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=disable",
			getEnvOrDefault("DB_USER", "devconnect"),
			getEnvOrDefault("DB_PASSWORD", "devconnect-super-secret-2024"),
			getEnvOrDefault("DB_HOST", "localhost"),
			getEnvOrDefault("DB_PORT", "5432"),
			getEnvOrDefault("DB_NAME", "devconnect_db"),
		)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	db := &PostgresDB{pool: pool}

	// Initialize schema
	if err := db.initSchema(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	log.Println("✅ Connected to PostgreSQL with pgvector")
	return db, nil
}

// Close closes the database connection pool.
func (db *PostgresDB) Close() {
	db.pool.Close()
}

// initSchema creates all tables and enables the pgvector extension.
func (db *PostgresDB) initSchema(ctx context.Context) error {
	schema := `
		-- Enable pgvector extension
		CREATE EXTENSION IF NOT EXISTS vector;

		-- Users table
		CREATE TABLE IF NOT EXISTS users (
			id          TEXT PRIMARY KEY,
			username    TEXT UNIQUE NOT NULL,
			email       TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL DEFAULT '',
			bio         TEXT DEFAULT '',
			skills      TEXT[] DEFAULT '{}',
			interests   TEXT[] DEFAULT '{}',
			github_url  TEXT DEFAULT '',
			avatar_url  TEXT DEFAULT '',
			location    TEXT DEFAULT '',
			created_at  TIMESTAMPTZ DEFAULT NOW()
		);

		-- Add password_hash to existing tables if needed
		ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT NOT NULL DEFAULT '';

		-- Projects table
		CREATE TABLE IF NOT EXISTS projects (
			id          TEXT PRIMARY KEY,
			title       TEXT NOT NULL,
			description TEXT NOT NULL,
			tech_stack  TEXT[] DEFAULT '{}',
			owner_id    TEXT REFERENCES users(id),
			owner_name  TEXT DEFAULT '',
			status      TEXT DEFAULT 'open',
			max_members INT DEFAULT 5,
			created_at  TIMESTAMPTZ DEFAULT NOW()
		);

		-- Chat rooms table
		CREATE TABLE IF NOT EXISTS chat_rooms (
			id           TEXT PRIMARY KEY,
			name         TEXT NOT NULL,
			description  TEXT DEFAULT '',
			participants TEXT[] DEFAULT '{}'
		);

		-- Messages table
		CREATE TABLE IF NOT EXISTS messages (
			id         TEXT PRIMARY KEY,
			sender_id  TEXT NOT NULL,
			username   TEXT NOT NULL,
			content    TEXT NOT NULL,
			room_id    TEXT REFERENCES chat_rooms(id),
			timestamp  TIMESTAMPTZ DEFAULT NOW()
		);

		-- Vector embeddings table (pgvector — 768 dimensions for text-embedding-004)
		CREATE TABLE IF NOT EXISTS embeddings (
			id          TEXT PRIMARY KEY,
			source_id   TEXT NOT NULL,
			source_type TEXT NOT NULL,
			text        TEXT NOT NULL,
			vector      vector(768),
			created_at  TIMESTAMPTZ DEFAULT NOW()
		);

		-- Index for fast vector similarity search
		CREATE INDEX IF NOT EXISTS idx_embeddings_vector 
			ON embeddings USING ivfflat (vector vector_cosine_ops) WITH (lists = 10);

		-- Index for source lookups
		CREATE INDEX IF NOT EXISTS idx_embeddings_source 
			ON embeddings (source_id, source_type);
	`

	_, err := db.pool.Exec(ctx, schema)
	return err
}

// ═══════════════════ USER OPERATIONS ═══════════════════

// CreateUser inserts a new user into PostgreSQL.
func (db *PostgresDB) CreateUser(user models.User) (models.User, error) {
	ctx := context.Background()
	user.ID = generateID()
	user.CreatedAt = time.Now()
	if user.AvatarURL == "" {
		user.AvatarURL = fmt.Sprintf("https://api.dicebear.com/7.x/avataaars/svg?seed=%s", user.Username)
	}

	_, err := db.pool.Exec(ctx,
		`INSERT INTO users (id, username, email, password_hash, bio, skills, interests, github_url, avatar_url, location, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		user.ID, user.Username, user.Email, user.PasswordHash, user.Bio, user.Skills, user.Interests,
		user.GithubURL, user.AvatarURL, user.Location, user.CreatedAt,
	)
	if err != nil {
		return models.User{}, fmt.Errorf("failed to create user: %w", err)
	}
	return user, nil
}

// GetUserByID retrieves a user by ID.
func (db *PostgresDB) GetUserByID(id string) (models.User, bool, error) {
	ctx := context.Background()
	var u models.User
	err := db.pool.QueryRow(ctx,
		`SELECT id, username, email, password_hash, bio, skills, interests, github_url, avatar_url, location, created_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Bio, &u.Skills, &u.Interests,
		&u.GithubURL, &u.AvatarURL, &u.Location, &u.CreatedAt)

	if err == pgx.ErrNoRows {
		return u, false, nil
	}
	if err != nil {
		return u, false, err
	}
	return u, true, nil
}

// ListUsers returns all users.
func (db *PostgresDB) ListUsers() ([]models.User, error) {
	ctx := context.Background()
	rows, err := db.pool.Query(ctx,
		`SELECT id, username, email, password_hash, bio, skills, interests, github_url, avatar_url, location, created_at
		 FROM users ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Bio, &u.Skills, &u.Interests,
			&u.GithubURL, &u.AvatarURL, &u.Location, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// AuthenticateUser verifies a username and password.
func (db *PostgresDB) AuthenticateUser(username, password string) (models.User, bool, error) {
	ctx := context.Background()
	var u models.User
	err := db.pool.QueryRow(ctx,
		`SELECT id, username, email, password_hash, bio, skills, interests, github_url, avatar_url, location, created_at
		 FROM users WHERE username = $1 AND password_hash = $2`, username, password, // Normally compare with bcrypt
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Bio, &u.Skills, &u.Interests,
		&u.GithubURL, &u.AvatarURL, &u.Location, &u.CreatedAt)

	if err == pgx.ErrNoRows {
		return u, false, nil
	}
	if err != nil {
		return u, false, err
	}
	return u, true, nil
}

// ═══════════════════ PROJECT OPERATIONS ═══════════════════

// CreateProject inserts a new project.
func (db *PostgresDB) CreateProject(project models.Project) (models.Project, error) {
	ctx := context.Background()
	project.ID = generateID()
	project.CreatedAt = time.Now()
	if project.Status == "" {
		project.Status = "open"
	}
	if project.MaxMembers == 0 {
		project.MaxMembers = 5
	}

	_, err := db.pool.Exec(ctx,
		`INSERT INTO projects (id, title, description, tech_stack, owner_id, owner_name, status, max_members, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		project.ID, project.Title, project.Description, project.TechStack,
		project.OwnerID, project.OwnerName, project.Status, project.MaxMembers, project.CreatedAt,
	)
	if err != nil {
		return models.Project{}, fmt.Errorf("failed to create project: %w", err)
	}
	return project, nil
}

// ListProjects returns all projects.
func (db *PostgresDB) ListProjects() ([]models.Project, error) {
	ctx := context.Background()
	rows, err := db.pool.Query(ctx,
		`SELECT id, title, description, tech_stack, owner_id, owner_name, status, max_members, created_at
		 FROM projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.TechStack,
			&p.OwnerID, &p.OwnerName, &p.Status, &p.MaxMembers, &p.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

// GetProjectByID retrieves a project by ID.
func (db *PostgresDB) GetProjectByID(id string) (models.Project, bool, error) {
	ctx := context.Background()
	var p models.Project
	err := db.pool.QueryRow(ctx,
		`SELECT id, title, description, tech_stack, owner_id, owner_name, status, max_members, created_at
		 FROM projects WHERE id = $1`, id,
	).Scan(&p.ID, &p.Title, &p.Description, &p.TechStack,
		&p.OwnerID, &p.OwnerName, &p.Status, &p.MaxMembers, &p.CreatedAt)

	if err == pgx.ErrNoRows {
		return p, false, nil
	}
	if err != nil {
		return p, false, err
	}
	return p, true, nil
}

// ═══════════════════ MESSAGE OPERATIONS ═══════════════════

// SaveMessage stores a message.
func (db *PostgresDB) SaveMessage(msg models.Message) (models.Message, error) {
	ctx := context.Background()
	msg.ID = generateID()
	msg.Timestamp = time.Now()

	_, err := db.pool.Exec(ctx,
		`INSERT INTO messages (id, sender_id, username, content, room_id, timestamp)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		msg.ID, msg.SenderID, msg.Username, msg.Content, msg.RoomID, msg.Timestamp,
	)
	if err != nil {
		return models.Message{}, err
	}
	return msg, nil
}

// GetMessages retrieves messages for a room.
func (db *PostgresDB) GetMessages(roomID string, limit int) ([]models.Message, error) {
	ctx := context.Background()
	if limit <= 0 {
		limit = 50
	}

	rows, err := db.pool.Query(ctx,
		`SELECT id, sender_id, username, content, room_id, timestamp
		 FROM messages WHERE room_id = $1 ORDER BY timestamp DESC LIMIT $2`, roomID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []models.Message
	for rows.Next() {
		var m models.Message
		if err := rows.Scan(&m.ID, &m.SenderID, &m.Username, &m.Content, &m.RoomID, &m.Timestamp); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}

	// Reverse to get chronological order
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

// ═══════════════════ ROOM OPERATIONS ═══════════════════

// CreateRoom creates a chat room.
func (db *PostgresDB) CreateRoom(room models.ChatRoom) (models.ChatRoom, error) {
	ctx := context.Background()
	room.ID = generateID()

	_, err := db.pool.Exec(ctx,
		`INSERT INTO chat_rooms (id, name, description, participants) VALUES ($1, $2, $3, $4)
		 ON CONFLICT (id) DO NOTHING`,
		room.ID, room.Name, room.Description, room.Participants,
	)
	if err != nil {
		return models.ChatRoom{}, err
	}
	return room, nil
}

// ListRooms returns all chat rooms.
func (db *PostgresDB) ListRooms() ([]models.ChatRoom, error) {
	ctx := context.Background()
	rows, err := db.pool.Query(ctx,
		`SELECT id, name, description, participants FROM chat_rooms`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []models.ChatRoom
	for rows.Next() {
		var r models.ChatRoom
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.Participants); err != nil {
			return nil, err
		}
		rooms = append(rooms, r)
	}
	return rooms, nil
}

// ═══════════════════ VECTOR / RAG OPERATIONS ═══════════════════

// StoreEmbedding saves a vector embedding into pgvector.
func (db *PostgresDB) StoreEmbedding(embedding models.Embedding) error {
	ctx := context.Background()
	embedding.ID = generateID()

	// Convert []float32 to pgvector format string: "[0.1,0.2,...]"
	vectorStr := float32SliceToVectorString(embedding.Vector)

	_, err := db.pool.Exec(ctx,
		`INSERT INTO embeddings (id, source_id, source_type, text, vector)
		 VALUES ($1, $2, $3, $4, $5::vector)
		 ON CONFLICT (id) DO UPDATE SET vector = EXCLUDED.vector, text = EXCLUDED.text`,
		embedding.ID, embedding.SourceID, embedding.SourceType, embedding.Text, vectorStr,
	)
	return err
}

// DeleteEmbedding removes an embedding by source.
func (db *PostgresDB) DeleteEmbedding(sourceID, sourceType string) error {
	ctx := context.Background()
	_, err := db.pool.Exec(ctx,
		`DELETE FROM embeddings WHERE source_id = $1 AND source_type = $2`,
		sourceID, sourceType,
	)
	return err
}

// SearchSimilar finds the most similar embeddings using pgvector cosine distance.
// This is the core of the RAG pipeline — the "Retrieval" step.
func (db *PostgresDB) SearchSimilar(queryVector []float32, sourceType string, excludeID string, limit int) ([]models.SimilarityResult, error) {
	ctx := context.Background()
	if limit <= 0 {
		limit = 5
	}

	vectorStr := float32SliceToVectorString(queryVector)

	rows, err := db.pool.Query(ctx,
		`SELECT source_id, source_type, 1 - (vector <=> $1::vector) AS similarity
		 FROM embeddings
		 WHERE source_type = $2 AND source_id != $3
		 ORDER BY vector <=> $1::vector
		 LIMIT $4`,
		vectorStr, sourceType, excludeID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("pgvector similarity search failed: %w", err)
	}
	defer rows.Close()

	var results []models.SimilarityResult
	for rows.Next() {
		var r models.SimilarityResult
		if err := rows.Scan(&r.SourceID, &r.SourceType, &r.Score); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

// FindRelatedDevelopers finds developers whose embeddings are close to each other.
// This powers the "related developers" connection feature.
func (db *PostgresDB) FindRelatedDevelopers(userID string, limit int) ([]models.SimilarityResult, error) {
	ctx := context.Background()
	if limit <= 0 {
		limit = 5
	}

	rows, err := db.pool.Query(ctx,
		`SELECT e2.source_id, e2.source_type, 1 - (e1.vector <=> e2.vector) AS similarity
		 FROM embeddings e1
		 JOIN embeddings e2 ON e1.source_type = 'user' AND e2.source_type = 'user' AND e1.source_id != e2.source_id
		 WHERE e1.source_id = $1
		 ORDER BY e1.vector <=> e2.vector
		 LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.SimilarityResult
	for rows.Next() {
		var r models.SimilarityResult
		if err := rows.Scan(&r.SourceID, &r.SourceType, &r.Score); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

// ═══════════════════ SEED DATA ═══════════════════

// SeedDefaultRooms inserts the default chat rooms if they don't exist.
func (db *PostgresDB) SeedDefaultRooms() error {
	ctx := context.Background()
	rooms := []struct{ id, name, desc string }{
		{"room_general", "🌐 General", "Hang out and talk about anything tech-related."},
		{"room_golang", "🐹 Go/Golang", "All things Go — goroutines, channels, and beyond."},
		{"room_devops", "🚀 DevOps & Cloud", "Kubernetes, Docker, Terraform, CI/CD discussions."},
		{"room_ai", "🤖 AI & Machine Learning", "LLMs, RAG, embeddings, and ML engineering."},
		{"room_opensource", "💻 Open Source", "Share your contributions, find projects to contribute to."},
		{"room_career", "💼 Career & Jobs", "Resume reviews, interview prep, job opportunities."},
	}

	for _, r := range rooms {
		_, err := db.pool.Exec(ctx,
			`INSERT INTO chat_rooms (id, name, description, participants) VALUES ($1, $2, $3, '{}')
			 ON CONFLICT (id) DO NOTHING`,
			r.id, r.name, r.desc,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// HasUsers checks if there are any users in the database.
func (db *PostgresDB) HasUsers() (bool, error) {
	ctx := context.Background()
	var count int
	err := db.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	return count > 0, err
}

// ═══════════════════ HELPERS ═══════════════════

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// float32SliceToVectorString converts []float32 to pgvector format "[0.1,0.2,...]"
func float32SliceToVectorString(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = fmt.Sprintf("%f", f)
	}
	return "[" + strings.Join(parts, ",") + "]"
}
