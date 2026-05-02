package websocket

import (
	"log"
	"net/http"

	"github.com/CiceroLucas/go-chat/internal/infrastructure/auth"
	"github.com/CiceroLucas/go-chat/internal/usecase"
	ws "github.com/gorilla/websocket"
)

var upgrader = ws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,

	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSHandler struct {
	hub        *Hub
	jwtService *auth.JWTService
	messageUC  *usecase.MessageUseCase
}

func NewWSHandler(hub *Hub, jwtService *auth.JWTService, messageUC *usecase.MessageUseCase) *WSHandler {
	return &WSHandler{
		hub:        hub,
		jwtService: jwtService,
		messageUC:  messageUC,
	}
}

func (h *WSHandler) ServeWS(w http.ResponseWriter, r *http.Request) {

	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		http.Error(w, "Token de autenticação obrigatório", http.StatusUnauthorized)
		return
	}

	claims, err := h.jwtService.ValidateToken(tokenStr)
	if err != nil {
		http.Error(w, "Token inválido ou expirado", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("⚠️ Erro no upgrade WebSocket: %v", err)
		return
	}

	client := NewClient(h.hub, conn, claims.UserID, claims.Username, h.messageUC)

	log.Printf("🔗 Nova conexão WebSocket: [%s] (ID: %s)", claims.Username, claims.UserID)

	go client.WritePump()
	go client.ReadPump()
}
