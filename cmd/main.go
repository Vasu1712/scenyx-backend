package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Vasu1712/scenyx-backend/internal/api/dms"
	"github.com/Vasu1712/scenyx-backend/internal/api/scenes"
	"github.com/Vasu1712/scenyx-backend/internal/middleware"
	"github.com/Vasu1712/scenyx-backend/internal/storage/postgres" // Import postgres package
	"github.com/Vasu1712/scenyx-backend/internal/ws"
)

func main() {
	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	// --- Database Setup ---
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set. Please provide the PostgreSQL connection string.")
	}

	// Initialize Postgres Scene Store
	sceneStore, err := postgres.NewPostgresSceneStore(databaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize PostgreSQL scene store: %v", err)
	}
	defer sceneStore.Close() // Ensure the database connection is closed when main exits

	// Initialize Postgres DM Store
	dmStore, err := postgres.NewPostgresDMStore(databaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize PostgreSQL DM store: %v", err)
	}
	defer dmStore.Close() // Ensure the database connection is closed when main exits


	// --- WebSocket Hub Setup ---
	hub := ws.NewHub()
	go hub.Run() // Start the WebSocket hub in a goroutine

	// --- Handlers Setup ---
	// Pass the PostgreSQL-backed stores to your handlers
	dmHandler := &dms.DMHandler{Store: dmStore, Hub: hub}
	sceneHandler := &scenes.SceneHandler{Store: sceneStore, Hub: hub}

	// --- HTTP Server Setup ---
	mux := http.NewServeMux()

	// Register routes for DMS
	dms.RegisterDMRoutes(mux, dmHandler)
	// Register routes for Scenes
	scenes.RegisterSceneRoutes(mux, sceneHandler)

	// Optional: catch-all logging for 404s
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[404] %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	})

	// Apply the CORS middleware to the entire multiplexer
	// (Assuming middleware.CORS is correctly defined in internal/middleware/cors.go)
	corsMux := middleware.CORS(mux)

	log.Printf("Scenyx backend listening on :%s", port)
	err = http.ListenAndServe(":"+port, corsMux) // Use corsMux here
	if err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
