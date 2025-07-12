package scenes

import (
	"encoding/json" // For encoding and decoding JSON
	"fmt"           // For string formatting, especially for redirects
	"log"           // For logging information
	"net/http"      // For HTTP request and response handling

	"github.com/Vasu1712/scenyx-backend/internal/models" // Import models package to use Scene struct
	"github.com/Vasu1712/scenyx-backend/internal/storage/postgres" // Import the postgres package to use PostgresSceneStore
	"github.com/Vasu1712/scenyx-backend/internal/ws"             // Import the WebSocket hub
	"github.com/gorilla/websocket"                              // WebSocket library
)

// SceneHandler holds the dependencies for handling scene-related HTTP requests.
type SceneHandler struct {
	Store *postgres.PostgresSceneStore // A pointer to the PostgresSceneStore to interact with scene data
	Hub   *ws.Hub                      // A pointer to the WebSocket Hub for active user tracking
}

// CreateScene handles the HTTP POST request to create a new scene.
// It expects a JSON payload in the request body with "name", "artistName", and "CreatorID" fields.
func (h *SceneHandler) CreateScene(w http.ResponseWriter, r *http.Request) {
	// Define a struct to parse the incoming JSON request body
	var req struct {
		Name       string `json:"name"`
		ArtistName string `json:"artistName"` // Matches models.Scene and frontend payload
		CreatorID  string `json:"CreatorID"`  // Matches models.Scene and frontend payload
	}

	// Decode the JSON request body into the req struct
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		log.Printf("Error decoding request body for CreateScene: %v", err)
		return
	}

	// Validate input: ensure name, artistName, and CreatorID are not empty
	if req.Name == "" || req.ArtistName == "" || req.CreatorID == "" {
		http.Error(w, "Scene Name, Artist Name, and Creator ID cannot be empty", http.StatusBadRequest)
		log.Println("Validation error: Scene Name, Artist Name, or Creator ID is empty")
		return
	}

	// Call the CreateScene method on the SceneStore to save the new scene.
	scene := h.Store.CreateScene(req.Name, req.ArtistName, req.CreatorID)
	if scene == nil {
		http.Error(w, "Failed to create scene", http.StatusInternalServerError)
		return
	}

	// Set the Content-Type header to application/json for the response
	w.Header().Set("Content-Type", "application/json")
	// Set the HTTP status code to 201 Created
	w.WriteHeader(http.StatusCreated)
	// Encode the created scene object into JSON and write it to the response body
	json.NewEncoder(w).Encode(scene)

	log.Printf("Created scene: ID=%s, Name=%s, Artist=%s, CreatorID=%s, Listeners=%d",
		scene.ID, scene.Name, scene.ArtistName, scene.CreatorID, scene.Listeners)
}

