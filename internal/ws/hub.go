package ws

import (
	"sync"
)

type Client struct {
	UserID   string
	DMID     string
	Send     chan []byte
	Conn     WebSocketConn // interface for Gorilla/WebSocket
}

type Hub struct {
	clients    map[string]map[*Client]bool // dmID -> clients
	register   chan *Client
	unregister chan *Client
	broadcast  chan BroadcastMessage
	mu         sync.RWMutex
}

type BroadcastMessage struct {
	DMID string
	Data []byte
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan BroadcastMessage),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.DMID] == nil {
				h.clients[client.DMID] = make(map[*Client]bool)
			}
			h.clients[client.DMID][client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.DMID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.Send)
				}
			}
			h.mu.Unlock()
		case msg := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients[msg.DMID] {
				select {
				case client.Send <- msg.Data:
				default:
					close(client.Send)
					delete(h.clients[msg.DMID], client)
				}
			}
			h.mu.RUnlock()
		}
	}
}
