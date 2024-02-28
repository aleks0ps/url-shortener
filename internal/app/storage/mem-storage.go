package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"os"

	"go.uber.org/zap"
)

type URLStorage struct {
	db   map[string]string
	file *os.File
	// writer
	writer *bufio.Writer
	// reader
	scanner *bufio.Scanner
	logger  *zap.SugaredLogger
}

type URLEvent struct {
	ID  uint   `json:"uuid"`
	Key string `json:"short_url"`
	URL string `json:"original_url"`
}

func NewURLStorage(filename string, s *zap.SugaredLogger) *URLStorage {
	if len(filename) > 0 {
		file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			s.Errorln(err.Error())
			return nil
		}
		URLs := URLStorage{
			db:      make(map[string]string),
			file:    file,
			writer:  bufio.NewWriter(file),
			scanner: bufio.NewScanner(file),
			logger:  s,
		}
		return &URLs
	}
	// just in-memory storage
	return &URLStorage{db: make(map[string]string), file: nil, writer: nil, scanner: nil, logger: s}
}

func (u *URLStorage) LoadFromFile(ctx context.Context) {
	// no persistent storage available
	if u.file == nil || u.writer == nil || u.scanner == nil {
		return
	}
	for {
		if !u.scanner.Scan() {
			if u.scanner.Err() == nil {
				break
			} else {
				u.logger.Errorln(u.scanner.Err().Error())
				return
			}
		}
		data := u.scanner.Bytes()
		event := URLEvent{}
		err := json.Unmarshal(data, &event)
		if err != nil {
			u.logger.Errorln(err.Error())
			return
		}
		// Store URLs to runtime struct
		u.db[event.Key] = event.URL
	}
}

func (u *URLStorage) isDuplicate(ctx context.Context, URL string) (string, bool) {
	for key, oURL := range u.db {
		if oURL == URL {
			return key, true
		}
	}
	return "", false
}

func (u *URLStorage) StoreBatch(ctx context.Context, URLs map[string]*URLRecord) (map[string]*URLRecord, bool, error) {
	origURLs := make(map[string]*URLRecord)
	for id, URL := range URLs {
		origKey, exist, err := u.Store(ctx, URL.ShortKey, URL.OriginalURL)
		if exist {
			origURLs[id] = &URLRecord{ShortKey: origKey, OriginalURL: URL.OriginalURL}
			return origURLs, true, err
		}
	}
	return origURLs, false, nil

}

func (u *URLStorage) StoreR(ctx context.Context, rec *URLRecord) (*URLRecord, bool, error) {
	var res URLRecord
	origKey, exist, err := u.Store(ctx, rec.ShortKey, rec.OriginalURL)
	if err != nil {
		return nil, exist, err
	}
	res.ShortKey = origKey
	res.OriginalURL = rec.OriginalURL
	return &res, exist, nil
}

func (u *URLStorage) Store(ctx context.Context, key string, URL string) (string, bool, error) {
	// return original key
	oKey, dup := u.isDuplicate(ctx, URL)
	if dup {
		return oKey, dup, nil
	}
	// no persistent storage available
	if u.file == nil || u.writer == nil || u.scanner == nil {
		u.db[key] = URL
	} else {
		u.db[key] = URL
		event := URLEvent{ID: uint(len(u.db)), Key: key, URL: URL}
		data, err := json.Marshal(&event)
		if err != nil {
			u.logger.Errorln(err.Error())
			return "", false, err
		}
		if _, err := u.writer.Write(data); err != nil {
			u.logger.Errorln(err.Error())
			return "", false, err
		}
		if err := u.writer.WriteByte('\n'); err != nil {
			u.logger.Errorln(err.Error())
			return "", false, err
		}
		if err := u.writer.Flush(); err != nil {
			u.logger.Errorln(os.Stderr, err.Error())
			return "", false, err
		}
	}
	return "", false, nil
}

func (u *URLStorage) Load(ctx context.Context, key string) (string, bool, error) {
	URL, ok := u.db[key]
	return URL, ok, nil
}

func (u *URLStorage) List(ctx context.Context, ID string) ([]*URLRecord, error) {
	var res []*URLRecord
	for key, URL := range u.db {
		res = append(res, &URLRecord{ShortKey: key, OriginalURL: URL})
	}
	return res, nil
}
