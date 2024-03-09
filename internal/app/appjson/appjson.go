package appjson

type URL struct {
	URL string `json:"url"`
}

type Result struct {
	Result string `json:"result"`
}

type CorrelationID struct {
	CorrelationID string `json:"correlation_id"`
}

type ShortURL struct {
	ShortURL string `json:"short_url"`
}

type OriginalURL struct {
	OriginalURL string `json:"original_url"`
}

type BatchOriginalURL struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type BatchShortURL struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

type URLRecord struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}
