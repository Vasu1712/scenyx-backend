package postgres

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/Vasu1712/scenyx-backend/internal/models"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// PostgresSceneStore implements the Scene storage interface using PostgreSQL.
type PostgresSceneStore struct {
	db *sql.DB
}

// NewPostgresSceneStore creates a new PostgresSceneStore instance.
// It takes a PostgreSQL connection string (DSN).
func NewPostgresSceneStore(dataSourceName string) (*PostgresSceneStore, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Ping the database to verify the connection
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set connection pool parameters (optional, but good for performance)
	db.SetMaxOpenConns(25) // Max number of open connections to the database
	db.SetMaxIdleConns(10) // Max number of idle connections in the pool
	db.SetConnMaxLifetime(5 * time.Minute) // Max lifetime for a connection

	log.Println("Successfully connected to PostgreSQL database for Scenes.")

	return &PostgresSceneStore{db: db}, nil
}

// CreateScene creates a new scene in the PostgreSQL database.
func (s *PostgresSceneStore) CreateScene(name, artistName, creatorID string) *models.Scene {
	scene := &models.Scene{}
	// Insert the new scene into the scenes table
	// RETURNING * will return all columns of the inserted row
	query := `INSERT INTO scenes (name, artist_name, creator_id) VALUES ($1, $2, $3) RETURNING id, name, artist_name, creator_id, created_at, updated_at`
	err := s.db.QueryRow(query, name, artistName, creatorID).Scan(
		&scene.ID, &scene.Name, &scene.ArtistName, &scene.CreatorID, &scene.CreatedAt, &scene.UpdatedAt,
	)
	if err != nil {
		log.Printf("Error creating scene in DB: %v", err)
		return nil
	}

	// Also add the creator as the first participant in scene_participants
	joinQuery := `INSERT INTO scene_participants (scene_id, user_id) VALUES ($1, $2) ON CONFLICT (scene_id, user_id) DO NOTHING`
	_, err = s.db.Exec(joinQuery, scene.ID, creatorID)
	if err != nil {
		log.Printf("Error adding creator %s to scene_participants for scene %s: %v", creatorID, scene.ID, err)
		// This is a non-fatal error for scene creation, but good to log
	} else {
		// Increment listeners count after successful join (for immediate return, though it's usually derived)
		scene.Listeners = 1
	}

	log.Printf("Scene created in DB: ID=%s, Name=%s, CreatorID=%s", scene.ID, scene.Name, scene.CreatorID)
	return scene
}

// GetScene retrieves a scene by its ID from the PostgreSQL database.
func (s *PostgresSceneStore) GetScene(sceneID string) *models.Scene {
	scene := &models.Scene{}
	query := `
		SELECT
			s.id, s.name, s.artist_name, s.creator_id,
			(SELECT COUNT(*) FROM scene_participants WHERE scene_id = s.id) AS listeners,
			s.active_users, s.created_at, s.updated_at
		FROM scenes s
		WHERE s.id = $1
	`
	err := s.db.QueryRow(query, sceneID).Scan(
		&scene.ID, &scene.Name, &scene.ArtistName, &scene.CreatorID,
		&scene.Listeners, &scene.ActiveUsers, &scene.CreatedAt, &scene.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil // Scene not found
	}
	if err != nil {
		log.Printf("Error getting scene %s from DB: %v", sceneID, err)
		return nil
	}
	return scene
}

// GetScenesForUser retrieves all scenes created by or joined by a specific user.
func (s *PostgresSceneStore) GetScenesForUser(userID string) []*models.Scene {
	var scenes []*models.Scene

	// Query for scenes created by the user OR where the user is a participant
	// Use UNION to combine results and DISTINCT to avoid duplicates if a user created and is also a participant (though unlikely for creator to be in scene_participants explicitly if always joining on creation).
	query := `
		SELECT DISTINCT ON (s.id)
			s.id, s.name, s.artist_name, s.creator_id,
			(SELECT COUNT(*) FROM scene_participants sp WHERE sp.scene_id = s.id) AS listeners,
			s.active_users, s.created_at, s.updated_at
		FROM scenes s
		LEFT JOIN scene_participants sp_join ON s.id = sp_join.scene_id
		WHERE s.creator_id = $1 OR sp_join.user_id = $1
		ORDER BY s.id, s.created_at DESC -- ORDER BY s.id is necessary for DISTINCT ON
	`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		log.Printf("Error getting scenes for user %s from DB: %v", userID, err)
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		scene := &models.Scene{}
		err := rows.Scan(
			&scene.ID, &scene.Name, &scene.ArtistName, &scene.CreatorID,
			&scene.Listeners, &scene.ActiveUsers, &scene.CreatedAt, &scene.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning scene row for user %s: %v", userID, err)
			continue
		}
		scenes = append(scenes, scene)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating scene rows for user %s: %v", userID, err)
		return nil
	}

	return scenes
}

// JoinScene adds a user to a scene's participants in the database.
func (s *PostgresSceneStore) JoinScene(sceneID, userID string) bool {
	// Check if the scene exists
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM scenes WHERE id = $1)", sceneID).Scan(&exists)
	if err != nil || !exists {
		log.Printf("Scene %s not found for join operation: %v", sceneID, err)
		return false
	}

	// Attempt to insert into scene_participants. ON CONFLICT DO NOTHING handles if user is already joined.
	query := `INSERT INTO scene_participants (scene_id, user_id) VALUES ($1, $2) ON CONFLICT (scene_id, user_id) DO NOTHING RETURNING scene_id`
	var insertedSceneID string
	err = s.db.QueryRow(query, sceneID, userID).Scan(&insertedSceneID)

	// If no row was returned (Scan error), it means ON CONFLICT DO NOTHING was triggered (user already joined)
	if err == sql.ErrNoRows {
		log.Printf("User %s already joined scene %s.", userID, sceneID)
		return false
	}
	if err != nil {
		log.Printf("Error joining user %s to scene %s in DB: %v", userID, sceneID, err)
		return false
	}

	log.Printf("User %s successfully joined scene %s.", userID, sceneID)
	return true
}

// LeaveScene removes a user from a scene's participants in the database.
func (s *PostgresSceneStore) LeaveScene(sceneID, userID string) bool {
	// Check if the scene exists
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM scenes WHERE id = $1)", sceneID).Scan(&exists)
	if err != nil || !exists {
		log.Printf("Scene %s not found for leave operation: %v", sceneID, err)
		return false
	}

	// Delete the participant entry
	result, err := s.db.Exec("DELETE FROM scene_participants WHERE scene_id = $1 AND user_id = $2", sceneID, userID)
	if err != nil {
		log.Printf("Error leaving user %s from scene %s in DB: %v", userID, sceneID, err)
		return false
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected after leaving scene: %v", err)
		return false
	}

	if rowsAffected == 0 {
		log.Printf("User %s was not found in scene %s participants to leave.", userID, sceneID)
		return false // User was not a participant
	}

	log.Printf("User %s successfully left scene %s.", userID, sceneID)
	return true
}

// Close closes the database connection.
func (s *PostgresSceneStore) Close() error {
	return s.db.Close()
}
