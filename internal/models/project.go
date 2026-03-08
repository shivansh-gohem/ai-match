package models

import "time"

// Project represents a collaboration project posted by a developer.
type Project struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	TechStack   []string  `json:"tech_stack"`
	OwnerID     string    `json:"owner_id"`
	OwnerName   string    `json:"owner_name"`
	Status      string    `json:"status"` // "open", "in-progress", "completed"
	MaxMembers  int       `json:"max_members"`
	CreatedAt   time.Time `json:"created_at"`
}

// ProjectCreateRequest is the payload for creating a new project.
type ProjectCreateRequest struct {
	Title       string   `json:"title" binding:"required"`
	Description string   `json:"description" binding:"required"`
	TechStack   []string `json:"tech_stack"`
	OwnerID     string   `json:"owner_id" binding:"required"`
	MaxMembers  int      `json:"max_members"`
}
