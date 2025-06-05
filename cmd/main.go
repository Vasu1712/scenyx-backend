package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Vasu1712/scenyx-backend/internal/api/dms"
	"github.com/Vasu1712/scenyx-backend/internal/storage/memory"
	"github.com/Vasu1712/scenyx-backend/internal/ws"
)

func main() {
	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	dmStore := memory.NewDMStore()
	hub := ws.NewHub()
	go hub.Run()

	dmHandler := &dms.DMHandler{Store: dmStore, Hub: hub}

	mux := http.NewServeMux()
	dms.RegisterDMRoutes(mux, dmHandler)

	// Optional: catch-all logging for 404s
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[404] %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	})

	log.Printf("Scenyx backend listening on :%s", port)
	err := http.ListenAndServe(":"+port, mux)
	if err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
