package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Vasu1712/scenyx-backend/internal/api/dms"
	"github.com/Vasu1712/scenyx-backend/internal/api/scenes"
	"github.com/Vasu1712/scenyx-backend/internal/middleware"
	"github.com/Vasu1712/scenyx-backend/internal/storage/memory"
	"github.com/Vasu1712/scenyx-backend/internal/ws"
)

func main() {
	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	// Initialize the central WebSocket Hub
	hub := ws.NewHub()
	go hub.Run() // Start the WebSocket hub in a goroutine

	// Initialize DM components
	dmStore := memory.NewDMStore()
	dmHandler := &dms.DMHandler{Store: dmStore, Hub: hub} // Pass the hub to DM handler

	// Initialize Scene components
	sceneStore := memory.NewSceneStore()
	sceneHandler := &scenes.SceneHandler{Store: sceneStore, Hub: hub} // Pass the hub to Scene handler

	mux := http.NewServeMux()

	// Register DM routes
	dms.RegisterDMRoutes(mux, dmHandler)

	// Register Scene routes
	scenes.RegisterSceneRoutes(mux, sceneHandler)

	// Optional: catch-all logging for 404s
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[404] %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	})

	// Apply the CORS middleware to the entire multiplexer
	corsMux := middleware.CORS(mux)

	log.Printf("Scenyx backend listening on :%s", port)
	err := http.ListenAndServe(":"+port, corsMux)
	if err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
