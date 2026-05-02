package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/CiceroLucas/go-chat/internal/domain"
	"github.com/CiceroLucas/go-chat/internal/usecase"
	"github.com/gorilla/mux"
)

type Handler struct {
	authUC    *usecase.AuthUseCase
	roomUC    *usecase.RoomUseCase
	messageUC *usecase.MessageUseCase
}

func NewHandler(authUC *usecase.AuthUseCase, roomUC *usecase.RoomUseCase, messageUC *usecase.MessageUseCase) *Handler {
	return &Handler{
		authUC:    authUC,
		roomUC:    roomUC,
		messageUC: messageUC,
	}
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Dados inválidos")
		return
	}

	result, err := h.authUC.Register(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case isError(err, domain.ErrDuplicateUser):
			status = http.StatusConflict
		case isError(err, domain.ErrInvalidInput):
			status = http.StatusBadRequest
		}
		respondError(w, status, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, result)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Dados inválidos")
		return
	}

	result, err := h.authUC.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case isError(err, domain.ErrInvalidCredentials):
			status = http.StatusUnauthorized
		case isError(err, domain.ErrInvalidInput):
			status = http.StatusBadRequest
		}
		respondError(w, status, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

type CreateRoomRequest struct {
	Name string `json:"name"`
}

func (h *Handler) ListRooms(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.roomUC.ListRooms(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Erro ao listar salas")
		return
	}

	if rooms == nil {
		rooms = []*domain.Room{}
	}

	respondJSON(w, http.StatusOK, rooms)
}

func (h *Handler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	var req CreateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Dados inválidos")
		return
	}

	userID, _ := r.Context().Value(ContextKeyUserID).(string)

	room, err := h.roomUC.CreateRoom(r.Context(), req.Name, userID)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case isError(err, domain.ErrDuplicateRoom):
			status = http.StatusConflict
		case isError(err, domain.ErrInvalidInput):
			status = http.StatusBadRequest
		}
		respondError(w, status, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, room)
}

func (h *Handler) GetRoomMessages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["id"]

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = 50
	}

	messages, err := h.messageUC.GetHistory(r.Context(), roomID, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Erro ao buscar mensagens")
		return
	}

	if messages == nil {
		messages = []*domain.Message{}
	}

	respondJSON(w, http.StatusOK, messages)
}

func (h *Handler) DeleteRoom(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["id"]

	err := h.roomUC.DeleteRoom(r.Context(), roomID)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case isError(err, domain.ErrRoomNotFound):
			status = http.StatusNotFound
		case isError(err, domain.ErrInvalidInput):
			status = http.StatusBadRequest
		}
		respondError(w, status, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Sala deletada com sucesso",
	})
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "go-chat",
	})
}

func isError(err, target error) bool {
	if err == nil || target == nil {
		return false
	}
	return err.Error() == target.Error() || contains(err.Error(), target.Error())
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
