package usecase

import (
	"context"
	"fmt"

	"github.com/CiceroLucas/go-chat/internal/domain"
	"github.com/CiceroLucas/go-chat/internal/infrastructure/auth"
	"github.com/google/uuid"
)

type AuthUseCase struct {
	userRepo   domain.UserRepository
	hasher     *auth.Hasher
	jwtService *auth.JWTService
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  *domain.User `json:"user"`
}

func NewAuthUseCase(repo domain.UserRepository, hasher *auth.Hasher, jwt *auth.JWTService) *AuthUseCase {
	return &AuthUseCase{
		userRepo:   repo,
		hasher:     hasher,
		jwtService: jwt,
	}
}

func (uc *AuthUseCase) Register(ctx context.Context, username, email, password string) (*AuthResponse, error) {

	if username == "" || email == "" || password == "" {
		return nil, fmt.Errorf("%w: todos os campos são obrigatórios", domain.ErrInvalidInput)
	}
	if len(username) < 3 || len(username) > 50 {
		return nil, fmt.Errorf("%w: username deve ter entre 3 e 50 caracteres", domain.ErrInvalidInput)
	}
	if len(password) < 6 {
		return nil, fmt.Errorf("%w: senha deve ter pelo menos 6 caracteres", domain.ErrInvalidInput)
	}

	hash, err := uc.hasher.Hash(password)
	if err != nil {
		return nil, fmt.Errorf("erro ao gerar hash da senha: %w", err)
	}

	user := &domain.User{
		ID:           uuid.New().String(),
		Username:     username,
		Email:        email,
		PasswordHash: hash,
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	token, err := uc.jwtService.GenerateToken(user.ID, user.Username)
	if err != nil {
		return nil, fmt.Errorf("erro ao gerar token: %w", err)
	}

	return &AuthResponse{Token: token, User: user}, nil
}

func (uc *AuthUseCase) Login(ctx context.Context, username, password string) (*AuthResponse, error) {
	if username == "" || password == "" {
		return nil, fmt.Errorf("%w: username e senha são obrigatórios", domain.ErrInvalidInput)
	}

	user, err := uc.userRepo.FindByUsername(ctx, username)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	if err := uc.hasher.Compare(user.PasswordHash, password); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	token, err := uc.jwtService.GenerateToken(user.ID, user.Username)
	if err != nil {
		return nil, fmt.Errorf("erro ao gerar token: %w", err)
	}

	return &AuthResponse{Token: token, User: user}, nil
}

func (uc *AuthUseCase) ValidateToken(tokenString string) (*auth.Claims, error) {
	return uc.jwtService.ValidateToken(tokenString)
}
