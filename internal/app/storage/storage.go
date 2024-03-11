package storage

import "context"

type URLRecord struct {
	ShortKey    string `json:"short_key" db:"short_key"`
	OriginalURL string `json:"original_url" db:"original_url"`
	UserID      string `json:"user_id" db:"user_id"`
	DeletedFlag bool   `db:"is_deleted"`
}

type Storager interface {
	Load(ctx context.Context, key string) (origRec URLRecord, ok bool, err error)
	StoreBatch(ctx context.Context, URLs map[string]*URLRecord) (map[string]*URLRecord, bool, error)
	List(ctx context.Context, ID string) ([]*URLRecord, error)
	Store(ctx context.Context, rec *URLRecord) (origRec *URLRecord, exist bool, err error)
	Delete(ctx context.Context, recs []*URLRecord) (err error)
}
