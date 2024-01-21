package storage

type URLStorage struct {
	db map[string]string
}

func NewURLStorage() *URLStorage {
	URLs := URLStorage{
		db: make(map[string]string),
	}
	return &URLs
}

func (u *URLStorage) StoreURL(key string, origURL string) {
	u.db[key] = origURL
}

func (u *URLStorage) GetURL(key string) (string, bool) {
	URL, ok := u.db[key]
	return URL, ok
}
