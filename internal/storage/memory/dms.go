package memory

import (
	"sync"
	"time"

	"github.com/Vasu1712/scenyx-backend/internal/models"
	"github.com/google/uuid"
)

type DMStore struct {
	mu            sync.RWMutex
	conversations map[string]*models.DMConversation // dmID -> conversation
	userIndex     map[string][]string               // userID -> []dmID
}

func NewDMStore() *DMStore {
	return &DMStore{
		conversations: make(map[string]*models.DMConversation),
		userIndex:     make(map[string][]string),
	}
}

func (s *DMStore) StartOrGetConversation(user1, user2 string) *models.DMConversation {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Check if conversation exists
	for _, dmID := range s.userIndex[user1] {
		conv := s.conversations[dmID]
		if (conv.Participants[0] == user1 && conv.Participants[1] == user2) ||
			(conv.Participants[0] == user2 && conv.Participants[1] == user1) {
			return conv
		}
	}
	// Create new conversation
	dmID := uuid.NewString()
	conv := &models.DMConversation{
		ID:           dmID,
		Participants: [2]string{user1, user2},
		Messages:     []models.DMMessage{},
	}
	s.conversations[dmID] = conv
	s.userIndex[user1] = append(s.userIndex[user1], dmID)
	s.userIndex[user2] = append(s.userIndex[user2], dmID)
	return conv
}

func (s *DMStore) GetConversations(userID string) []*models.DMConversation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*models.DMConversation
	for _, dmID := range s.userIndex[userID] {
		result = append(result, s.conversations[dmID])
	}
	return result
}

func (s *DMStore) AddMessage(dmID, senderID, content string) *models.DMMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	msg := models.DMMessage{
		ID:        uuid.NewString(),
		SenderID:  senderID,
		Content:   content,
		Timestamp: time.Now().Unix(),
	}
	conv, ok := s.conversations[dmID]
	if ok {
		conv.Messages = append(conv.Messages, msg)
	}
	return &msg
}

func (s *DMStore) GetMessages(dmID string) []models.DMMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if conv, ok := s.conversations[dmID]; ok {
		return conv.Messages
	}
	return nil
}
