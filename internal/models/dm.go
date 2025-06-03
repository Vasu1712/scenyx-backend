package models

type DMMessage struct {
	ID        string `json:"id"`
	SenderID  string `json:"sender_id"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

type DMConversation struct {
	ID           string      `json:"id"`
	Participants [2]string   `json:"participants"` // Always 2 for DM
	Messages     []DMMessage `json:"messages"`
}
