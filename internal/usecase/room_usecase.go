package usecase

import (
	"context"
	"fmt"

	"github.com/CiceroLucas/go-chat/internal/domain"
	"github.com/google/uuid"
)

type RoomUseCase struct {
	roomRepo    domain.RoomRepository
	messageRepo domain.MessageRepository
}

func NewRoomUseCase(roomRepo domain.RoomRepository, messageRepo domain.MessageRepository) *RoomUseCase {
	return &RoomUseCase{roomRepo: roomRepo, messageRepo: messageRepo}
}

func (uc *RoomUseCase) CreateRoom(ctx context.Context, name, createdBy string) (*domain.Room, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: nome da sala é obrigatório", domain.ErrInvalidInput)
	}
	if len(name) > 100 {
		return nil, fmt.Errorf("%w: nome da sala deve ter no máximo 100 caracteres", domain.ErrInvalidInput)
	}

	room := &domain.Room{
		ID:        uuid.New().String(),
		Name:      name,
		CreatedBy: createdBy,
	}

	if err := uc.roomRepo.Create(ctx, room); err != nil {
		return nil, err
	}
	return room, nil
}

func (uc *RoomUseCase) ListRooms(ctx context.Context) ([]*domain.Room, error) {
	return uc.roomRepo.ListAll(ctx)
}

func (uc *RoomUseCase) GetRoom(ctx context.Context, id string) (*domain.Room, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: ID da sala é obrigatório", domain.ErrInvalidInput)
	}
	return uc.roomRepo.FindByID(ctx, id)
}

func (uc *RoomUseCase) DeleteRoom(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("%w: ID da sala é obrigatório", domain.ErrInvalidInput)
	}

	room, err := uc.roomRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if room.Name == "Geral" {
		return fmt.Errorf("%w: a sala padrão 'Geral' não pode ser deletada", domain.ErrInvalidInput)
	}

	if uc.messageRepo != nil {
		if err := uc.messageRepo.DeleteByRoomID(ctx, id); err != nil {
			return fmt.Errorf("erro ao deletar mensagens da sala: %w", err)
		}
	}

	return uc.roomRepo.Delete(ctx, id)
}

func (uc *RoomUseCase) EnsureDefaultRoom(ctx context.Context, systemUserID string) (*domain.Room, error) {
	room, err := uc.roomRepo.FindByName(ctx, "Geral")
	if err == nil {
		return room, nil
	}

	return uc.CreateRoom(ctx, "Geral", systemUserID)
}
