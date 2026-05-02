package usecase

import (
	"context"
	"fmt"

	"github.com/CiceroLucas/go-chat/internal/domain"
)

type MessageUseCase struct {
	messageRepo domain.MessageRepository
}

func NewMessageUseCase(repo domain.MessageRepository) *MessageUseCase {
	return &MessageUseCase{messageRepo: repo}
}

func (uc *MessageUseCase) SaveMessage(ctx context.Context, msg *domain.Message) error {
	if msg.Content == "" {
		return fmt.Errorf("%w: conteúdo da mensagem é obrigatório", domain.ErrInvalidInput)
	}
	if msg.RoomID == "" {
		return fmt.Errorf("%w: ID da sala é obrigatório", domain.ErrInvalidInput)
	}
	if len(msg.Content) > 5000 {
		return fmt.Errorf("%w: mensagem deve ter no máximo 5000 caracteres", domain.ErrInvalidInput)
	}
	return uc.messageRepo.Save(ctx, msg)
}

func (uc *MessageUseCase) GetHistory(ctx context.Context, roomID string, limit, offset int) ([]*domain.Message, error) {
	if roomID == "" {
		return nil, fmt.Errorf("%w: ID da sala é obrigatório", domain.ErrInvalidInput)
	}
	if limit <= 0 {
		limit = 50
	}
	return uc.messageRepo.FindByRoomID(ctx, roomID, limit, offset)
}
