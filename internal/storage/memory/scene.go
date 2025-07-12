package memory

import (
	"log"    // For logging messages
	"sync"   // For RWMutex to handle concurrent access

	"github.com/Vasu1712/scenyx-backend/internal/models" // Import the models package to use the Scene struct
	"github.com/google/uuid"                           // Import uuid to generate unique IDs
)

// SceneStore manages the storage and retrieval of Scene objects in memory.
type SceneStore struct {
	mu             sync.RWMutex                     // Read-write mutex for concurrent access to the scenes map
	scenes         map[string]*models.Scene         // Map to store scenes by their ID
	userSceneIndex map[string][]string              // userID -> []sceneID: maps a user to the scenes they created/joined
}

// NewSceneStore creates and returns a new instance of SceneStore.
func NewSceneStore() *SceneStore {
	return &SceneStore{
		scenes:         make(map[string]*models.Scene),     // Initialize the scenes map
		userSceneIndex: make(map[string][]string), // Initialize the user-scene index
	}
}

// CreateScene creates a new scene with the given name, artist name, and creator ID.
// It initializes the scene with the creator as the first joined user and sets listeners to 1.
// It returns the newly created Scene object.
func (s *SceneStore) CreateScene(name, artistName, creatorID string) *models.Scene {
	s.mu.Lock() // Acquire a write lock
	defer s.mu.Unlock() // Release the lock

	sceneID := uuid.NewString()

	// Initialize JoinedUserIDs with the creator's ID
	joinedUserIDs := []string{creatorID}

	scene := &models.Scene{
		ID:            sceneID,
		Name:          name,          // Corrected from SceneName to Name
		ArtistName:    artistName,
		CreatorID:     creatorID,
		JoinedUserIDs: joinedUserIDs, // Set the initial joined users
		Listeners:     1,             // Creator is the first listener
		ActiveUsers:   0,             // Active users start at 0, updated via WebSocket
	}

	s.scenes[sceneID] = scene
	s.userSceneIndex[creatorID] = append(s.userSceneIndex[creatorID], sceneID)

	log.Printf("Scene created: ID=%s, Name=%s, CreatorID=%s", scene.ID, scene.Name, scene.CreatorID)
	return scene
}

// GetScene retrieves a scene by its ID.
func (s *SceneStore) GetScene(sceneID string) *models.Scene {
	s.mu.RLock() // Acquire a read lock
	defer s.mu.RUnlock() // Release the lock

	return s.scenes[sceneID]
}

// GetScenesForUser retrieves all scenes created by a specific user ID.
func (s *SceneStore) GetScenesForUser(userID string) []*models.Scene {
	s.mu.RLock() // Acquire a read lock
	defer s.mu.RUnlock() // Release the lock

	var userScenes []*models.Scene
	for _, sceneID := range s.userSceneIndex[userID] {
		if scene, ok := s.scenes[sceneID]; ok {
			userScenes = append(userScenes, scene)
		}
	}
	return userScenes
}

// JoinScene adds a user to a scene's joined users list.
// It returns true if the user was successfully added, false if they were already joined or scene not found.
func (s *SceneStore) JoinScene(sceneID, userID string) bool {
	s.mu.Lock() // Acquire a write lock
	defer s.mu.Unlock() // Release the lock

	scene, ok := s.scenes[sceneID]
	if !ok {
		log.Printf("Attempted to join non-existent scene: %s", sceneID)
		return false // Scene not found
	}

	// Check if user is already joined
	for _, id := range scene.JoinedUserIDs {
		if id == userID {
			log.Printf("User %s already joined scene %s", userID, sceneID)
			return false // User already joined
		}
	}

	scene.JoinedUserIDs = append(scene.JoinedUserIDs, userID)
	scene.Listeners = len(scene.JoinedUserIDs) // Update listeners count
	s.userSceneIndex[userID] = append(s.userSceneIndex[userID], sceneID) // Add scene to user's index

	log.Printf("User %s joined scene %s. Total listeners: %d", userID, sceneID, scene.Listeners)
	return true
}

// LeaveScene removes a user from a scene's joined users list.
// It returns true if the user was successfully removed, false if they were not joined or scene not found.
func (s *SceneStore) LeaveScene(sceneID, userID string) bool {
	s.mu.Lock() // Acquire a write lock
	defer s.mu.Unlock() // Release the lock

	scene, ok := s.scenes[sceneID]
	if !ok {
		log.Printf("Attempted to leave non-existent scene: %s", sceneID)
		return false // Scene not found
	}

	found := false
	for i, id := range scene.JoinedUserIDs {
		if id == userID {
			// Remove user from JoinedUserIDs slice
			scene.JoinedUserIDs = append(scene.JoinedUserIDs[:i], scene.JoinedUserIDs[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		log.Printf("User %s was not found in scene %s joined users list", userID, sceneID)
		return false // User not found in joined list
	}

	scene.Listeners = len(scene.JoinedUserIDs) // Update listeners count

	// Remove scene from user's index
	userScenes := s.userSceneIndex[userID]
	for i, id := range userScenes {
		if id == sceneID {
			s.userSceneIndex[userID] = append(userScenes[:i], userScenes[i+1:]...)
			break
		}
	}

	log.Printf("User %s left scene %s. Total listeners: %d", userID, sceneID, scene.Listeners)
	return true
}
