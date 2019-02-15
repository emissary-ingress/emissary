package posts

import (
	"fmt"
	"sync"
	"time"
)

// Post represents one post in the model cluster blog app.
type Post struct {
	ID       string
	Created  time.Time
	Title    string
	AuthorID string
	Content  string
}

var store = map[string]Post{}
var counter int
var lock sync.Mutex

// Add creates a new post and adds it to the store.
func Add(title, authorID, content string) string {
	lock.Lock()
	defer lock.Unlock()
	post := Post{
		ID:       fmt.Sprintf("post-%d", counter),
		Created:  time.Now(),
		Title:    title,
		AuthorID: authorID,
		Content:  content,
	}
	counter += 1
	store[post.ID] = post
	return post.ID
}

// Get retrieves a post by ID.
func Get(postID string) (Post, bool) {
	lock.Lock()
	defer lock.Unlock()
	post, ok := store[postID]
	return post, ok
}

// NarrowBy specifies filters for a list operation.
type NarrowBy struct {
	AuthorID string
}

// List returns a slice of post IDs, optionally narrowing to the subset defined
// by the specified filters.
func List(filter NarrowBy) []string {
	lock.Lock()
	defer lock.Unlock()
	res := make([]string, 0)
	for _, post := range store {
		if filter.AuthorID == "" || filter.AuthorID == post.AuthorID {
			res = append(res, post.ID)
		}
	}
	return res
}
