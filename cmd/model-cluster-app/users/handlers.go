package users

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type createArgs struct {
	Email string
	Name  string
	URL   string
}

func handlePost(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var args createArgs
	err := decoder.Decode(&args)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	resourceID := Add(args.Email, args.Name, args.URL)
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
	resourceIDs := List()
	result, err := json.Marshal(resourceIDs)
	if err != nil {
		http.Error(w, "Failed to marshal resource list", 500)
		return
	}
	w.Write(result)
}

// RegisterHandlers adds HTTP handlers for all the supported methods
func RegisterHandlers(r *mux.Router) {
	r.HandleFunc("/users", handlePost).Methods("POST")
	r.HandleFunc("/users/{id}", handleGet).Methods("GET")
	r.HandleFunc("/users", handleList).Methods("GET")
}
