package models

import "time"

// User represents a developer on the platform.
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Bio       string    `json:"bio"`
	Skills    []string  `json:"skills"`
	Interests []string  `json:"interests"`
	GithubURL string    `json:"github_url"`
	GithubID  string    `json:"github_id"`
	AvatarURL string    `json:"avatar_url"`
	Location     string    `json:"location"`
	PasswordHash string    `json:"-"` // Don't expose in JSON
	CreatedAt    time.Time `json:"created_at"`
}

// UserCreateRequest is the payload for creating a new user.
type UserCreateRequest struct {
	Username  string   `json:"username" binding:"required"`
	Email     string   `json:"email" binding:"required"`
	Bio       string   `json:"bio"`
	Skills    []string `json:"skills"`
	Interests []string `json:"interests"`
	GithubURL string   `json:"github_url"`
	GithubID  string   `json:"github_id" binding:"required"`
	Location  string   `json:"location"`
	Password  string   `json:"password" binding:"required"`
}

// UserLoginRequest is the payload for authenticating a user.
type UserLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse is the response payload for a successful login,
// containing the user data and a JWT token.
type AuthResponse struct {
	User  User   `json:"user"`
	Token string `json:"token"`
}
