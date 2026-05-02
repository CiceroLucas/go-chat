package tests

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/CiceroLucas/go-chat/internal/delivery/websocket"
)

func TestNewHub(t *testing.T) {
	hub := websocket.NewHub(nil, nil)
	if hub == nil {
		t.Fatal("NewHub retornou nil")
	}
}

func TestWSMessageTypes(t *testing.T) {
	tests := []struct {
		name     string
		msgType  websocket.WSMessageType
		expected string
	}{
		{"Message", websocket.WSTypeMessage, "message"},
		{"JoinRoom", websocket.WSTypeJoinRoom, "join_room"},
		{"LeaveRoom", websocket.WSTypeLeaveRoom, "leave_room"},
		{"RoomHistory", websocket.WSTypeRoomHistory, "room_history"},
		{"System", websocket.WSTypeSystem, "system"},
		{"Error", websocket.WSTypeError, "error"},
		{"UserList", websocket.WSTypeUserList, "user_list"},
		{"RoomList", websocket.WSTypeRoomList, "room_list"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.msgType) != tt.expected {
				t.Errorf("esperado %q, obteve %q", tt.expected, tt.msgType)
			}
		})
	}
}

func TestNewSystemMessage(t *testing.T) {
	msg := websocket.NewSystemMessage("room-1", "Usuário entrou na sala")

	if msg.Type != websocket.WSTypeSystem {
		t.Errorf("tipo esperado %q, obteve %q", websocket.WSTypeSystem, msg.Type)
	}
	if msg.RoomID != "room-1" {
		t.Errorf("room_id esperado %q, obteve %q", "room-1", msg.RoomID)
	}
	if msg.Content != "Usuário entrou na sala" {
		t.Errorf("content esperado %q, obteve %q", "Usuário entrou na sala", msg.Content)
	}
	if msg.Username != "Sistema" {
		t.Errorf("username esperado %q, obteve %q", "Sistema", msg.Username)
	}
	if msg.Timestamp.IsZero() {
		t.Error("timestamp não deveria ser zero")
	}
}

func TestNewErrorMessage(t *testing.T) {
	msg := websocket.NewErrorMessage("Algo deu errado")

	if msg.Type != websocket.WSTypeError {
		t.Errorf("tipo esperado %q, obteve %q", websocket.WSTypeError, msg.Type)
	}
	if msg.Content != "Algo deu errado" {
		t.Errorf("content esperado %q, obteve %q", "Algo deu errado", msg.Content)
	}
}

func TestWSMessageSerialization(t *testing.T) {
	msg := &websocket.WSMessage{
		Type:      websocket.WSTypeMessage,
		RoomID:    "sala-teste",
		Content:   "Olá, mundo!",
		Username:  "usuario1",
		UserID:    "user-123",
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("erro ao serializar: %v", err)
	}

	var decoded websocket.WSMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("erro ao deserializar: %v", err)
	}

	if decoded.Type != msg.Type {
		t.Errorf("type: esperado %q, obteve %q", msg.Type, decoded.Type)
	}
	if decoded.RoomID != msg.RoomID {
		t.Errorf("room_id: esperado %q, obteve %q", msg.RoomID, decoded.RoomID)
	}
	if decoded.Content != msg.Content {
		t.Errorf("content: esperado %q, obteve %q", msg.Content, decoded.Content)
	}
	if decoded.Username != msg.Username {
		t.Errorf("username: esperado %q, obteve %q", msg.Username, decoded.Username)
	}
}

func TestWSMessageSerializationOmitEmpty(t *testing.T) {
	msg := &websocket.WSMessage{
		Type: websocket.WSTypeSystem,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("erro ao serializar: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("erro ao deserializar: %v", err)
	}

	if _, ok := decoded["room_id"]; ok {
		t.Error("room_id deveria ser omitido quando vazio")
	}
	if _, ok := decoded["content"]; ok {
		t.Error("content deveria ser omitido quando vazio")
	}
}

func TestHubGetRoomUsersEmpty(t *testing.T) {
	hub := websocket.NewHub(nil, nil)
	users := hub.GetRoomUsers("sala-inexistente")
	if users != nil {
		t.Errorf("esperado nil, obteve %v", users)
	}
}

func TestConcurrentMessageCreation(t *testing.T) {
	var wg sync.WaitGroup
	messages := make([]*websocket.WSMessage, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			messages[idx] = websocket.NewSystemMessage("room-1", "Mensagem concorrente")
		}(i)
	}

	wg.Wait()

	for i, msg := range messages {
		if msg == nil {
			t.Errorf("mensagem %d é nil", i)
		}
		if msg != nil && msg.Type != websocket.WSTypeSystem {
			t.Errorf("mensagem %d tem tipo errado: %q", i, msg.Type)
		}
	}
}
