package postgres

import (
	"database/sql"
	"fmt"
	"log"
	"sort" // To ensure consistent participant order for unique constraint
	"time"

	"github.com/Vasu1712/scenyx-backend/internal/models"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// PostgresDMStore implements the DM storage interface using PostgreSQL.
type PostgresDMStore struct {
	db *sql.DB
}

// NewPostgresDMStore creates a new PostgresDMStore instance.
func NewPostgresDMStore(dataSourceName string) (*PostgresDMStore, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection for DMs: %w", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database for DMs: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("Successfully connected to PostgreSQL database for DMs.")

	return &PostgresDMStore{db: db}, nil
}

// StartOrGetConversation finds an existing conversation between two users or creates a new one.
func (s *PostgresDMStore) StartOrGetConversation(user1, user2 string) *models.DMConversation {
	// Ensure consistent order of participants to handle the UNIQUE constraint
	participants := []string{user1, user2}
	sort.Strings(participants)
	p1, p2 := participants[0], participants[1]

	conv := &models.DMConversation{}
	query := `
		SELECT id, participant1_id, participant2_id, created_at, updated_at
		FROM dm_conversations
		WHERE (participant1_id = $1 AND participant2_id = $2)
	`
	err := s.db.QueryRow(query, p1, p2).Scan(
		&conv.ID, &conv.Participants[0], &conv.Participants[1], &conv.CreatedAt, &conv.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Conversation does not exist, create a new one
		insertQuery := `
			INSERT INTO dm_conversations (participant1_id, participant2_id)
			VALUES ($1, $2)
			RETURNING id, participant1_id, participant2_id, created_at, updated_at
		`
		err = s.db.QueryRow(insertQuery, p1, p2).Scan(
			&conv.ID, &conv.Participants[0], &conv.Participants[1], &conv.CreatedAt, &conv.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error creating new DM conversation: %v", err)
			return nil
		}
		log.Printf("Created new DM conversation: %s between %s and %s", conv.ID, p1, p2)
		return conv
	} else if err != nil {
		log.Printf("Error getting DM conversation: %v", err)
		return nil
	}

	log.Printf("Retrieved existing DM conversation: %s between %s and %s", conv.ID, p1, p2)
	return conv
}

// GetConversations lists all conversations a user is a part of.
func (s *PostgresDMStore) GetConversations(userID string) []*models.DMConversation {
	var convs []*models.DMConversation
	query := `
		SELECT id, participant1_id, participant2_id, created_at, updated_at
		FROM dm_conversations
		WHERE participant1_id = $1 OR participant2_id = $1
		ORDER BY updated_at DESC
	`
	rows, err := s.db.Query(query, userID)
	if err != nil {
		log.Printf("Error getting conversations for user %s: %v", userID, err)
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		conv := &models.DMConversation{}
		err := rows.Scan(
			&conv.ID, &conv.Participants[0], &conv.Participants[1], &conv.CreatedAt, &conv.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning DM conversation row for user %s: %v", userID, err)
			continue
		}
		convs = append(convs, conv)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating DM conversation rows for user %s: %v", userID, err)
		return nil
	}
	return convs
}

// GetMessages retrieves all messages for a given conversation ID.
func (s *PostgresDMStore) GetMessages(dmID string) []models.DMMessage {
	var msgs []models.DMMessage
	query := `
		SELECT id, dm_conversation_id, sender_id, content, timestamp
		FROM dm_messages
		WHERE dm_conversation_id = $1
		ORDER BY timestamp ASC
	`
	rows, err := s.db.Query(query, dmID)
	if err != nil {
		log.Printf("Error getting messages for DM %s: %v", dmID, err)
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		msg := models.DMMessage{}
		err := rows.Scan(
			&msg.ID, &msg.DMConversationID, &msg.SenderID, &msg.Content, &msg.Timestamp,
		)
		if err != nil {
			log.Printf("Error scanning DM message row for DM %s: %v", dmID, err)
			continue
		}
		msgs = append(msgs, msg)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating DM message rows for DM %s: %v", dmID, err)
		return nil
	}
	return msgs
}

// AddMessage adds a new message to a conversation in the database.
func (s *PostgresDMStore) AddMessage(dmID, senderID, content string) *models.DMMessage {
	msg := &models.DMMessage{}
	query := `
		INSERT INTO dm_messages (dm_conversation_id, sender_id, content)
		VALUES ($1, $2, $3)
		RETURNING id, dm_conversation_id, sender_id, content, timestamp
	`
	err := s.db.QueryRow(query, dmID, senderID, content).Scan(
		&msg.ID, &msg.DMConversationID, &msg.SenderID, &msg.Content, &msg.Timestamp,
	)
	if err != nil {
		log.Printf("Error adding message to DM %s: %v", dmID, err)
		return nil
	}

	// Update the updated_at timestamp of the conversation
	updateConvQuery := `UPDATE dm_conversations SET updated_at = NOW() WHERE id = $1`
	_, err = s.db.Exec(updateConvQuery, dmID)
	if err != nil {
		log.Printf("Error updating conversation %s timestamp: %v", dmID, err)
		// This is non-fatal for message sending, but good to log
	}

	log.Printf("Added message %s to DM %s from sender %s", msg.ID, dmID, senderID)
	return msg
}

// Close closes the database connection.
func (s *PostgresDMStore) Close() error {
	return s.db.Close()
}
