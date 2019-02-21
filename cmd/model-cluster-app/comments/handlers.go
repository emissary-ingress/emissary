package comments

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type createArgs struct {
	AuthorID string
	PostID   string
	Content  string
}

func handlePost(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var args createArgs
	err := decoder.Decode(&args)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	resourceID := Add(args.AuthorID, args.PostID, args.Content)
	w.Write([]byte(resourceID))
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resourceID := vars["id"]
	resource, ok := Get(resourceID)
	if !ok {
		http.Error(w, http.StatusText(404), 404)
		return
	}
	result, err := json.Marshal(resource)
	if err != nil {
		http.Error(w, "Failed to marshal resource", 500)
		return
	}
	w.Write(result)
}

func handleList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := NarrowBy{
		AuthorID: q.Get("author"),
		PostID:   q.Get("post"),
	}
	resourceIDs := List(filter)
	result, err := json.Marshal(resourceIDs)
	if err != nil {
		http.Error(w, "Failed to marshal resource list", 500)
		return
	}
	w.Write(result)
}

// RegisterHandlers adds HTTP handlers for all the supported methods
func RegisterHandlers(r *mux.Router) {
	r.HandleFunc("/comments", handlePost).Methods("POST")
	r.HandleFunc("/comments/{id}", handleGet).Methods("GET")
	r.HandleFunc("/comments", handleList).Methods("GET")
}
