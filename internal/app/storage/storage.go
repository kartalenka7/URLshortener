package storage

// слой хранилища

type Repository interface {
	AddLink(gToken string, longURL string) error
	GetLongURL(sToken string) (string, error)
}

type SavedLinks struct {
	LinksMap map[string]string
	gToken   string
}
