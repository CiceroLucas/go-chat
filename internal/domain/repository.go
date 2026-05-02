package domain

import "context"

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id string) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
}

type RoomRepository interface {
	Create(ctx context.Context, room *Room) error
	FindByID(ctx context.Context, id string) (*Room, error)
	FindByName(ctx context.Context, name string) (*Room, error)
	ListAll(ctx context.Context) ([]*Room, error)
	Delete(ctx context.Context, id string) error
}

type MessageRepository interface {
	Save(ctx context.Context, message *Message) error
	FindByRoomID(ctx context.Context, roomID string, limit, offset int) ([]*Message, error)
	DeleteByRoomID(ctx context.Context, roomID string) error
}
