// модуль models содержит описание основных сущностей приложения
package models

import (
	"errors"
)

// в BatchReq передаются данные для batch запросов
type BatchReq struct {
	CorrID string `json:"correlation_id"`
	URL    string `json:"original_url"`
}

// BatchResp используется для передачи ответа в формате json
type BatchResp struct {
	CorrID   string `json:"correlation_id"`
	ShortURL string `json:"short_url"`
}

// в структуру LinksData парсим данные из sql запросов
type LinksData struct {
	ShortURL string `json:"short"`
	LongURL  string `json:"long"`
	User     string `json:"user"`
}

// Структура TokenUser, куда будем накапливать токены URLов, подлежащиe удалению
type TokenUser struct {
	Token string
	User  string
}

// Структура struct для общего числа пользователей и скоращенных URL
type Stats struct {
	URLs  int `json:"urls"`
	Users int `json:"users"`
}

// Сообщения об ошибках
var (
	ErrorAlreadyExist   = errors.New("already exist")
	ErrLinkNotFound     = errors.New("link is not found")
	ErrLinkDeleted      = errors.New("link has been deleted")
	ErrNotTrustedSubnet = errors.New("not trusted subnet")
)
