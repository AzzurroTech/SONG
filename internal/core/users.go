package core

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the system
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	Role         string    `json:"role"` // admin, developer, viewer
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Active       bool      `json:"active"`
}

// UsersData represents the structure of users.json
type UsersData struct {
	Version     string `json:"version"`
	LastUpdated string `json:"last_updated"`
	Users       []User `json:"users"`
}

// UserManager handles user operations
type UserManager struct {
	store *Store
}

// NewUserManager creates a new user manager
func NewUserManager(store *Store) *UserManager {
	return &UserManager{store: store}
}

// ErrUserNotFound is returned when a user cannot be located
var ErrUserNotFound = errors.New("user not found")

// ErrUserExists is returned when attempting to create a duplicate user
var ErrUserExists = errors.New("user already exists")

// GetAll returns all users
func (um *UserManager) GetAll() ([]User, error) {
	var data UsersData
	if err := um.store.Read("users.json", &data); err != nil {
		return nil, err
	}
	return data.Users, nil
}

// GetByID retrieves a user by their ID
func (um *UserManager) GetByID(id string) (*User, error) {
	users, err := um.GetAll()
	if err != nil {
		return nil, err
	}

	for _, u := range users {
		if u.ID == id {
			return &u, nil
		}
	}
	return nil, ErrUserNotFound
}

// GetByUsername retrieves a user by their username
func (um *UserManager) GetByUsername(username string) (*User, error) {
	users, err := um.GetAll()
	if err != nil {
		return nil, err
	}

	for _, u := range users {
		if u.Username == username {
			return &u, nil
		}
	}
	return nil, ErrUserNotFound
}

// GetByEmail retrieves a user by their email
func (um *UserManager) GetByEmail(email string) (*User, error) {
	users, err := um.GetAll()
	if err != nil {
		return nil, err
	}

	for _, u := range users {
		if u.Email == email {
			return &u, nil
		}
	}
	return nil, ErrUserNotFound
}

// Create creates a new user with a hashed password
func (um *UserManager) Create(id, username, email, password, role string) (*User, error) {
	// Check if username or email already exists
	existingUser, _ := um.GetByUsername(username)
	if existingUser != nil {
		return nil, ErrUserExists
	}

	existingEmail, _ := um.GetByEmail(email)
	if existingEmail != nil {
		return nil, errors.New("email already in use")
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now().UTC()
	user := User{
		ID:           id,
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
		Role:         role,
		CreatedAt:    now,
		UpdatedAt:    now,
		Active:       true,
	}

	// Load existing data
	var data UsersData
	if err := um.store.Read("users.json", &data); err != nil {
		return nil, err
	}

	// Initialize if empty
	if data.Users == nil {
		data.Users = []User{}
		data.Version = "1.0"
	}

	// Add new user
	data.Users = append(data.Users, user)
	data.LastUpdated = now.Format(time.RFC3339)

	// Save
	if err := um.store.Write("users.json", data); err != nil {
		return nil, err
	}

	return &user, nil
}

// Update modifies an existing user
func (um *UserManager) Update(id string, updates map[string]interface{}) (*User, error) {
	var data UsersData
	if err := um.store.Read("users.json", &data); err != nil {
		return nil, err
	}

	found := false
	for i, u := range data.Users {
		if u.ID == id {
			found = true

			// Apply updates
			if email, ok := updates["email"].(string); ok {
				u.Email = email
			}
			if role, ok := updates["role"].(string); ok {
				u.Role = role
			}
			if active, ok := updates["active"].(bool); ok {
				u.Active = active
			}
			if password, ok := updates["password"].(string); ok {
				hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
				if err != nil {
					return nil, fmt.Errorf("failed to hash password: %w", err)
				}
				u.PasswordHash = string(hashedPassword)
			}

			u.UpdatedAt = time.Now().UTC()
			data.Users[i] = u

			break
		}
	}

	if !found {
		return nil, ErrUserNotFound
	}

	data.LastUpdated = time.Now().UTC().Format(time.RFC3339)
	if err := um.store.Write("users.json", data); err != nil {
		return nil, err
	}

	// Return updated user
	return um.GetByID(id)
}

// Delete removes a user by ID
func (um *UserManager) Delete(id string) error {
	var data UsersData
	if err := um.store.Read("users.json", &data); err != nil {
		return err
	}

	found := false
	newUsers := make([]User, 0, len(data.Users)-1)
	for _, u := range data.Users {
		if u.ID == id {
			found = true
			continue
		}
		newUsers = append(newUsers, u)
	}

	if !found {
		return ErrUserNotFound
	}

	data.Users = newUsers
	data.LastUpdated = time.Now().UTC().Format(time.RFC3339)

	return um.store.Write("users.json", data)
}

// ValidatePassword checks if the provided password matches the stored hash
func (um *UserManager) ValidatePassword(user *User, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	return err == nil
}

// Count returns the total number of users
func (um *UserManager) Count() (int, error) {
	users, err := um.GetAll()
	if err != nil {
		return 0, err
	}
	return len(users), nil
}
