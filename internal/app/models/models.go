package models

type BatchReq struct {
	CorrID string `json:"correlation_id"`
	URL    string `json:"original_url"`
	Cookie string
}

type BatchResp struct {
	CorrID   string `json:"correlation_id"`
	ShortURL string `json:"short_url"`
}

type LinksData struct {
	ShortURL string `json:"short"`
	LongURL  string `json:"long"`
	User     string `json:"user"`
}