// ListScenes handles the HTTP GET request to list all scenes associated with a user.
// It expects the user ID as a query parameter "user_id".
func (h *SceneHandler) ListScenes(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")

	if userID == "" {
		http.Error(w, "User ID is required as a query parameter (e.g., ?user_id=some_id)", http.StatusBadRequest)
		log.Println("Validation error: User ID is empty for ListScenes")
		return
	}

	scenes := h.Store.GetScenesForUser(userID)
	if scenes == nil { // Handle case where no scenes are found or an error occurred
		scenes = []*models.Scene{} // Return an empty slice instead of nil
	}

	// For each scene, dynamically update active users from the hub before sending
	for _, scene := range scenes {
		scene.ActiveUsers = h.Hub.GetActiveSceneUsersCount(scene.ID)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(scenes)

	log.Printf("Listed %d scenes for user ID: %s", len(scenes), userID)
}

// GetSceneData handles the HTTP POST request to get specific data for a scene.
// It expects a JSON payload in the request body with a "sceneID" field.
// It returns artistName, listeners, and activeUsers.
func (h *SceneHandler) GetSceneData(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SceneID string `json:"sceneID"` // Scene ID from the request body
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		log.Printf("Error decoding request body for GetSceneData: %v", err)
		return
	}

	if req.SceneID == "" {
		http.Error(w, "Scene ID cannot be empty", http.StatusBadRequest)
		log.Println("Validation error: Scene ID is empty for GetSceneData")
		return
	}

	scene := h.Store.GetScene(req.SceneID)
	if scene == nil {
		http.Error(w, "Scene not found", http.StatusNotFound)
		log.Printf("Scene not found for ID: %s", req.SceneID)
		return
	}

	// Dynamically get the active users count from the hub
	activeUsers := h.Hub.GetActiveSceneUsersCount(scene.ID)

	// Define the response struct to match the desired output format and frontend's expectations
	var res struct {
		Name       	 string `json:"name"`
		ArtistName   string `json:"artistName"`
		Listeners    int    `json:"listeners"`
		ActiveUsers  int    `json:"activeUsers"`
	}

	res.Name = scene.Name
	res.ArtistName = scene.ArtistName
	res.Listeners = scene.Listeners // This is now derived from len(scene.JoinedUserIDs)
	res.ActiveUsers = activeUsers   // This is now from the WebSocket hub

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)

	log.Printf("Retrieved data for scene ID: %s (Listeners: %d, ActiveUsers: %d)", req.SceneID, res.Listeners, res.ActiveUsers)
}

// JoinScene handles the HTTP POST request to add a user to a scene's joined listeners.
// It expects a JSON payload with "sceneID" and "userID".
func (h *SceneHandler) JoinScene(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SceneID string `json:"sceneID"`
		UserID  string `json:"userID"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		log.Printf("Error decoding request body for JoinScene: %v", err)
		return
	}

	if req.SceneID == "" || req.UserID == "" {
		http.Error(w, "Scene ID and User ID cannot be empty", http.StatusBadRequest)
		log.Println("Validation error: Scene ID or User ID is empty for JoinScene")
		return
	}

	if h.Store.JoinScene(req.SceneID, req.UserID) {
		scene := h.Store.GetScene(req.SceneID) // Get updated scene to return current listener count
		if scene == nil {
			http.Error(w, "Scene not found after join operation", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":   "User joined scene successfully",
			"listeners": scene.Listeners,
		})
	} else {
		http.Error(w, "Failed to join scene or user already joined", http.StatusConflict)
	}
}

// LeaveScene handles the HTTP POST request to remove a user from a scene's joined listeners.
// It expects a JSON payload with "sceneID" and "userID".
func (h *SceneHandler) LeaveScene(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SceneID string `json:"sceneID"`
		UserID  string `json:"userID"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		log.Printf("Error decoding request body for LeaveScene: %v", err)
		return
	}

	if req.SceneID == "" || req.UserID == "" {
		http.Error(w, "Scene ID and User ID cannot be empty", http.StatusBadRequest)
		log.Println("Validation error: Scene ID or User ID is empty for LeaveScene")
		return
	}

	if h.Store.LeaveScene(req.SceneID, req.UserID) {
		scene := h.Store.GetScene(req.SceneID) // Get updated scene to return current listener count
		if scene == nil {
			// This case means the scene might have been deleted or an error occurred after leaving
			http.Error(w, "Scene not found or error after leave operation", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":   "User left scene successfully",
			"listeners": scene.Listeners,
		})
	} else {
		http.Error(w, "Failed to leave scene or user not found in joined list", http.StatusConflict)
	}
}

// GenerateShareLink confirms a scene exists and returns its ID for link generation.
// This is a GET request, taking scene_id as a query parameter.
func (h *SceneHandler) GenerateShareLink(w http.ResponseWriter, r *http.Request) {
	sceneID := r.URL.Query().Get("scene_id")

	if sceneID == "" {
		http.Error(w, "Scene ID is required as a query parameter", http.StatusBadRequest)
		log.Println("Validation error: Scene ID is empty for GenerateShareLink")
		return
	}

	scene := h.Store.GetScene(sceneID)
	if scene == nil {
		http.Error(w, "Scene not found", http.StatusNotFound)
		log.Printf("Scene not found for ID: %s", sceneID)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"sceneID": scene.ID,
		"message": "Scene exists and can be shared.",
	})
	log.Printf("Share link requested for scene ID: %s", sceneID)
}

