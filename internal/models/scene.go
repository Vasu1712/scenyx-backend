package models

import "time"

// Scene represents a user-created scene with a unique ID, name, artist, creator,
// total listeners (derived), and active users (real-time via WebSocket).
type Scene struct {
	ID          string    `json:"id"`             // Unique identifier for the scene (UUID)
	Name        string    `json:"name"`           // Name of the scene
	ArtistName  string    `json:"artistName"`     // Name of the artist who created the scene
	CreatorID   string    `json:"CreatorID"`      // The ID of the user who created this scene
	Listeners   int       `json:"listeners"`      // Total number of listeners for the scene (derived from DB count)
	ActiveUsers int       `json:"activeUsers"`    // Number of active users currently in the scene (real-time via WebSocket)
	CreatedAt   time.Time `json:"createdAt"`      // Timestamp when the scene was created
	UpdatedAt   time.Time `json:"updatedAt"`      // Timestamp when the scene was last updated
}
