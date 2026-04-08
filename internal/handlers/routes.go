package handlers

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/shiva/ai-match/internal/middleware"
	"github.com/shiva/ai-match/internal/models"
	"github.com/shiva/ai-match/internal/repository"
	"github.com/shiva/ai-match/internal/service"
)

// Handler holds all dependencies for the API handlers.
type Handler struct {
	FakeDB      *repository.FakeDB
	PgDB        *repository.PostgresDB
	Gemini      *service.GeminiService
	Groq        *service.GroqService
	Matcher     *service.MatchService
	Hub         *service.Hub
	usePostgres bool
}

// NewHandler creates a new Handler with all dependencies.
func NewHandler(fakeDB *repository.FakeDB, pgDB *repository.PostgresDB, gemini *service.GeminiService, groq *service.GroqService, matcher *service.MatchService, hub *service.Hub) *Handler {
	return &Handler{
		FakeDB:      fakeDB,
		PgDB:        pgDB,
		Gemini:      gemini,
		Groq:        groq,
		Matcher:     matcher,
		Hub:         hub,
		usePostgres: pgDB != nil,
	}
}

// SetupRoutes registers all API endpoints on the Gin router.
func (h *Handler) SetupRoutes(r *gin.Engine) {
	// Serve static frontend files
	r.Static("/static", "./web")
	r.StaticFile("/", "./web/index.html")

	api := r.Group("/api/v1")
	{
		// ── Public routes (no auth required) ──
		api.GET("/health", h.HealthCheck)
		api.GET("/stats", h.GetStats)
		api.POST("/auth/login", h.Login)
		api.POST("/users", h.CreateUser)

		// ── Protected routes (JWT required) ──
		protected := api.Group("/")
		protected.Use(middleware.AuthRequired())
		{
			// Users
			protected.GET("/users", h.ListUsers)
			protected.GET("/users/:id", h.GetUser)

			// Projects
			protected.POST("/projects", h.CreateProject)
			protected.GET("/projects", h.ListProjects)
			protected.GET("/projects/:id", h.GetProject)
			protected.POST("/projects/:id/join", h.JoinProject)
			protected.GET("/projects/:id/members", h.GetProjectMembers)

			// AI Matching
			protected.GET("/match/user/:id", h.GetUserMatches)
			protected.GET("/match/project/:id", h.GetProjectMatches)
			protected.GET("/connections/user/:id", h.GetUserConnections)

			// AI Chat (Groq)
			protected.POST("/ai/chat", h.AIChat)

			// Chat rooms
			protected.GET("/rooms", h.ListRooms)
			protected.GET("/rooms/:id/messages", h.GetRoomMessages)

			// DM routes
			protected.POST("/dm/start", h.StartDM)
			protected.GET("/dm/list/:userId", h.ListDMs)
		}

		// WebSocket — uses OptionalAuth (token via query param)
		api.GET("/ws", middleware.OptionalAuth(), h.HandleWebSocket)
	}
}

// ---------- HEALTH ----------

func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "API is running 🚀",
		"ai_status": h.Groq.IsAvailable(),
		"online":    h.Hub.GetOnlineCount(),
		"db_type":   func() string { if h.usePostgres { return "postgres" }; return "fake" }(),
	})
}

// ---------- AUTH ----------

