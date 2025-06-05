package ws

import (
	"sync"
	"github.com/gorilla/websocket"
)

type Client struct {
	UserID   string
	DMID     string
	Send     chan []byte
	Conn     *websocket.Conn // interface for Gorilla/WebSocket
}

type Hub struct {
	Clients    map[string]map[*Client]bool // dmID -> clients
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan BroadcastMessage
	mu         sync.RWMutex
}

type BroadcastMessage struct {
	DMID string
	Data []byte
}

func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[string]map[*Client]bool),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan BroadcastMessage),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			if h.Clients[client.DMID] == nil {
				h.Clients[client.DMID] = make(map[*Client]bool)
			}
			h.Clients[client.DMID][client] = true
			h.mu.Unlock()
		case client := <-h.Unregister:
			h.mu.Lock()
			if clients, ok := h.Clients[client.DMID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.Send)
				}
			}
			h.mu.Unlock()
		case msg := <-h.Broadcast:
			h.mu.RLock()
			for client := range h.Clients[msg.DMID] {
				select {
				case client.Send <- msg.Data:
				default:
					close(client.Send)
					delete(h.Clients[msg.DMID], client)
				}
			}
			h.mu.RUnlock()
		}
	}
}
