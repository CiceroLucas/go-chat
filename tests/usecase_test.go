package tests

import (
	"context"
	"testing"
	"time"

	"github.com/CiceroLucas/go-chat/internal/domain"
	"github.com/CiceroLucas/go-chat/internal/infrastructure/auth"
	"github.com/CiceroLucas/go-chat/internal/usecase"
)

type MockUserRepository struct {
	users map[string]*domain.User
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{users: make(map[string]*domain.User)}
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	for _, u := range m.users {
		if u.Username == user.Username {
			return domain.ErrDuplicateUser
		}
		if u.Email == user.Email {
			return domain.ErrDuplicateUser
		}
	}
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	if user, ok := m.users[id]; ok {
		return user, nil
	}
	return nil, domain.ErrUserNotFound
}

func (m *MockUserRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	for _, user := range m.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

type MockMessageRepository struct {
	messages []*domain.Message
}

func NewMockMessageRepository() *MockMessageRepository {
	return &MockMessageRepository{messages: make([]*domain.Message, 0)}
}

func (m *MockMessageRepository) Save(ctx context.Context, msg *domain.Message) error {
	m.messages = append(m.messages, msg)
	return nil
}

func (m *MockMessageRepository) FindByRoomID(ctx context.Context, roomID string, limit, offset int) ([]*domain.Message, error) {
	var result []*domain.Message
	for _, msg := range m.messages {
		if msg.RoomID == roomID {
			result = append(result, msg)
		}
	}

	if offset >= len(result) {
		return nil, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}

func (m *MockMessageRepository) DeleteByRoomID(ctx context.Context, roomID string) error {
	filtered := make([]*domain.Message, 0)
	for _, msg := range m.messages {
		if msg.RoomID != roomID {
			filtered = append(filtered, msg)
		}
	}
	m.messages = filtered
	return nil
}

type MockRoomRepository struct {
	rooms map[string]*domain.Room
}

func NewMockRoomRepository() *MockRoomRepository {
	return &MockRoomRepository{rooms: make(map[string]*domain.Room)}
}

func (m *MockRoomRepository) Create(ctx context.Context, room *domain.Room) error {
	for _, r := range m.rooms {
		if r.Name == room.Name {
			return domain.ErrDuplicateRoom
		}
	}
	m.rooms[room.ID] = room
	return nil
}

func (m *MockRoomRepository) FindByID(ctx context.Context, id string) (*domain.Room, error) {
	if room, ok := m.rooms[id]; ok {
		return room, nil
	}
	return nil, domain.ErrRoomNotFound
}

func (m *MockRoomRepository) FindByName(ctx context.Context, name string) (*domain.Room, error) {
	for _, room := range m.rooms {
		if room.Name == name {
			return room, nil
		}
	}
	return nil, domain.ErrRoomNotFound
}

func (m *MockRoomRepository) ListAll(ctx context.Context) ([]*domain.Room, error) {
	var result []*domain.Room
	for _, room := range m.rooms {
		result = append(result, room)
	}
	return result, nil
}

func (m *MockRoomRepository) Delete(ctx context.Context, id string) error {
	if _, ok := m.rooms[id]; !ok {
		return domain.ErrRoomNotFound
	}
	delete(m.rooms, id)
	return nil
}

func TestAuthRegister(t *testing.T) {
	repo := NewMockUserRepository()
	hasher := auth.NewHasher(4)
	jwtSvc := auth.NewJWTService("test-secret", 1*time.Hour)
	uc := usecase.NewAuthUseCase(repo, hasher, jwtSvc)

	ctx := context.Background()

	t.Run("registro com sucesso", func(t *testing.T) {
		result, err := uc.Register(ctx, "testuser", "test@email.com", "123456")
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if result.Token == "" {
			t.Error("token não deveria ser vazio")
		}
		if result.User.Username != "testuser" {
			t.Errorf("username esperado %q, obteve %q", "testuser", result.User.Username)
		}
		if result.User.Email != "test@email.com" {
			t.Errorf("email esperado %q, obteve %q", "test@email.com", result.User.Email)
		}
	})

	t.Run("registro com username duplicado", func(t *testing.T) {
		_, err := uc.Register(ctx, "testuser", "other@email.com", "123456")
		if err == nil {
			t.Error("deveria retornar erro de duplicação")
		}
	})

	t.Run("registro com campos vazios", func(t *testing.T) {
		_, err := uc.Register(ctx, "", "a@b.com", "123456")
		if err == nil {
			t.Error("deveria retornar erro para username vazio")
		}
	})

	t.Run("registro com senha curta", func(t *testing.T) {
		_, err := uc.Register(ctx, "user2", "u2@email.com", "123")
		if err == nil {
			t.Error("deveria retornar erro para senha curta")
		}
	})
}

func TestAuthLogin(t *testing.T) {
	repo := NewMockUserRepository()
	hasher := auth.NewHasher(4)
	jwtSvc := auth.NewJWTService("test-secret", 1*time.Hour)
	uc := usecase.NewAuthUseCase(repo, hasher, jwtSvc)

	ctx := context.Background()

	_, err := uc.Register(ctx, "loginuser", "login@email.com", "senha123")
	if err != nil {
		t.Fatalf("erro ao registrar: %v", err)
	}

	t.Run("login com sucesso", func(t *testing.T) {
		result, err := uc.Login(ctx, "loginuser", "senha123")
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if result.Token == "" {
			t.Error("token não deveria ser vazio")
		}
		if result.User.Username != "loginuser" {
			t.Errorf("username esperado %q, obteve %q", "loginuser", result.User.Username)
		}
	})

	t.Run("login com senha errada", func(t *testing.T) {
		_, err := uc.Login(ctx, "loginuser", "senhaerrada")
		if err == nil {
			t.Error("deveria retornar erro para senha errada")
		}
	})

	t.Run("login com usuário inexistente", func(t *testing.T) {
		_, err := uc.Login(ctx, "inexistente", "senha123")
		if err == nil {
			t.Error("deveria retornar erro para usuário inexistente")
		}
	})

	t.Run("login com campos vazios", func(t *testing.T) {
		_, err := uc.Login(ctx, "", "")
		if err == nil {
			t.Error("deveria retornar erro para campos vazios")
		}
	})
}

func TestJWTService(t *testing.T) {
	svc := auth.NewJWTService("minha-chave-secreta", 1*time.Hour)

	t.Run("gerar e validar token", func(t *testing.T) {
		token, err := svc.GenerateToken("user-123", "testuser")
		if err != nil {
			t.Fatalf("erro ao gerar token: %v", err)
		}
		if token == "" {
			t.Fatal("token não deveria ser vazio")
		}

		claims, err := svc.ValidateToken(token)
		if err != nil {
			t.Fatalf("erro ao validar token: %v", err)
		}
		if claims.UserID != "user-123" {
			t.Errorf("user_id esperado %q, obteve %q", "user-123", claims.UserID)
		}
		if claims.Username != "testuser" {
			t.Errorf("username esperado %q, obteve %q", "testuser", claims.Username)
		}
	})

	t.Run("token inválido", func(t *testing.T) {
		_, err := svc.ValidateToken("token-invalido")
		if err == nil {
			t.Error("deveria retornar erro para token inválido")
		}
	})

	t.Run("token expirado", func(t *testing.T) {
		expiredSvc := auth.NewJWTService("chave", -1*time.Hour)
		token, _ := expiredSvc.GenerateToken("user-1", "user")
		_, err := expiredSvc.ValidateToken(token)
		if err == nil {
			t.Error("deveria retornar erro para token expirado")
		}
	})

	t.Run("chave secreta diferente", func(t *testing.T) {
		otherSvc := auth.NewJWTService("outra-chave", 1*time.Hour)
		token, _ := svc.GenerateToken("user-1", "user")
		_, err := otherSvc.ValidateToken(token)
		if err == nil {
			t.Error("deveria retornar erro para chave diferente")
		}
	})
}

func TestHasher(t *testing.T) {
	hasher := auth.NewHasher(4)

	t.Run("hash e compare com sucesso", func(t *testing.T) {
		hash, err := hasher.Hash("minha-senha")
		if err != nil {
			t.Fatalf("erro ao gerar hash: %v", err)
		}
		if hash == "" {
			t.Fatal("hash não deveria ser vazio")
		}
		if hash == "minha-senha" {
			t.Fatal("hash não deveria ser igual à senha original")
		}

		if err := hasher.Compare(hash, "minha-senha"); err != nil {
			t.Errorf("compare deveria ser bem-sucedido: %v", err)
		}
	})

	t.Run("compare com senha errada", func(t *testing.T) {
		hash, _ := hasher.Hash("senha-correta")
		if err := hasher.Compare(hash, "senha-errada"); err == nil {
			t.Error("compare deveria falhar para senha errada")
		}
	})
}

func TestRoomUseCase(t *testing.T) {
	repo := NewMockRoomRepository()
	msgRepo := NewMockMessageRepository()
	uc := usecase.NewRoomUseCase(repo, msgRepo)
	ctx := context.Background()

	t.Run("criar sala com sucesso", func(t *testing.T) {
		room, err := uc.CreateRoom(ctx, "Sala Teste", "user-1")
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if room.Name != "Sala Teste" {
			t.Errorf("nome esperado %q, obteve %q", "Sala Teste", room.Name)
		}
		if room.ID == "" {
			t.Error("ID não deveria ser vazio")
		}
	})

	t.Run("criar sala duplicada", func(t *testing.T) {
		_, err := uc.CreateRoom(ctx, "Sala Teste", "user-1")
		if err == nil {
			t.Error("deveria retornar erro para sala duplicada")
		}
	})

	t.Run("criar sala com nome vazio", func(t *testing.T) {
		_, err := uc.CreateRoom(ctx, "", "user-1")
		if err == nil {
			t.Error("deveria retornar erro para nome vazio")
		}
	})

	t.Run("listar salas", func(t *testing.T) {
		rooms, err := uc.ListRooms(ctx)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if len(rooms) == 0 {
			t.Error("deveria retornar pelo menos uma sala")
		}
	})

	t.Run("sala padrão", func(t *testing.T) {
		room, err := uc.EnsureDefaultRoom(ctx, "system")
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if room.Name != "Geral" {
			t.Errorf("nome esperado %q, obteve %q", "Geral", room.Name)
		}

		room2, err := uc.EnsureDefaultRoom(ctx, "system")
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if room2.ID != room.ID {
			t.Error("deveria retornar a mesma sala")
		}
	})

	t.Run("deletar sala com sucesso", func(t *testing.T) {
		room, _ := uc.CreateRoom(ctx, "Sala Deletavel", "user-1")
		err := uc.DeleteRoom(ctx, room.ID)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}

		rooms, _ := uc.ListRooms(ctx)
		for _, r := range rooms {
			if r.ID == room.ID {
				t.Error("sala deveria ter sido deletada")
			}
		}
	})

	t.Run("deletar sala Geral bloqueado", func(t *testing.T) {
		err := uc.DeleteRoom(ctx, room2ID(uc, ctx, t))
		if err == nil {
			t.Error("deveria retornar erro ao deletar sala Geral")
		}
	})

	t.Run("deletar sala inexistente", func(t *testing.T) {
		err := uc.DeleteRoom(ctx, "sala-inexistente")
		if err == nil {
			t.Error("deveria retornar erro para sala inexistente")
		}
	})
}

func room2ID(uc *usecase.RoomUseCase, ctx context.Context, t *testing.T) string {
	rooms, err := uc.ListRooms(ctx)
	if err != nil {
		t.Fatalf("erro ao listar salas: %v", err)
	}
	for _, r := range rooms {
		if r.Name == "Geral" {
			return r.ID
		}
	}
	t.Fatal("sala Geral não encontrada")
	return ""
}

func TestMessageUseCase(t *testing.T) {
	repo := NewMockMessageRepository()
	uc := usecase.NewMessageUseCase(repo)
	ctx := context.Background()

	t.Run("salvar mensagem com sucesso", func(t *testing.T) {
		msg := &domain.Message{
			ID:       "msg-1",
			RoomID:   "room-1",
			UserID:   "user-1",
			Username: "testuser",
			Content:  "Olá, mundo!",
			Type:     domain.MessageTypeText,
		}

		if err := uc.SaveMessage(ctx, msg); err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
	})

	t.Run("salvar mensagem sem conteúdo", func(t *testing.T) {
		msg := &domain.Message{
			RoomID: "room-1",
			UserID: "user-1",
		}
		if err := uc.SaveMessage(ctx, msg); err == nil {
			t.Error("deveria retornar erro para conteúdo vazio")
		}
	})

	t.Run("salvar mensagem sem sala", func(t *testing.T) {
		msg := &domain.Message{
			Content: "Teste",
			UserID:  "user-1",
		}
		if err := uc.SaveMessage(ctx, msg); err == nil {
			t.Error("deveria retornar erro para sala vazia")
		}
	})

	t.Run("buscar histórico", func(t *testing.T) {

		for i := 0; i < 5; i++ {
			uc.SaveMessage(ctx, &domain.Message{
				ID:       "msg-" + string(rune('A'+i)),
				RoomID:   "room-1",
				UserID:   "user-1",
				Username: "testuser",
				Content:  "Mensagem " + string(rune('A'+i)),
			})
		}

		messages, err := uc.GetHistory(ctx, "room-1", 10, 0)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if len(messages) == 0 {
			t.Error("deveria retornar mensagens")
		}
	})

	t.Run("buscar histórico sala vazia", func(t *testing.T) {
		_, err := uc.GetHistory(ctx, "", 10, 0)
		if err == nil {
			t.Error("deveria retornar erro para sala vazia")
		}
	})
}

func TestMessageType(t *testing.T) {
	if domain.MessageTypeText != "text" {
		t.Errorf("esperado %q, obteve %q", "text", domain.MessageTypeText)
	}
	if domain.MessageTypeSystem != "system" {
		t.Errorf("esperado %q, obteve %q", "system", domain.MessageTypeSystem)
	}
}
