package models

import "time"

// Message represents a chat message in a room.
type Message struct {
	ID        string    `json:"id"`
	SenderID  string    `json:"sender_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	RoomID    string    `json:"room_id"`
	Timestamp time.Time `json:"timestamp"`
}

// ChatRoom represents a chat room for discussions.
type ChatRoom struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Participants []string `json:"participants"`
}

// WSMessage is the WebSocket message envelope.
type WSMessage struct {
	Type     string `json:"type"`     // "chat", "join", "leave", "system"
	Content  string `json:"content"`
	Username string `json:"username"`
	RoomID   string `json:"room_id"`
}

// MatchResult represents an AI-generated match suggestion.
type MatchResult struct {
	User       User    `json:"user"`
	Score      float64 `json:"score"`
	Reason     string  `json:"reason"`
}

// AIResponse wraps a Gemini API response.
type AIResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}
