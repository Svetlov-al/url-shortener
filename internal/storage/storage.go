package storage

import "errors"

var (
	ErrURLNotFound      = errors.New("Ссылка не найдена")
	ErrURLAlreadyExists = errors.New("Ссылка уже существует")
)
