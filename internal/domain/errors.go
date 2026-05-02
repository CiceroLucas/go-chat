package domain

import "errors"

var (
	ErrUserNotFound       = errors.New("usuário não encontrado")
	ErrRoomNotFound       = errors.New("sala não encontrada")
	ErrDuplicateUser      = errors.New("nome de usuário ou email já existe")
	ErrDuplicateRoom      = errors.New("nome da sala já existe")
	ErrInvalidCredentials = errors.New("credenciais inválidas")
	ErrUnauthorized       = errors.New("não autorizado")
	ErrRateLimited        = errors.New("muitas requisições, tente novamente mais tarde")
	ErrInvalidInput       = errors.New("dados de entrada inválidos")
)
