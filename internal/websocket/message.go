package websocket

import (
	"time"

	"github.com/google/uuid"
)

const (
	MessageTypeChat       = "chat_message"
	MessageTypeUserJoined = "user_joined"
	MessageTypeUserLeft   = "user_left"
	MessageTypePing       = "ping"
	MessageTypePong       = "pong"
	MessageTypeError      = "error"
)

type Message struct {
	ID        string `json:"id,omitempty"`
	Type      string `json:"type"`
	SenderID  string `json:"sender_id,omitempty"`
	Username  string `json:"username,omitempty"`
	Content   string `json:"content,omitempty"`
	Distance  string `json:"distance,omitempty"`
	Timestamp int64  `json:"timestamp"`
	Geohash   string `json:"-"` // Not exposed to clients
	UserCount int    `json:"user_count,omitempty"`
	ErrorCode string `json:"code,omitempty"`
}

type IncomingMessage struct {
	Type      string `json:"type"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

func NewChatMessage(senderID, username, content, geohash, distance string) *Message {
	return &Message{
		ID:        uuid.New().String(),
		Type:      MessageTypeChat,
		SenderID:  senderID,
		Username:  username,
		Content:   content,
		Distance:  distance,
		Geohash:   geohash,
		Timestamp: time.Now().Unix(),
	}
}

func NewErrorMessage(errMsg, code string) *Message {
	return &Message{
		Type:      MessageTypeError,
		Content:   errMsg,
		ErrorCode: code,
		Timestamp: time.Now().Unix(),
	}
}