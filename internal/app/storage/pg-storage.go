package storage

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PGURLStorage struct {
	DB *pgxpool.Pool
}

func tmpCreateTable(ctx context.Context, db *pgxpool.Pool) {
	_, err := db.Exec(ctx, `CREATE TABLE IF NOT EXISTS urls (
			uuid SERIAL PRIMARY KEY,
			short_url TEXT NOT NULL,
			original_url TEXT NOT NULL)`)
	if err != nil {
		log.Fatalln("Unable to create table in a database:", err)
	}
}

func tmpCreateIndex(ctx context.Context, db *pgxpool.Pool) {
	_, err := db.Exec(ctx, `CREATE UNIQUE INDEX uniq_urls ON urls (original_url) NULLS NOT DISTINCT`)
	if err != nil {
		log.Println("Unable to create index for original_url in urls table: ", err)
	}
}

func PGNewURLStorage(ctx context.Context, databaseDSN string) *PGURLStorage {
	if len(databaseDSN) > 0 {
		poolConfig, err := pgxpool.ParseConfig(databaseDSN)
		if err != nil {
			log.Fatalln("Unable to parse `databaseDSN`:", err)
			return nil
		}
		db, err := pgxpool.NewWithConfig(ctx, poolConfig)
		if err != nil {
			log.Fatalln("Unable to create connection pool:", err)
			return nil
		}
		tmpCreateTable(ctx, db)
		tmpCreateIndex(ctx, db)
		return &PGURLStorage{DB: db}
	}
	return &PGURLStorage{DB: nil}
}

func (p *PGURLStorage) IsReady() bool {
	return p.DB != nil
}

func (p *PGURLStorage) StoreURL(ctx context.Context, key string, origURL string) (uniqKey string, urlExist bool) {
	if p.DB == nil {
		fmt.Fprintln(os.Stderr, "error: no connection to database")
		return "", false
	}
	if _, err := p.DB.Exec(ctx, `insert into urls(short_url, original_url) values ($1,$2)`, key, origURL); err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		if pgerrcode.IsIntegrityConstraintViolation(err.Error()) {
			err := p.DB.QueryRow(ctx, "select short_url from urls where original_url=$1", origURL).Scan(&uniqKey)
			if err != nil {
				fmt.Fprint(os.Stderr, err.Error())
				return "", false
			}
			return uniqKey, true
		}
	}
	return "", false
}

func (p *PGURLStorage) GetURL(ctx context.Context, key string) (string, bool) {
	var originalURL string
	if p.DB == nil {
		fmt.Fprintln(os.Stderr, "error: no connection to database")
		return "", false
	}
	err := p.DB.QueryRow(ctx, "select original_url from urls where short_url=$1", key).Scan(&originalURL)
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		return "", false
	}
	return originalURL, true
}
