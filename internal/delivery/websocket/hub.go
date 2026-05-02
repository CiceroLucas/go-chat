package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/CiceroLucas/go-chat/internal/domain"
	"github.com/CiceroLucas/go-chat/internal/usecase"
	"github.com/google/uuid"
)

type Hub struct {
	rooms map[string]map[*Client]bool

	register chan *ClientRegistration

	unregister chan *ClientRegistration

	broadcast chan *WSMessage

	mu sync.RWMutex

	messageUC *usecase.MessageUseCase
	roomUC    *usecase.RoomUseCase
}

type ClientRegistration struct {
	Client *Client
	RoomID string
}

func NewHub(messageUC *usecase.MessageUseCase, roomUC *usecase.RoomUseCase) *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *ClientRegistration),
		unregister: make(chan *ClientRegistration),
		broadcast:  make(chan *WSMessage, 256),
		messageUC:  messageUC,
		roomUC:     roomUC,
	}
}

func (h *Hub) Run() {
	log.Println("🔄 Hub iniciado — aguardando conexões...")
	for {
		select {
		case reg := <-h.register:
			h.handleRegister(reg)

		case reg := <-h.unregister:
			h.handleUnregister(reg)

		case msg := <-h.broadcast:
			h.handleBroadcast(msg)
		}
	}
}

func (h *Hub) handleRegister(reg *ClientRegistration) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.rooms[reg.RoomID]; !ok {
		h.rooms[reg.RoomID] = make(map[*Client]bool)
	}

	h.rooms[reg.RoomID][reg.Client] = true
	reg.Client.currentRoom = reg.RoomID

	log.Printf("👤 [%s] entrou na sala [%s] — %d usuários online",
		reg.Client.username, reg.RoomID, len(h.rooms[reg.RoomID]))

	sysMsg := NewSystemMessage(reg.RoomID, reg.Client.username+" entrou na sala")
	h.broadcastToRoom(reg.RoomID, sysMsg, reg.Client)

	h.sendUserList(reg.RoomID)
}

func (h *Hub) handleUnregister(reg *ClientRegistration) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.rooms[reg.RoomID]; ok {
		if _, exists := clients[reg.Client]; exists {
			delete(clients, reg.Client)

			log.Printf("👋 [%s] saiu da sala [%s] — %d usuários restantes",
				reg.Client.username, reg.RoomID, len(clients))

			if len(clients) == 0 {

			}

			sysMsg := NewSystemMessage(reg.RoomID, reg.Client.username+" saiu da sala")
			h.broadcastToRoom(reg.RoomID, sysMsg, nil)

			h.sendUserList(reg.RoomID)
		}
	}
}

func (h *Hub) handleBroadcast(msg *WSMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if msg.Type == WSTypeMessage && h.messageUC != nil {
		go func() {
			domainMsg := &domain.Message{
				ID:        uuid.New().String(),
				RoomID:    msg.RoomID,
				UserID:    msg.UserID,
				Username:  msg.Username,
				Content:   msg.Content,
				Type:      domain.MessageTypeText,
				CreatedAt: msg.Timestamp,
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := h.messageUC.SaveMessage(ctx, domainMsg); err != nil {
				log.Printf("⚠️ Erro ao salvar mensagem: %v", err)
			}
		}()
	}

	h.broadcastToRoom(msg.RoomID, msg, nil)
}

func (h *Hub) broadcastToRoom(roomID string, msg *WSMessage, exclude *Client) {
	clients, ok := h.rooms[roomID]
	if !ok {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("⚠️ Erro ao serializar mensagem: %v", err)
		return
	}

	for client := range clients {
		if client == exclude {
			continue
		}
		select {
		case client.send <- data:
		default:

			log.Printf("⚠️ Buffer cheio para [%s], desconectando", client.username)
			close(client.send)
			delete(clients, client)
		}
	}
}

func (h *Hub) sendUserList(roomID string) {
	clients, ok := h.rooms[roomID]
	if !ok {
		return
	}

	users := make([]string, 0, len(clients))
	for client := range clients {
		users = append(users, client.username)
	}

	userData, _ := json.Marshal(users)
	msg := &WSMessage{
		Type:      WSTypeUserList,
		RoomID:    roomID,
		Data:      userData,
		Timestamp: time.Now(),
	}

	data, _ := json.Marshal(msg)
	for client := range clients {
		select {
		case client.send <- data:
		default:
		}
	}
}

func (h *Hub) RemoveClientFromAllRooms(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for roomID, clients := range h.rooms {
		if _, exists := clients[client]; exists {
			delete(clients, client)
			log.Printf("🔌 [%s] desconectou da sala [%s]", client.username, roomID)

			sysMsg := NewSystemMessage(roomID, client.username+" desconectou")
			go func(rid string) {
				h.mu.RLock()
				defer h.mu.RUnlock()
				h.broadcastToRoomUnsafe(rid, sysMsg, nil)
				h.sendUserListUnsafe(rid)
			}(roomID)
		}
	}
}

func (h *Hub) broadcastToRoomUnsafe(roomID string, msg *WSMessage, exclude *Client) {
	clients, ok := h.rooms[roomID]
	if !ok {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	for client := range clients {
		if client == exclude {
			continue
		}
		select {
		case client.send <- data:
		default:
		}
	}
}

func (h *Hub) sendUserListUnsafe(roomID string) {
	clients, ok := h.rooms[roomID]
	if !ok {
		return
	}

	users := make([]string, 0, len(clients))
	for client := range clients {
		users = append(users, client.username)
	}

	userData, _ := json.Marshal(users)
	msg := &WSMessage{
		Type:      WSTypeUserList,
		RoomID:    roomID,
		Data:      userData,
		Timestamp: time.Now(),
	}

	data, _ := json.Marshal(msg)
	for client := range clients {
		select {
		case client.send <- data:
		default:
		}
	}
}

func (h *Hub) GetRoomUsers(roomID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.rooms[roomID]
	if !ok {
		return nil
	}

	users := make([]string, 0, len(clients))
	for client := range clients {
		users = append(users, client.username)
	}
	return users
}
