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
// Returns an error if the email is already registered.
func (db *FakeDB) CreateUser(user models.User) (models.User, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Check for duplicate email
	for _, existing := range db.Users {
		if existing.Email == user.Email {
			return models.User{}, fmt.Errorf("Email is already registered")
		}
	}

	user.ID = generateID()
	user.CreatedAt = time.Now()

	// GitHub avatar: use GitHub profile picture if GithubID is provided
	if user.GithubID != "" {
		user.AvatarURL = fmt.Sprintf("https://github.com/%s.png", user.GithubID)
	}
	if user.AvatarURL == "" {
		user.AvatarURL = fmt.Sprintf("https://api.dicebear.com/7.x/avataaars/svg?seed=%s", user.Username)
	}

	db.Users[user.ID] = user
	return user, nil
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

// ---------- DM OPERATIONS ----------

// CreateOrGetDMRoom creates a DM room between two users, or returns the existing one.
func (db *FakeDB) CreateOrGetDMRoom(userID1, userID2, username1, username2 string) models.ChatRoom {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Create a canonical DM room ID (sorted to ensure consistency)
	var dmID string
	if userID1 < userID2 {
		dmID = "dm_" + userID1 + "_" + userID2
	} else {
		dmID = "dm_" + userID2 + "_" + userID1
	}

	// Check if already exists
	if room, ok := db.Rooms[dmID]; ok {
		return room
	}

	// Create new DM room
	room := models.ChatRoom{
		ID:           dmID,
		Name:         fmt.Sprintf("💬 %s & %s", username1, username2),
		Description:  fmt.Sprintf("Direct messages between %s and %s", username1, username2),
		Participants: []string{userID1, userID2},
	}
	db.Rooms[dmID] = room
	return room
}

// ListDMRooms lists all DM rooms for a user.
func (db *FakeDB) ListDMRooms(userID string) []models.ChatRoom {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var dms []models.ChatRoom
	for _, room := range db.Rooms {
		if len(room.ID) > 3 && room.ID[:3] == "dm_" {
			for _, p := range room.Participants {
				if p == userID {
					dms = append(dms, room)
					break
				}
			}
		}
	}
	return dms
}

// ---------- PROJECT MEMBER OPERATIONS ----------

// JoinProject adds a user to a project's members list.
func (db *FakeDB) JoinProject(projectID, userID string) (models.Project, bool, string) {
	db.mu.Lock()
	defer db.mu.Unlock()

	project, ok := db.Projects[projectID]
	if !ok {
		return models.Project{}, false, "Project not found"
	}

	// Check if already a member
	for _, m := range project.Members {
		if m == userID {
			return project, false, "Already a member of this project"
		}
	}

	// Check member limit
	if len(project.Members) >= project.MaxMembers {
		return project, false, "Project is full"
	}

	// Add user
	project.Members = append(project.Members, userID)

	// Resolve username
	if user, uok := db.Users[userID]; uok {
		project.MemberNames = append(project.MemberNames, user.Username)
	}

	db.Projects[projectID] = project
	return project, true, "Joined successfully"
}

// GetProjectMembers returns the member user objects for a project.
func (db *FakeDB) GetProjectMembers(projectID string) ([]models.User, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	project, ok := db.Projects[projectID]
	if !ok {
		return nil, false
	}

	var members []models.User
	for _, uid := range project.Members {
		if user, uok := db.Users[uid]; uok {
			members = append(members, user)
		}
	}
	return members, true
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