func (h *Handler) Login(c *gin.Context) {
	var req models.UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	var ok bool
	var err error

	if h.usePostgres {
		user, ok, err = h.PgDB.AuthenticateUser(req.Username, req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		user, ok = h.FakeDB.AuthenticateUser(req.Username, req.Password)
	}

	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// Generate JWT token
	token, err := middleware.GenerateToken(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, models.AuthResponse{
		User:  user,
		Token: token,
	})
}

// ---------- USERS ----------

// verifyGitHubUser checks if a GitHub username actually exists by calling the GitHub API.
func verifyGitHubUser(username string) (bool, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("https://api.github.com/users/%s", username))
	if err != nil {
		return false, fmt.Errorf("could not verify GitHub username: %w", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK, nil
}

func (h *Handler) CreateUser(c *gin.Context) {
	var req models.UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// ── Email format validation ──
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email format"})
		return
	}

	// ── GitHub ID format validation ──
	githubRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?$`)
	if len(req.GithubID) == 0 || len(req.GithubID) > 39 || !githubRegex.MatchString(req.GithubID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid GitHub username format. Must be 1-39 characters, alphanumeric or hyphens only."})
		return
	}

	// ── Verify GitHub user actually exists via GitHub API ──
	exists, err := verifyGitHubUser(req.GithubID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Could not verify GitHub username. Please try again."})
		return
	}
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("GitHub user '%s' does not exist. Please enter your real GitHub username.", req.GithubID)})
		return
	}

	// Auto-generate GithubURL from GithubID
	githubURL := req.GithubURL
	if githubURL == "" {
		githubURL = "https://github.com/" + req.GithubID
	}

	user := models.User{
		Username:  req.Username,
		Email:     req.Email,
		Bio:       req.Bio,
		Skills:    req.Skills,
		Interests: req.Interests,
		GithubURL: githubURL,
		GithubID:  req.GithubID,
		Location:  req.Location,
	}

	var created models.User

	user.PasswordHash = req.Password // Normally hash this with bcrypt

	if h.usePostgres {
		created, err = h.PgDB.CreateUser(user)
		if err != nil {
			// Catch Postgres UNIQUE constraint violation on email
			if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Email is already registered"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		created, err = h.FakeDB.CreateUser(user)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusCreated, created)
}

func (h *Handler) ListUsers(c *gin.Context) {
	if h.usePostgres {
		users, err := h.PgDB.ListUsers()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, users)
	} else {
		users := h.FakeDB.ListUsers()
		c.JSON(http.StatusOK, users)
	}
}

func (h *Handler) GetUser(c *gin.Context) {
	id := c.Param("id")
	
	if h.usePostgres {
		user, ok, err := h.PgDB.GetUserByID(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusOK, user)
	} else {
		user, ok := h.FakeDB.GetUserByID(id)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusOK, user)
	}
}

// ---------- PROJECTS ----------

func (h *Handler) CreateProject(c *gin.Context) {
	var req models.ProjectCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	project := models.Project{
		Title:       req.Title,
		Description: req.Description,
		TechStack:   req.TechStack,
		OwnerID:     req.OwnerID,
		MaxMembers:  req.MaxMembers,
	}

	var created models.Project
	var err error

	if h.usePostgres {
		created, err = h.PgDB.CreateProject(project)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		created = h.FakeDB.CreateProject(project)
	}

	c.JSON(http.StatusCreated, created)
}

func (h *Handler) ListProjects(c *gin.Context) {
	if h.usePostgres {
		projects, err := h.PgDB.ListProjects()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, projects)
	} else {
		projects := h.FakeDB.ListProjects()
		c.JSON(http.StatusOK, projects)
	}
}

func (h *Handler) GetProject(c *gin.Context) {
	id := c.Param("id")
	
	if h.usePostgres {
		project, ok, err := h.PgDB.GetProjectByID(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		c.JSON(http.StatusOK, project)
	} else {
		project, ok := h.FakeDB.GetProjectByID(id)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		c.JSON(http.StatusOK, project)
	}
}

func (h *Handler) JoinProject(c *gin.Context) {
	projectID := c.Param("id")
	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	project, ok, msg := h.FakeDB.JoinProject(projectID, req.UserID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": msg,
		"project": project,
	})
}

func (h *Handler) GetProjectMembers(c *gin.Context) {
	projectID := c.Param("id")

	members, ok := h.FakeDB.GetProjectMembers(projectID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	c.JSON(http.StatusOK, members)
}

// ---------- AI MATCHING ----------

func (h *Handler) GetUserMatches(c *gin.Context) {
	id := c.Param("id")
	matches, err := h.Matcher.FindMatches(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if matches == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user_id": id,
		"matches": matches,
		"count":   len(matches),
	})
}

func (h *Handler) GetProjectMatches(c *gin.Context) {
	id := c.Param("id")
	matches, err := h.Matcher.FindMatchesForProject(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if matches == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"project_id": id,
		"matches":    matches,
		"count":      len(matches),
	})
}

func (h *Handler) GetUserConnections(c *gin.Context) {
	id := c.Param("id")
	connections, err := h.Matcher.FindConnections(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if connections == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user_id":     id,
		"connections": connections,
		"count":       len(connections),
	})
}

// ---------- AI CHAT (Groq-Powered) ----------

func (h *Handler) AIChat(c *gin.Context) {
	var req struct {
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message is required"})
		return
	}

	// Gather current platform data for AI context
	users := h.FakeDB.ListUsers()
	projects := h.FakeDB.ListProjects()

	response, err := h.Groq.ChatWithContext(req.Message, users, projects)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.AIResponse{
		Message: response,
		Success: true,
	})
}

// ---------- CHAT ROOMS ----------

func (h *Handler) ListRooms(c *gin.Context) {
	var rooms []models.ChatRoom
	
	if h.usePostgres {
		var err error
		rooms, err = h.PgDB.ListRooms()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		rooms = h.FakeDB.ListRooms()
	}

	// Filter out DM rooms from the public room list
	var publicRooms []models.ChatRoom
	for _, r := range rooms {
		if len(r.ID) < 3 || r.ID[:3] != "dm_" {
			publicRooms = append(publicRooms, r)
		}
	}

	// Attach online counts
	type RoomWithCount struct {
		models.ChatRoom
		OnlineCount int `json:"online_count"`
	}
	var result []RoomWithCount
	for _, r := range publicRooms {
		result = append(result, RoomWithCount{
			ChatRoom:    r,
			OnlineCount: h.Hub.GetRoomCount(r.ID),
		})
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetRoomMessages(c *gin.Context) {
	roomID := c.Param("id")
	
	if h.usePostgres {
		messages, err := h.PgDB.GetMessages(roomID, 50)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, messages)
	} else {
		messages := h.FakeDB.GetMessages(roomID)
		c.JSON(http.StatusOK, messages)
	}
}

// ---------- DM ----------

func (h *Handler) StartDM(c *gin.Context) {
	var req struct {
		User1ID   string `json:"user1_id" binding:"required"`
		User2ID   string `json:"user2_id" binding:"required"`
		Username1 string `json:"username1" binding:"required"`
		Username2 string `json:"username2" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	room := h.FakeDB.CreateOrGetDMRoom(req.User1ID, req.User2ID, req.Username1, req.Username2)
	c.JSON(http.StatusOK, room)
}

func (h *Handler) ListDMs(c *gin.Context) {
	userID := c.Param("userId")
	dms := h.FakeDB.ListDMRooms(userID)
	if dms == nil {
		dms = []models.ChatRoom{}
	}
	c.JSON(http.StatusOK, dms)
}

// ---------- WEBSOCKET ----------

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for MVP
	},
}

