package events

import (
	"fmt"
	"sync"
	"time"
)

// Event represents one event in the model cluster blog app.
type Event struct {
	ID        string
	Created   time.Time
	Action    string
	UserID    string
	PostID    string
	CommentID string
}

var store = map[string]Event{}
var counter int
var lock sync.Mutex

// Add creates a new event and adds it to the store.
func Add(action, userID, postID, commentID string) string {
	lock.Lock()
	defer lock.Unlock()
	event := Event{
		ID:        fmt.Sprintf("event-%d", counter),
		Created:   time.Now(),
		UserID:    userID,
		PostID:    postID,
		CommentID: commentID,
	}
	counter += 1
	store[event.ID] = event
	return event.ID
}

// Get retrieves a event by ID.
func Get(eventID string) (Event, bool) {
	lock.Lock()
	defer lock.Unlock()
	event, ok := store[eventID]
	return event, ok
}

// NarrowBy specifies filters for a list operation.
type NarrowBy struct {
	UserID    string
	PostID    string
	CommentID string
}

// List returns a slice of event IDs, optionally narrowing to the subset defined
// by the specified filters.
func List(filter NarrowBy) []string {
	lock.Lock()
	defer lock.Unlock()
	res := make([]string, 0)
	for _, event := range store {
		match := true
		match = match && (filter.UserID == "" || filter.UserID == event.UserID)
		match = match && (filter.PostID == "" || filter.PostID == event.PostID)
		match = match && (filter.CommentID == "" || filter.CommentID == event.CommentID)
		if match {
			res = append(res, event.ID)
		}
	}
	return res
}
