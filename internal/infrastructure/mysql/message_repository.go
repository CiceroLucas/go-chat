package mysql

import (
	"context"
	"database/sql"
	"time"

	"github.com/CiceroLucas/go-chat/internal/domain"
	"github.com/google/uuid"
)

type MessageRepository struct {
	db *sql.DB
}

func NewMessageRepository(db *sql.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Save(ctx context.Context, msg *domain.Message) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	if msg.Type == "" {
		msg.Type = domain.MessageTypeText
	}

	query := `INSERT INTO messages (id, room_id, user_id, username, content, msg_type, created_at) 
	          VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		msg.ID, msg.RoomID, msg.UserID, msg.Username, msg.Content, msg.Type, msg.CreatedAt,
	)
	return err
}

func (r *MessageRepository) FindByRoomID(ctx context.Context, roomID string, limit, offset int) ([]*domain.Message, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	query := `SELECT id, room_id, user_id, username, content, msg_type, created_at 
	          FROM messages 
	          WHERE room_id = ? 
	          ORDER BY created_at ASC 
	          LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query, roomID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*domain.Message
	for rows.Next() {
		msg := &domain.Message{}
		if err := rows.Scan(
			&msg.ID, &msg.RoomID, &msg.UserID, &msg.Username,
			&msg.Content, &msg.Type, &msg.CreatedAt,
		); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func (r *MessageRepository) DeleteByRoomID(ctx context.Context, roomID string) error {
	query := `DELETE FROM messages WHERE room_id = ?`
	_, err := r.db.ExecContext(ctx, query, roomID)
	return err
}
