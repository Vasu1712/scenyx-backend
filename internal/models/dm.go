package models

import "time"

type DMMessage struct {
    ID             string    `json:"id"`
    DMConversationID string    `json:"dm_conversation_id"`
    SenderID       string    `json:"sender_id"`
    Content        string    `json:"content"`
    Timestamp      time.Time `json:"timestamp"`
}

type DMConversation struct {
    ID             string    `json:"id"`
    Participants   [2]string `json:"participants"`
    CreatedAt      time.Time `json:"createdAt"`
    UpdatedAt      time.Time `json:"updatedAt"`
}