// JoinSceneByLink handles a user joining a scene via a shared URL.
// It expects scene_id and user_id as query parameters.
// After processing, it redirects the user to a frontend scene view.
func (h *SceneHandler) JoinSceneByLink(w http.ResponseWriter, r *http.Request) {
	sceneID := r.URL.Query().Get("scene_id")
	userID := r.URL.Query().Get("user_id") // Assuming user ID is available from frontend or session

	if sceneID == "" || userID == "" {
		http.Error(w, "Scene ID and User ID are required as query parameters", http.StatusBadRequest)
		log.Println("Validation error: Scene ID or User ID missing for JoinSceneByLink")
		return
	}

	// Check if the scene exists
	scene := h.Store.GetScene(sceneID)
	if scene == nil {
		http.Error(w, "Scene not found", http.StatusNotFound)
		log.Printf("Attempted to join non-existent scene via link: %s", sceneID)
		return
	}

	// Attempt to add the user to the scene's joined listeners
	joined := h.Store.JoinScene(sceneID, userID)

	if joined {
		log.Printf("User %s successfully joined scene %s via link.", userID, sceneID)
	} else {
		log.Printf("User %s was already in scene %s or failed to join via link.", userID, sceneID)
	}

	// ** IMPORTANT: Redirect to your frontend scene view **
	// You need to replace "http://127.0.0.1:5173/scene-view" with the actual URL
	// of your frontend page that displays the scene, passing the sceneID.
	frontendSceneURL := fmt.Sprintf("http://127.0.0.1:5173/scene-view?scene_id=%s", sceneID)
	http.Redirect(w, r, frontendSceneURL, http.StatusFound) // 302 Found for temporary redirect
}

// WebSocket handler for scenes
var sceneUpgrader = websocket.Upgrader{} // Use a separate upgrader for scenes if needed, or reuse DM one.

func (h *SceneHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	sceneID := r.URL.Query().Get("scene_id")
	userID := r.URL.Query().Get("user_id") // Assume user ID is passed for tracking active users

	if sceneID == "" || userID == "" {
		http.Error(w, "Scene ID and User ID are required for WebSocket connection", http.StatusBadRequest)
		log.Println("Validation error: Scene ID or User ID missing for Scene WS")
		return
	}

	conn, err := sceneUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket for scene %s: %v", sceneID, err)
		return
	}
	log.Printf("WebSocket connection upgraded for SceneID: %s, UserID: %s", sceneID, userID)

	client := &ws.Client{
		UserID:  userID,
		SceneID: sceneID, // Set the SceneID for this client
		Send:    make(chan []byte, 256),
		Conn:    conn,
	}
	h.Hub.Register <- client

	// Read pump: reads messages from the WebSocket connection
	go func() {
		defer func() {
			h.Hub.Unregister <- client
			conn.Close()
			log.Printf("Read pump closed for client %s in scene %s", userID, sceneID)
		}()
		for {
			// In a real application, you might expect messages from the client (e.g., chat messages within the scene)
			// For now, we just read to keep the connection alive and detect disconnections.
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket read error for client %s in scene %s: %v", userID, sceneID, err)
				}
				break
			}
			// If you want to broadcast messages received from clients in a scene:
			// h.Hub.Broadcast <- ws.BroadcastMessage{SceneID: sceneID, Data: message}
		}
	}()

	// Write pump: writes messages from the hub to the WebSocket connection
	go func() {
		defer func() {
			conn.Close()
			log.Printf("Write pump closed for client %s in scene %s", userID, sceneID)
		}()
		for message := range client.Send {
			err := conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Printf("WebSocket write error for client %s in scene %s: %v", userID, sceneID, err)
				return // Break from loop if write fails
			}
		}
	}()
}
