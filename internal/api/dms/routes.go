package dms

import (
	"log"
	"net/http"
)

// RegisterDMRoutes registers all DM-related HTTP and WebSocket routes.
func RegisterDMRoutes(mux *http.ServeMux, handler *DMHandler) {
	mux.HandleFunc("/api/v1/dms/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		log.Printf("[DM] %s %s", r.Method, r.URL.Path)
		handler.StartOrGetConversation(w, r)
	})

	mux.HandleFunc("/api/v1/dms/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		log.Printf("[DM] %s %s", r.Method, r.URL.Path)
		handler.ListConversations(w, r)
	})

	mux.HandleFunc("/api/v1/dms/messages", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		log.Printf("[DM] %s %s", r.Method, r.URL.Path)
		handler.GetMessages(w, r)
	})

	mux.HandleFunc("/api/v1/dms/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		log.Printf("[DM] %s %s", r.Method, r.URL.Path)
		handler.SendMessage(w, r)
	})

	mux.HandleFunc("/ws/dms", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[DM] WebSocket %s", r.URL.String())
		handler.ServeWS(w, r)
	})
}
