// модуль models содержит описание основных сущностей приложения
package models

import (
	"errors"
)

type BatchReq struct {
	CorrID string `json:"correlation_id"`
	URL    string `json:"original_url"`
}

// BatchResp используется для передачи ответа в формате json
type BatchResp struct {
	CorrID   string `json:"correlation_id"`
	ShortURL string `json:"short_url"`
}

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

var DeletedTokens []TokenUser

var ErrorAlreadyExist = errors.New("already exist")
var ErrLinkNotFound = errors.New("link is not found")
var ErrLinkDeleted = errors.New("link has been deleted")
