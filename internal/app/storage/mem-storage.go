package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"os"

	"go.uber.org/zap"
)

type URLStorage struct {
	db   map[string]*URLRecord
	file *os.File
	// writer
	writer *bufio.Writer
	// reader
	scanner *bufio.Scanner
	logger  *zap.SugaredLogger
}

type URLEvent struct {
	ID uint `json:"uuid"`
	URLRecord
}

func NewMemStorage(ctx context.Context, path string, logger *zap.SugaredLogger) *URLStorage {
	mem := NewURLStorage(path, logger)
	mem.LoadFromFile(ctx)
	return mem
}

func NewURLStorage(filename string, s *zap.SugaredLogger) *URLStorage {
	if len(filename) > 0 {
		file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			s.Errorln(err.Error())
			return nil
		}
		URLs := URLStorage{
			db:      make(map[string]*URLRecord),
			file:    file,
			writer:  bufio.NewWriter(file),
			scanner: bufio.NewScanner(file),
			logger:  s,
		}
		return &URLs
	}
	// just in-memory storage
	return &URLStorage{db: make(map[string]*URLRecord), file: nil, writer: nil, scanner: nil, logger: s}
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
		rec := &event.URLRecord
		u.db[rec.ShortKey] = rec
	}
}

func (u *URLStorage) isDuplicate(ctx context.Context, Rec *URLRecord) (URLRecord, bool) {
	for _, oRec := range u.db {
		if oRec.OriginalURL == Rec.OriginalURL {
			return *oRec, true
		}
	}
	return URLRecord{}, false
}

func (u *URLStorage) StoreBatch(ctx context.Context, URLs map[string]*URLRecord) (map[string]*URLRecord, bool, error) {
	origRecs := make(map[string]*URLRecord)
	for id, Rec := range URLs {
		origRec, exist, err := u.Store(ctx, Rec)
		if exist {
			origRecs[id] = origRec
			return origRecs, true, err
		}
	}
	return origRecs, false, nil
}

func (u *URLStorage) Store(ctx context.Context, rec *URLRecord) (*URLRecord, bool, error) {
	/*
		// return original key
		oRec, dup := u.isDuplicate(ctx, rec)
		if dup {
			return &oRec, dup, nil
		}
	*/
	// no persistent storage available
	if u.file == nil || u.writer == nil || u.scanner == nil {
		u.db[rec.ShortKey] = rec
	} else {
		u.db[rec.ShortKey] = rec
		event := URLEvent{ID: uint(len(u.db)), URLRecord: *rec}
		data, err := json.Marshal(&event)
		if err != nil {
			u.logger.Errorln(err.Error())
			return nil, false, err
		}
		if _, err := u.writer.Write(data); err != nil {
			u.logger.Errorln(err.Error())
			return nil, false, err
		}
		if err := u.writer.WriteByte('\n'); err != nil {
			u.logger.Errorln(err.Error())
			return nil, false, err
		}
		if err := u.writer.Flush(); err != nil {
			u.logger.Errorln(os.Stderr, err.Error())
			return nil, false, err
		}
	}
	return nil, false, nil
}

func (u *URLStorage) Load(ctx context.Context, key string) (URLRecord, bool, error) {
	URLrec, ok := u.db[key]
	if !ok {
		return URLRecord{}, ok, nil
	}
	return *URLrec, ok, nil
}

func (u *URLStorage) List(ctx context.Context, ID string) ([]*URLRecord, error) {
	var res []*URLRecord
	for _, URLrec := range u.db {
		if URLrec.UserID == ID {
			res = append(res, URLrec)
		}
	}
	return res, nil
}

func (u *URLStorage) Delete(ctx context.Context, recs []*URLRecord) error {
	// not implemented
	return nil
}
