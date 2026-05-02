package websocket

import (
	"encoding/json"
	"time"
)

type WSMessageType string

const (
	WSTypeMessage     WSMessageType = "message"
	WSTypeJoinRoom    WSMessageType = "join_room"
	WSTypeLeaveRoom   WSMessageType = "leave_room"
	WSTypeRoomHistory WSMessageType = "room_history"
	WSTypeSystem      WSMessageType = "system"
	WSTypeError       WSMessageType = "error"
	WSTypeUserList    WSMessageType = "user_list"
	WSTypeRoomList    WSMessageType = "room_list"
)

type WSMessage struct {
	Type      WSMessageType   `json:"type"`
	RoomID    string          `json:"room_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	Username  string          `json:"username,omitempty"`
	UserID    string          `json:"user_id,omitempty"`
	Timestamp time.Time       `json:"timestamp,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
}

func NewSystemMessage(roomID, content string) *WSMessage {
	return &WSMessage{
		Type:      WSTypeSystem,
		RoomID:    roomID,
		Content:   content,
		Username:  "Sistema",
		Timestamp: time.Now(),
	}
}

func NewErrorMessage(content string) *WSMessage {
	return &WSMessage{
		Type:      WSTypeError,
		Content:   content,
		Timestamp: time.Now(),
	}
}
