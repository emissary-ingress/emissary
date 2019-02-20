package comments

import (
	"fmt"
	"sync"
	"time"
)

// Comment represents one comment in the model cluster blog app.
type Comment struct {
	ID       string
	Created  time.Time
	AuthorID string
	PostID   string
	Content  string
}

var store = map[string]Comment{}
var counter int
var lock sync.Mutex

// Add creates a new comment and adds it to the store.
func Add(authorID, postID, content string) string {
	lock.Lock()
	defer lock.Unlock()
	comment := Comment{
		ID:       fmt.Sprintf("comment-%d", counter),
		Created:  time.Now(),
		AuthorID: authorID,
		PostID:   postID,
		Content:  content,
	}
	counter += 1
	store[comment.ID] = comment
	return comment.ID
}

// Get retrieves a comment by ID.
func Get(commentID string) (Comment, bool) {
	lock.Lock()
	defer lock.Unlock()
	comment, ok := store[commentID]
	return comment, ok
}

// NarrowBy specifies filters for a list operation.
type NarrowBy struct {
	AuthorID string
	PostID   string
}

// List returns a slice of comment IDs, optionally narrowing to the subset defined
// by the specified filters.
func List(filter NarrowBy) []string {
	lock.Lock()
	defer lock.Unlock()
	res := make([]string, 0)
	for _, comment := range store {
		match := true
		match = match && (filter.AuthorID == "" || filter.AuthorID == comment.AuthorID)
		match = match && (filter.PostID == "" || filter.PostID == comment.PostID)
		if match {
			res = append(res, comment.ID)
		}
	}
	return res
}
