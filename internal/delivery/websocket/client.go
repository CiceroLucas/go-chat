package websocket

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/CiceroLucas/go-chat/internal/usecase"
	ws "github.com/gorilla/websocket"
)

const (
	writeWait = 10 * time.Second

	pongWait = 60 * time.Second

	pingPeriod = (pongWait * 9) / 10

	maxMessageSize = 4096

	sendBufferSize = 256
)

type Client struct {
	hub         *Hub
	conn        *ws.Conn
	send        chan []byte
	userID      string
	username    string
	currentRoom string
	messageUC   *usecase.MessageUseCase
}

func NewClient(hub *Hub, conn *ws.Conn, userID, username string, messageUC *usecase.MessageUseCase) *Client {
	return &Client{
		hub:       hub,
		conn:      conn,
		send:      make(chan []byte, sendBufferSize),
		userID:    userID,
		username:  username,
		messageUC: messageUC,
	}
}

func (c *Client) ReadPump() {
	defer func() {

		c.hub.RemoveClientFromAllRooms(c)
		c.conn.Close()
		log.Printf("🔌 ReadPump encerrado para [%s]", c.username)
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, rawMessage, err := c.conn.ReadMessage()
		if err != nil {
			if ws.IsUnexpectedCloseError(err, ws.CloseGoingAway, ws.CloseNormalClosure) {
				log.Printf("⚠️ Erro de leitura [%s]: %v", c.username, err)
			}
			break
		}

		var msg WSMessage
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			log.Printf("⚠️ Mensagem inválida de [%s]: %v", c.username, err)
			errMsg := NewErrorMessage("Formato de mensagem inválido")
			data, _ := json.Marshal(errMsg)
			c.send <- data
			continue
		}

		c.handleMessage(&msg)
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
		log.Printf("🔌 WritePump encerrado para [%s]", c.username)
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {

				c.conn.WriteMessage(ws.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(ws.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte("\n"))
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:

			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(ws.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleMessage(msg *WSMessage) {

	msg.UserID = c.userID
	msg.Username = c.username
	msg.Timestamp = time.Now()

	switch msg.Type {
	case WSTypeJoinRoom:
		c.handleJoinRoom(msg)

	case WSTypeLeaveRoom:
		c.handleLeaveRoom(msg)

	case WSTypeMessage:
		c.handleChatMessage(msg)

	case WSTypeRoomHistory:
		c.handleRoomHistory(msg)

	default:
		errMsg := NewErrorMessage("Tipo de mensagem desconhecido: " + string(msg.Type))
		data, _ := json.Marshal(errMsg)
		c.send <- data
	}
}

func (c *Client) handleJoinRoom(msg *WSMessage) {
	if msg.RoomID == "" {
		errMsg := NewErrorMessage("ID da sala é obrigatório")
		data, _ := json.Marshal(errMsg)
		c.send <- data
		return
	}

	if c.currentRoom != "" && c.currentRoom != msg.RoomID {
		c.hub.unregister <- &ClientRegistration{Client: c, RoomID: c.currentRoom}
	}

	c.hub.register <- &ClientRegistration{Client: c, RoomID: msg.RoomID}

	log.Printf("📥 [%s] solicitou entrada na sala [%s]", c.username, msg.RoomID)
}

func (c *Client) handleLeaveRoom(msg *WSMessage) {
	if msg.RoomID == "" {
		return
	}
	c.hub.unregister <- &ClientRegistration{Client: c, RoomID: msg.RoomID}
	c.currentRoom = ""
	log.Printf("📤 [%s] saiu da sala [%s]", c.username, msg.RoomID)
}

func (c *Client) handleChatMessage(msg *WSMessage) {
	if msg.RoomID == "" || msg.Content == "" {
		errMsg := NewErrorMessage("Sala e conteúdo são obrigatórios")
		data, _ := json.Marshal(errMsg)
		c.send <- data
		return
	}

	if len(msg.Content) > 5000 {
		errMsg := NewErrorMessage("Mensagem muito longa (máximo 5000 caracteres)")
		data, _ := json.Marshal(errMsg)
		c.send <- data
		return
	}

	c.hub.broadcast <- msg
}

func (c *Client) handleRoomHistory(msg *WSMessage) {
	if msg.RoomID == "" || c.messageUC == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	messages, err := c.messageUC.GetHistory(ctx, msg.RoomID, 50, 0)
	if err != nil {
		log.Printf("⚠️ Erro ao buscar histórico: %v", err)
		return
	}

	historyData, _ := json.Marshal(messages)
	response := &WSMessage{
		Type:      WSTypeRoomHistory,
		RoomID:    msg.RoomID,
		Data:      historyData,
		Timestamp: time.Now(),
	}

	data, _ := json.Marshal(response)
	c.send <- data
}
