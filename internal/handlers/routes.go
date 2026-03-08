package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/shiva/ai-match/internal/models"
	"github.com/shiva/ai-match/internal/repository"
	"github.com/shiva/ai-match/internal/service"
)

// Handler holds all dependencies for the API handlers.
type Handler struct {
	FakeDB      *repository.FakeDB
	PgDB        *repository.PostgresDB
	Gemini      *service.GeminiService
	Matcher     *service.MatchService
	Hub         *service.Hub
	usePostgres bool
}

// NewHandler creates a new Handler with all dependencies.
func NewHandler(fakeDB *repository.FakeDB, pgDB *repository.PostgresDB, gemini *service.GeminiService, matcher *service.MatchService, hub *service.Hub) *Handler {
	return &Handler{
		FakeDB:      fakeDB,
		PgDB:        pgDB,
		Gemini:      gemini,
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
		// Health check
		api.GET("/health", h.HealthCheck)

		// User routes
		api.POST("/users", h.CreateUser)
		api.GET("/users", h.ListUsers)
		api.GET("/users/:id", h.GetUser)

		// Auth routes
		api.POST("/auth/login", h.Login)

		// Project routes
		api.POST("/projects", h.CreateProject)
		api.GET("/projects", h.ListProjects)
		api.GET("/projects/:id", h.GetProject)

		// AI Matching & related users routes
		api.GET("/match/user/:id", h.GetUserMatches)
		api.GET("/match/project/:id", h.GetProjectMatches)
		api.GET("/connections/user/:id", h.GetUserConnections)

		// AI Chat
		api.POST("/ai/chat", h.AIChat)

		// Chat rooms
		api.GET("/rooms", h.ListRooms)
		api.GET("/rooms/:id/messages", h.GetRoomMessages)

		// WebSocket
		api.GET("/ws", h.HandleWebSocket)

		// Stats
		api.GET("/stats", h.GetStats)
	}
}

// ---------- HEALTH ----------

func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "API is running 🚀",
		"ai_status": h.Gemini.IsAvailable(),
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

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"user":    user,
	})
}

// ---------- USERS ----------

func (h *Handler) CreateUser(c *gin.Context) {
	var req models.UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := models.User{
		Username:  req.Username,
		Email:     req.Email,
		Bio:       req.Bio,
		Skills:    req.Skills,
		Interests: req.Interests,
		GithubURL: req.GithubURL,
		Location:  req.Location,
	}

	var created models.User
	var err error

	user.PasswordHash = req.Password // Normally hash this with bcrypt

	if h.usePostgres {
		created, err = h.PgDB.CreateUser(user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		created = h.FakeDB.CreateUser(user)
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

// ---------- AI CHAT ----------

func (h *Handler) AIChat(c *gin.Context) {
	var req struct {
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message is required"})
		return
	}

	response, err := h.Gemini.ChatWithAI(req.Message)
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

	// Attach online counts
	type RoomWithCount struct {
		models.ChatRoom
		OnlineCount int `json:"online_count"`
	}
	var result []RoomWithCount
	for _, r := range rooms {
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
		"ai_enabled":       h.Gemini.IsAvailable(),
		"db_type":          func() string { if h.usePostgres { return "postgres" }; return "fake" }(),
	})
}
