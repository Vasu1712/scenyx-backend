package scenes

import (
	"log"      // For logging messages
	"net/http" // For HTTP request and response handling
)

// RegisterSceneRoutes registers all scene-related HTTP routes with the provided ServeMux.
func RegisterSceneRoutes(mux *http.ServeMux, handler *SceneHandler) {
	// Register the handler for the "/api/v1/scenes/create" endpoint.
	// This route is used to create a new scene.
	mux.HandleFunc("/api/v1/scenes/create", func(w http.ResponseWriter, r *http.Request) {
		// Ensure that only POST requests are allowed for this endpoint.
		if r.Method != http.MethodPost {
			// If the method is not POST, return a "Method Not Allowed" error.
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			log.Printf("[Scene] Method Not Allowed: %s %s", r.Method, r.URL.Path)
			return
		}
		// Log the incoming request.
		log.Printf("[Scene] %s %s", r.Method, r.URL.Path)
		// Call the CreateScene method of the SceneHandler to process the request.
		handler.CreateScene(w, r)
	})

	mux.HandleFunc("/api/v1/scenes/list", func(w http.ResponseWriter, r *http.Request) {
		// Ensure that only GET requests are allowed for this endpoint.
		if r.Method != http.MethodGet {
			// If the method is not GET, return a "Method Not Allowed" error.
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			log.Printf("[Scene] Method Not Allowed: %s %s", r.Method, r.URL.Path)
			return
		}
		// Log the incoming request.
		log.Printf("[Scene] %s %s", r.Method, r.URL.Path)
		// Call the ListScenes method of the SceneHandler to process the request.
		handler.ListScenes(w, r)
	})

	mux.HandleFunc("/api/v1/scenes/data", func(w http.ResponseWriter, r *http.Request) {
		// Ensure that only POST requests are allowed for this endpoint as it takes a body.
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			log.Printf("[Scene] Method Not Allowed: %s %s", r.Method, r.URL.Path)
			return
		}
		// Log the incoming request.
		log.Printf("[Scene] %s %s", r.Method, r.URL.Path)
		// Call the GetSceneData method of the SceneHandler to process the request.
		handler.GetSceneData(w, r)
	})

		mux.HandleFunc("/api/v1/scenes/join", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			log.Printf("[Scene] Method Not Allowed: %s %s", r.Method, r.URL.Path)
			return
		}
		log.Printf("[Scene] %s %s", r.Method, r.URL.Path)
		handler.JoinScene(w, r)
	})

	// New route to allow a user to leave a scene
	mux.HandleFunc("/api/v1/scenes/leave", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			log.Printf("[Scene] Method Not Allowed: %s %s", r.Method, r.URL.Path)
			return
		}
		log.Printf("[Scene] %s %s", r.Method, r.URL.Path)
		handler.LeaveScene(w, r)
	})

	// New WebSocket route for scene real-time updates
	mux.HandleFunc("/ws/scenes", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[Scene] WebSocket %s", r.URL.String())
		handler.ServeWS(w, r)
	})

	mux.HandleFunc("/api/v1/scenes/generate-share-link", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet { // This is a GET request, as it just retrieves info
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			log.Printf("[Scene] Method Not Allowed: %s %s", r.Method, r.URL.Path)
			return
		}
		log.Printf("[Scene] %s %s", r.Method, r.URL.Path)
		handler.GenerateShareLink(w, r)
	})

	// New route for a user to join a scene by clicking a shared link
	mux.HandleFunc("/api/v1/scenes/join-by-link", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet { // This is a GET request, as it's a direct URL hit
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			log.Printf("[Scene] Method Not Allowed: %s %s", r.Method, r.URL.Path)
			return
		}
		log.Printf("[Scene] %s %s", r.Method, r.URL.Path)
		handler.JoinSceneByLink(w, r)
	})
}


