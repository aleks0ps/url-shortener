package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type URLStorage struct {
	db   map[string]string
	file *os.File
	// writer
	writer *bufio.Writer
	// reader
	scanner *bufio.Scanner
}

type URLEvent struct {
	ID  uint   `json:"uuid"`
	Key string `json:"short_url"`
	URL string `json:"original_url"`
}

func NewURLStorage(filename string) *URLStorage {
	if len(filename) > 0 {
		file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Fprint(os.Stderr, err.Error())
			return nil
		}
		URLs := URLStorage{
			db:      make(map[string]string),
			file:    file,
			writer:  bufio.NewWriter(file),
			scanner: bufio.NewScanner(file),
		}
		return &URLs
	}
	// just in-memory storage
	return &URLStorage{db: make(map[string]string), file: nil, writer: nil, scanner: nil}
}

func (u *URLStorage) LoadFromFile() {
	// no persistent storage available
	if u.file == nil || u.writer == nil || u.scanner == nil {
		return
	}
	for {
		if !u.scanner.Scan() {
			if u.scanner.Err() == nil {
				break
			} else {
				fmt.Fprint(os.Stderr, u.scanner.Err().Error())
				return
			}
		}
		data := u.scanner.Bytes()
		event := URLEvent{}
		err := json.Unmarshal(data, &event)
		if err != nil {
			fmt.Fprint(os.Stderr, err.Error())
			return
		}
		// Store URLs to runtime struct
		u.db[event.Key] = event.URL
	}
}

func (u *URLStorage) StoreURL(key string, origURL string) {
	// no persistent storage available
	if u.file == nil || u.writer == nil || u.scanner == nil {
		u.db[key] = origURL
	} else {
		u.db[key] = origURL
		event := URLEvent{ID: uint(len(u.db)), Key: key, URL: origURL}
		data, err := json.Marshal(&event)
		if err != nil {
			fmt.Fprint(os.Stderr, err.Error())
			return
		}
		if _, err := u.writer.Write(data); err != nil {
			fmt.Fprint(os.Stderr, err.Error())
			return
		}
		if err := u.writer.WriteByte('\n'); err != nil {
			fmt.Fprint(os.Stderr, err.Error())
			return
		}
		if err := u.writer.Flush(); err != nil {
			fmt.Fprint(os.Stderr, err.Error())
			return
		}
	}
}

func (u *URLStorage) GetURL(key string) (string, bool) {
	URL, ok := u.db[key]
	return URL, ok
}
