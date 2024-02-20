package storage

import (
	"context"
	"errors"
	"os"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type PGURLStorage struct {
	DB     *pgxpool.Pool
	logger *zap.SugaredLogger
}

func tmpDbInit(ctx context.Context, db *pgxpool.Pool, s *zap.SugaredLogger) {
	_, err := db.Exec(ctx, `CREATE TABLE IF NOT EXISTS urls (
			uuid BIGSERIAL PRIMARY KEY,
			short_url TEXT NOT NULL,
			original_url TEXT NOT NULL);
			CREATE UNIQUE INDEX uniq_urls ON urls (original_url) NULLS NOT DISTINCT
			`)
	if err != nil {
		s.Infoln("Unable to init db:", err)
	}
}

func PGNewURLStorage(ctx context.Context, databaseDSN string, s *zap.SugaredLogger) *PGURLStorage {
	if len(databaseDSN) > 0 {
		poolConfig, err := pgxpool.ParseConfig(databaseDSN)
		if err != nil {
			s.Errorln("Unable to parse `databaseDSN`:", err)
			return nil
		}
		db, err := pgxpool.NewWithConfig(ctx, poolConfig)
		if err != nil {
			s.Errorln("Unable to create connection pool:", err)
			return nil
		}
		tmpDbInit(ctx, db, s)
		return &PGURLStorage{DB: db, logger: s}
	}
	return &PGURLStorage{DB: nil, logger: s}
}

func (p *PGURLStorage) IsReady() bool {
	return p.DB != nil
}

func (p *PGURLStorage) Store(ctx context.Context, key string, origURL string) (oKey string, dup bool) {
	if !p.IsReady() {
		p.logger.Errorln(os.Stderr, "error: no connection to database")
		return "", false
	}
	if _, err := p.DB.Exec(ctx, `insert into urls(short_url, original_url) values ($1,$2)`, key, origURL); err != nil {
		p.logger.Errorln(err.Error())
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
				err := p.DB.QueryRow(ctx, "select short_url from urls where original_url=$1", origURL).Scan(&oKey)
				if err != nil {
					p.logger.Errorln(err.Error())
					return "", false
				}
				return oKey, true
			}
		}
	}
	return "", false
}

func (p *PGURLStorage) Load(ctx context.Context, key string) (string, bool) {
	var URL string
	if !p.IsReady() {
		p.logger.Errorln("error: no connection to database")
		return "", false
	}
	err := p.DB.QueryRow(ctx, "select original_url from urls where short_url=$1", key).Scan(&URL)
	if err != nil {
		p.logger.Errorln(err.Error())
		return "", false
	}
	return URL, true
}
