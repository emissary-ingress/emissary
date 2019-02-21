package users

import (
	"fmt"
	"sync"
	"time"
)

// User represents one user the model cluster blog app.
type User struct {
	ID      string
	Created time.Time
	Email   string
	Name    string
	URL     string
}

var store = map[string]User{}
var counter int
var lock sync.Mutex

// Add creates a new user and adds it to the store
func Add(email, name, url string) string {
	lock.Lock()
	defer lock.Unlock()
	user := User{
		ID:      fmt.Sprintf("user-%d", counter),
		Created: time.Now(),
		Email:   email,
		Name:    name,
		URL:     url,
	}
	counter += 1
	store[user.ID] = user
	return user.ID
}

// Get retrieves a user by ID.
func Get(userID string) (User, bool) {
	lock.Lock()
	defer lock.Unlock()
	user, ok := store[userID]
	return user, ok
}

// List returns a slice of all user IDs.
func List() []string {
	lock.Lock()
	defer lock.Unlock()
	res := make([]string, 0)
	for userID := range store {
		res = append(res, userID)
	}
	return res
}
