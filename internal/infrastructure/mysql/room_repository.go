package mysql

import (
	"context"
	"database/sql"
	"time"

	"github.com/CiceroLucas/go-chat/internal/domain"
	"github.com/google/uuid"
)

type RoomRepository struct {
	db *sql.DB
}

func NewRoomRepository(db *sql.DB) *RoomRepository {
	return &RoomRepository{db: db}
}

func (r *RoomRepository) Create(ctx context.Context, room *domain.Room) error {
	if room.ID == "" {
		room.ID = uuid.New().String()
	}
	if room.CreatedAt.IsZero() {
		room.CreatedAt = time.Now()
	}

	query := `INSERT INTO rooms (id, name, created_by, created_at) VALUES (?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, room.ID, room.Name, room.CreatedBy, room.CreatedAt)
	if err != nil {
		if isDuplicateError(err) {
			return domain.ErrDuplicateRoom
		}
		return err
	}
	return nil
}

func (r *RoomRepository) FindByID(ctx context.Context, id string) (*domain.Room, error) {
	room := &domain.Room{}
	query := `SELECT id, name, created_by, created_at FROM rooms WHERE id = ?`
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&room.ID, &room.Name, &room.CreatedBy, &room.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrRoomNotFound
	}
	if err != nil {
		return nil, err
	}
	return room, nil
}

func (r *RoomRepository) FindByName(ctx context.Context, name string) (*domain.Room, error) {
	room := &domain.Room{}
	query := `SELECT id, name, created_by, created_at FROM rooms WHERE name = ?`
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&room.ID, &room.Name, &room.CreatedBy, &room.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrRoomNotFound
	}
	if err != nil {
		return nil, err
	}
	return room, nil
}

func (r *RoomRepository) ListAll(ctx context.Context) ([]*domain.Room, error) {
	query := `SELECT id, name, created_by, created_at FROM rooms ORDER BY created_at ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []*domain.Room
	for rows.Next() {
		room := &domain.Room{}
		if err := rows.Scan(&room.ID, &room.Name, &room.CreatedBy, &room.CreatedAt); err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}
	return rooms, rows.Err()
}

func (r *RoomRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM rooms WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrRoomNotFound
	}
	return nil
}
