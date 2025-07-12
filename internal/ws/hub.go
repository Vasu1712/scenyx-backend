package ws

import (
	"log"  // For logging messages
	"sync" // For RWMutex to handle concurrent access

	"github.com/gorilla/websocket" // WebSocket library
)

// Client represents a single WebSocket connection.
type Client struct {
	UserID string // ID of the user connected
	DMID   string // ID of the DM conversation this client is connected to (if any)
	SceneID string // ID of the Scene this client is connected to (if any)
	Send   chan []byte       // Buffered channel for outgoing messages
	Conn   *websocket.Conn   // The WebSocket connection
}

// Hub maintains the set of active clients and broadcasts messages to them.
type Hub struct {
	mu         sync.RWMutex                      // Read-write mutex for concurrent access to client maps
	DMClients  map[string]map[*Client]bool       // dmID -> clients connected to that DM
	SceneClients map[string]map[*Client]bool     // sceneID -> clients connected to that Scene
	Register   chan *Client                      // Channel for clients to register with the hub
	Unregister chan *Client                      // Channel for clients to unregister from the hub
	Broadcast  chan BroadcastMessage             // Channel for broadcasting messages
}

// BroadcastMessage contains the target ID (DM or Scene) and the data to broadcast.
type BroadcastMessage struct {
	DMID    string // DM ID for DM messages
	SceneID string // Scene ID for Scene messages
	Data    []byte // The actual message data
}

// NewHub creates and returns a new instance of Hub.
func NewHub() *Hub {
	return &Hub{
		DMClients:    make(map[string]map[*Client]bool),
		SceneClients: make(map[string]map[*Client]bool),
		Register:     make(chan *Client),
		Unregister:   make(chan *Client),
		Broadcast:    make(chan BroadcastMessage),
	}
}

// Run starts the hub's event loop, processing client registrations, unregistrations, and broadcasts.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock() // Acquire a write lock
			if client.DMID != "" {
				if h.DMClients[client.DMID] == nil {
					h.DMClients[client.DMID] = make(map[*Client]bool)
				}
				h.DMClients[client.DMID][client] = true
				log.Printf("Client %s registered to DM %s", client.UserID, client.DMID)
			}
			if client.SceneID != "" {
				if h.SceneClients[client.SceneID] == nil {
					h.SceneClients[client.SceneID] = make(map[*Client]bool)
				}
				h.SceneClients[client.SceneID][client] = true
				log.Printf("Client %s registered to Scene %s", client.UserID, client.SceneID)
			}
			h.mu.Unlock() // Release the lock

		case client := <-h.Unregister:
			h.mu.Lock() // Acquire a write lock
			if client.DMID != "" {
				if clients, ok := h.DMClients[client.DMID]; ok {
					if _, ok := clients[client]; ok {
						delete(clients, client)
						close(client.Send) // Close the send channel
						log.Printf("Client %s unregistered from DM %s", client.UserID, client.DMID)
					}
				}
			}
			if client.SceneID != "" {
				if clients, ok := h.SceneClients[client.SceneID]; ok {
					if _, ok := clients[client]; ok {
						delete(clients, client)
						// Only close client.Send once, if it hasn't been closed by DM unregister
						// This assumes a client is either in a DM or a Scene, or if in both,
						// the first unregister closes the channel. A more robust solution
						// would involve a reference counter or separate send channels.
						// For now, if client.Send is already closed, this will panic.
						// A simple check can prevent this:
						select {
						case <-client.Send: // Check if channel is already closed
						default:
							close(client.Send)
						}
						log.Printf("Client %s unregistered from Scene %s", client.UserID, client.SceneID)
					}
				}
			}
			h.mu.Unlock() // Release the lock

		case msg := <-h.Broadcast:
			h.mu.RLock() // Acquire a read lock
			if msg.DMID != "" {
				if clients, ok := h.DMClients[msg.DMID]; ok {
					for client := range clients {
						select {
						case client.Send <- msg.Data:
						default:
							// If sending fails, assume client is gone and unregister
							close(client.Send)
							delete(h.DMClients[msg.DMID], client)
							log.Printf("Failed to send to client %s in DM %s. Unregistering.", client.UserID, client.DMID)
						}
					}
				}
			}
			if msg.SceneID != "" {
				if clients, ok := h.SceneClients[msg.SceneID]; ok {
					for client := range clients {
						select {
						case client.Send <- msg.Data:
						default:
							// If sending fails, assume client is gone and unregister
							close(client.Send)
							delete(h.SceneClients[msg.SceneID], client)
							log.Printf("Failed to send to client %s in Scene %s. Unregistering.", client.UserID, client.SceneID)
						}
					}
				}
			}
			h.mu.RUnlock() // Release the lock
		}
	}
}

// GetActiveSceneUsersCount returns the number of active WebSocket connections for a given scene.
func (h *Hub) GetActiveSceneUsersCount(sceneID string) int {
	h.mu.RLock() // Acquire a read lock
	defer h.mu.RUnlock() // Release the lock

	if clients, ok := h.SceneClients[sceneID]; ok {
		return len(clients)
	}
	return 0
}