func (h *Handler) HandleWebSocket(c *gin.Context) {
	username := c.Query("username")
	roomID := c.Query("room")

	if username == "" {
		username = "anonymous"
	}
	if roomID == "" {
		roomID = "room_general"
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upgrade connection"})
		return
	}

	client := &service.Client{
		Hub:      h.Hub,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Username: username,
		RoomID:   roomID,
	}

	h.Hub.Register <- client

	go client.WritePump()
	go client.ReadPump()
}

// ---------- STATS ----------

func (h *Handler) GetStats(c *gin.Context) {
	var developersCount, projectsCount, roomsCount int

	if h.usePostgres {
		users, _ := h.PgDB.ListUsers()
		developersCount = len(users)
		
		projects, _ := h.PgDB.ListProjects()
		projectsCount = len(projects)
		
		rooms, _ := h.PgDB.ListRooms()
		roomsCount = len(rooms)
	} else {
		users := h.FakeDB.ListUsers()
		developersCount = len(users)
		
		projects := h.FakeDB.ListProjects()
		projectsCount = len(projects)
		
		rooms := h.FakeDB.ListRooms()
		roomsCount = len(rooms)
	}

	c.JSON(http.StatusOK, gin.H{
		"total_developers": developersCount,
		"total_projects":   projectsCount,
		"total_rooms":      roomsCount,
		"online_now":       h.Hub.GetOnlineCount(),
		"ai_enabled":       h.Groq.IsAvailable(),
		"db_type":          func() string { if h.usePostgres { return "postgres" }; return "fake" }(),
	})
}
