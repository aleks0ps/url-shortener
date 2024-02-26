package storage

import (
	"context"
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type PGURLStorage struct {
	DB     *pgxpool.Pool
	logger *zap.SugaredLogger
}

func tmpDBInit(ctx context.Context, db *pgxpool.Pool, s *zap.SugaredLogger) error {
	_, err := db.Exec(ctx, `CREATE TABLE IF NOT EXISTS urls (
			uuid BIGSERIAL PRIMARY KEY,
			short_url TEXT NOT NULL,
			original_url TEXT NOT NULL);
			CREATE UNIQUE INDEX uniq_urls ON urls (original_url) NULLS NOT DISTINCT
			`)
	if err != nil {
		s.Errorln("Unable to init db:", err)
		return err
	}
	return nil
}

func PGNewURLStorage(ctx context.Context, databaseDSN string, s *zap.SugaredLogger) (*PGURLStorage, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseDSN)
	if err != nil {
		s.Errorln("Unable to parse `databaseDSN`:", err)
		return nil, err
	}
	db, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		s.Errorln("Unable to create connection pool:", err)
		return nil, err
	}
	err = tmpDBInit(ctx, db, s)
	if err != nil {
		s.Errorln(err)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// not DuplicateTable = "42P07"
			if !pgerrcode.IsSyntaxErrororAccessRuleViolation(pgErr.Code) {
				return nil, err
			}
		}
	}
	return &PGURLStorage{DB: db, logger: s}, nil
}

func (p *PGURLStorage) StoreBatch(ctx context.Context, URLs map[string]*URLRecord) (map[string]*URLRecord, bool, error) {
	origURLs := make(map[string]*URLRecord)
	tx, err := p.DB.Begin(ctx)
	if err != nil {
		p.logger.Errorln(err.Error())
		return origURLs, false, err
	}
	defer tx.Rollback(ctx)
	for id, URLrec := range URLs {
		_, err := tx.Exec(ctx, `insert into urls(short_url, original_url) values ($1,$2)`, URLrec.ShortKey, URLrec.OriginalURL)
		if err != nil {
			p.logger.Errorln(err.Error())
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				if pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
					var origKey string
					err := p.DB.QueryRow(ctx, "select short_url from urls where original_url=$1", URLrec.OriginalURL).Scan(&origKey)
					if err != nil {
						p.logger.Errorln(err.Error())
						return origURLs, true, err
					}
					// save original short key
					origURLs[id] = &URLRecord{ShortKey: origKey, OriginalURL: URLrec.OriginalURL}
					return origURLs, true, err
				}
			}
		}
	}
	if err = tx.Commit(ctx); err != nil {
		p.logger.Errorln(err.Error())
		return origURLs, false, err
	}
	return origURLs, false, nil
}

func (p *PGURLStorage) Store(ctx context.Context, key string, origURL string) (origKey string, exist bool, e error) {
	if _, err := p.DB.Exec(ctx, `insert into urls(short_url, original_url) values ($1,$2)`, key, origURL); err != nil {
		p.logger.Errorln(err.Error())
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
				err := p.DB.QueryRow(ctx, "select short_url from urls where original_url=$1", origURL).Scan(&origKey)
				if err != nil {
					p.logger.Errorln(err.Error())
					return "", false, err
				}
				return origKey, true, err
			}
		}
	}
	return "", false, nil
}

func (p *PGURLStorage) Load(ctx context.Context, key string) (string, bool, error) {
	var URL string
	err := p.DB.QueryRow(ctx, "select original_url from urls where short_url=$1", key).Scan(&URL)
	if err != nil {
		p.logger.Errorln(err.Error())
		return "", false, err
	}
	return URL, true, nil
}
