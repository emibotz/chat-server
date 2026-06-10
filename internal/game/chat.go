package game

import "github.com/google/uuid"

type ChatMessage struct {
	senderID uuid.UUID
	message  string
}

func (m *ChatMessage) GetSenderID() uuid.UUID {
	return m.senderID
}

func (m *ChatMessage) GetMessage() string {
	return m.message
}
