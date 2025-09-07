package domain

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserRole string

const (
	RoleUser      UserRole = "user"
	RoleAdmin     UserRole = "admin"
	RoleModerator UserRole = "moderator"
)

type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Role         UserRole  `json:"role" db:"role"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role,omitempty"`
}

type UpdateUserRequest struct {
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
	Role     string `json:"role,omitempty"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	User         *User  `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// NewUser creates a new user with validation
func NewUser(username, email, password string, role UserRole) (*User, error) {
	user := &User{
		ID:        uuid.New(),
		Username:  strings.TrimSpace(username),
		Email:     strings.TrimSpace(strings.ToLower(email)),
		Role:      role,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := user.Validate(); err != nil {
		return nil, err
	}

	if err := user.SetPassword(password); err != nil {
		return nil, err
	}

	return user, nil
}

// Validate validates user fields
func (u *User) Validate() error {
	if len(u.Username) < 3 || len(u.Username) > 50 {
		return fmt.Errorf("username must be between 3 and 50 characters")
	}

	if !isValidEmail(u.Email) {
		return fmt.Errorf("invalid email format")
	}

	if !isValidRole(string(u.Role)) {
		return fmt.Errorf("invalid role: %s", u.Role)
	}

	return nil
}

// SetPassword hashes and sets the user password
func (u *User) SetPassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	u.PasswordHash = string(hashedPassword)
	u.UpdatedAt = time.Now()
	return nil
}

// CheckPassword verifies the provided password against the stored hash
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// IsAdmin checks if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsModerator checks if the user has moderator role
func (u *User) IsModerator() bool {
	return u.Role == RoleModerator
}

// CanManageUsers checks if the user can manage other users
func (u *User) CanManageUsers() bool {
	return u.IsAdmin() || u.IsModerator()
}

// MarshalJSON customizes JSON marshaling to exclude sensitive data
func (u *User) MarshalJSON() ([]byte, error) {
	type Alias User
	return json.Marshal(&struct {
		*Alias
		PasswordHash string `json:"-"`
	}{
		Alias: (*Alias)(u),
	})
}

func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
	return emailRegex.MatchString(email)
}

func isValidRole(role string) bool {
	validRoles := []string{string(RoleUser), string(RoleAdmin), string(RoleModerator)}
	for _, validRole := range validRoles {
		if role == validRole {
			return true
		}
	}
	return false
}
