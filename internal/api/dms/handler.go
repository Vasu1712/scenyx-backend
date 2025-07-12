package dms

import (
	"encoding/json"
	"net/http"

	"github.com/Vasu1712/scenyx-backend/internal/storage/postgres"
	"github.com/Vasu1712/scenyx-backend/internal/ws"
	"github.com/gorilla/websocket"
)

type DMHandler struct {
	Store *postgres.PostgresDMStore
	Hub   *ws.Hub
}

func (h *DMHandler) StartOrGetConversation(w http.ResponseWriter, r *http.Request) {
	// Assume user IDs are in POST body or JWT
	var req struct {
		User1 string `json:"user1"`
		User2 string `json:"user2"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	conv := h.Store.StartOrGetConversation(req.User1, req.User2)
	json.NewEncoder(w).Encode(conv)
}

func (h *DMHandler) ListConversations(w http.ResponseWriter, r *http.Request) {
	// Assume userID from JWT or query param
	userID := r.URL.Query().Get("user_id")
	convs := h.Store.GetConversations(userID)
	json.NewEncoder(w).Encode(convs)
}

func (h *DMHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	dmID := r.URL.Query().Get("dm_id")
	msgs := h.Store.GetMessages(dmID)
	json.NewEncoder(w).Encode(msgs)
}

func (h *DMHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DMID     string `json:"dm_id"`
		SenderID string `json:"sender_id"`
		Content  string `json:"content"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	msg := h.Store.AddMessage(req.DMID, req.SenderID, req.Content)
	// Broadcast via WebSocket
	data, _ := json.Marshal(msg)
	h.Hub.Broadcast <- ws.BroadcastMessage{DMID: req.DMID, Data: data}
	json.NewEncoder(w).Encode(msg)
}

// WebSocket handler
var upgrader = websocket.Upgrader{}

func (h *DMHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	dmID := r.URL.Query().Get("dm_id")
	userID := r.URL.Query().Get("user_id")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := &ws.Client{
		UserID: userID,
		DMID:   dmID,
		Send:   make(chan []byte, 256),
		Conn:   conn,
	}
	h.Hub.Register <- client

	// Read pump
	go func() {
		defer func() {
			h.Hub.Unregister <- client
			conn.Close()
		}()
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			h.Hub.Broadcast <- ws.BroadcastMessage{DMID: dmID, Data: msg}
		}
	}()
	// Write pump
	go func() {
		for message := range client.Send {
			conn.WriteMessage(websocket.TextMessage, message)
		}
	}()
}
