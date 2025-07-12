package models

// Scene represents a user-created scene with a unique ID, name, artist, creator,
// a list of joined user IDs, total listeners, and active users.
type Scene struct {
	ID           string   `json:"id"`             // Unique identifier for the scene (UUID)
	Name         string   `json:"name"`           // Name of the scene
	ArtistName   string   `json:"artistName"`     // Name of the artist who created the scene (matches frontend camelCase)
	CreatorID    string   `json:"CreatorID"`      // The ID of the user who created this scene (matches frontend camelCase)
	JoinedUserIDs []string `json:"-"`              // List of user IDs who have joined this scene. Excluded from JSON output.
	Listeners    int      `json:"listeners"`      // Total number of listeners for the scene (derived from JoinedUserIDs)
	ActiveUsers  int      `json:"activeUsers"`    // Number of active users currently in the scene (real-time via WebSocket)
}
