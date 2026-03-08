package repository

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/shiva/ai-match/internal/models"
)

// FakeDB is a thread-safe in-memory database for the MVP.
type FakeDB struct {
	mu       sync.RWMutex
	Users    map[string]models.User
	Projects map[string]models.Project
	Messages map[string][]models.Message // roomID -> messages
	Rooms    map[string]models.ChatRoom
}

// NewFakeDB creates a new in-memory database and seeds it with fake data.
func NewFakeDB() *FakeDB {
	db := &FakeDB{
		Users:    make(map[string]models.User),
		Projects: make(map[string]models.Project),
		Messages: make(map[string][]models.Message),
		Rooms:    make(map[string]models.ChatRoom),
	}
	db.Seed()
	return db
}

// ---------- USER OPERATIONS ----------

// CreateUser adds a new user to the database.
func (db *FakeDB) CreateUser(user models.User) models.User {
	db.mu.Lock()
	defer db.mu.Unlock()
	user.ID = generateID()
	user.CreatedAt = time.Now()
	if user.AvatarURL == "" {
		user.AvatarURL = fmt.Sprintf("https://api.dicebear.com/7.x/avataaars/svg?seed=%s", user.Username)
	}
	db.Users[user.ID] = user
	return user
}

// GetUserByID retrieves a user by their ID.
func (db *FakeDB) GetUserByID(id string) (models.User, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	user, ok := db.Users[id]
	return user, ok
}

// ListUsers returns all users.
func (db *FakeDB) ListUsers() []models.User {
	db.mu.RLock()
	defer db.mu.RUnlock()
	users := make([]models.User, 0, len(db.Users))
	for _, u := range db.Users {
		users = append(users, u)
	}
	return users
}

// AuthenticateUser verifies a username and password.
func (db *FakeDB) AuthenticateUser(username, password string) (models.User, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	for _, u := range db.Users {
		if u.Username == username && u.PasswordHash == password { // Normally compare with bcrypt
			return u, true
		}
	}
	return models.User{}, false
}

// ---------- PROJECT OPERATIONS ----------

// CreateProject adds a new project to the database.
func (db *FakeDB) CreateProject(project models.Project) models.Project {
	db.mu.Lock()
	defer db.mu.Unlock()
	project.ID = generateID()
	project.CreatedAt = time.Now()
	if project.Status == "" {
		project.Status = "open"
	}
	if project.MaxMembers == 0 {
		project.MaxMembers = 5
	}
	// Attach owner name
	if owner, ok := db.Users[project.OwnerID]; ok {
		project.OwnerName = owner.Username
	}
	db.Projects[project.ID] = project
	return project
}

// ListProjects returns all projects.
func (db *FakeDB) ListProjects() []models.Project {
	db.mu.RLock()
	defer db.mu.RUnlock()
	projects := make([]models.Project, 0, len(db.Projects))
	for _, p := range db.Projects {
		projects = append(projects, p)
	}
	return projects
}

// GetProjectByID retrieves a project by ID.
func (db *FakeDB) GetProjectByID(id string) (models.Project, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	project, ok := db.Projects[id]
	return project, ok
}

// ---------- MESSAGE OPERATIONS ----------

// SaveMessage stores a message in a chat room.
func (db *FakeDB) SaveMessage(msg models.Message) models.Message {
	db.mu.Lock()
	defer db.mu.Unlock()
	msg.ID = generateID()
	msg.Timestamp = time.Now()
	db.Messages[msg.RoomID] = append(db.Messages[msg.RoomID], msg)
	return msg
}

// GetMessages retrieves messages for a room.
func (db *FakeDB) GetMessages(roomID string) []models.Message {
	db.mu.RLock()
	defer db.mu.RUnlock()
	msgs, ok := db.Messages[roomID]
	if !ok {
		return []models.Message{}
	}
	return msgs
}

// ---------- ROOM OPERATIONS ----------

// CreateRoom creates a new chat room.
func (db *FakeDB) CreateRoom(room models.ChatRoom) models.ChatRoom {
	db.mu.Lock()
	defer db.mu.Unlock()
	room.ID = generateID()
	db.Rooms[room.ID] = room
	return room
}

// ListRooms returns all chat rooms.
func (db *FakeDB) ListRooms() []models.ChatRoom {
	db.mu.RLock()
	defer db.mu.RUnlock()
	rooms := make([]models.ChatRoom, 0, len(db.Rooms))
	for _, r := range db.Rooms {
		rooms = append(rooms, r)
	}
	return rooms
}

// GetRoomByID retrieves a chat room by ID.
func (db *FakeDB) GetRoomByID(id string) (models.ChatRoom, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	room, ok := db.Rooms[id]
	return room, ok
}

// ---------- MATCHING ----------

// FindMatchingUsers returns users whose skills overlap with the given skills/interests.
func (db *FakeDB) FindMatchingUsers(skills []string, excludeID string) []models.User {
	db.mu.RLock()
	defer db.mu.RUnlock()

	skillSet := make(map[string]bool)
	for _, s := range skills {
		skillSet[s] = true
	}

	var matches []models.User
	for _, user := range db.Users {
		if user.ID == excludeID {
			continue
		}
		score := 0
		for _, s := range user.Skills {
			if skillSet[s] {
				score++
			}
		}
		for _, i := range user.Interests {
			if skillSet[i] {
				score++
			}
		}
		if score > 0 {
			matches = append(matches, user)
		}
	}
	return matches
}

// ---------- HELPERS ----------

func generateID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 12)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}
