package config

import (
	"database/sql"
	"encoding/json"
	"errors"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type StreamConfig struct {
	Title         string `json:"title"`
	AccentColor   string `json:"accentColor"`
	BannerURL     string `json:"bannerUrl"`
	RTMPStreamKey string `json:"rtmpStreamKey"`
	PusherAppID   string `json:"pusherAppId"`
	PusherKey     string `json:"pusherKey"`
	PusherSecret  string `json:"pusherSecret"`
	PusherCluster string `json:"pusherCluster"`
}

type Store struct {
	mu sync.RWMutex
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	stmt := `
	CREATE TABLE IF NOT EXISTS stream_config (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		payload TEXT NOT NULL
	);`
	if _, err := db.Exec(stmt); err != nil {
		return nil, err
	}

	store := &Store{db: db}
	_, err = store.db.Exec(`INSERT OR IGNORE INTO stream_config (id, payload) VALUES (1, ?)`, defaultPayload())
	if err != nil {
		return nil, err
	}

	return store, nil
}

func defaultPayload() string {
	cfg := StreamConfig{
		Title:       "SovereignStream Live",
		AccentColor: "#7c3aed",
	}
	data, _ := json.Marshal(cfg)
	return string(data)
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Load() (StreamConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var payload string
	err := s.db.QueryRow(`SELECT payload FROM stream_config WHERE id = 1`).Scan(&payload)
	if err != nil {
		return StreamConfig{}, err
	}

	var cfg StreamConfig
	if err := json.Unmarshal([]byte(payload), &cfg); err != nil {
		return StreamConfig{}, err
	}
	return cfg, nil
}

func (s *Store) Save(cfg StreamConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	res, err := s.db.Exec(`UPDATE stream_config SET payload = ? WHERE id = 1`, string(data))
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("stream config row missing")
	}
	return nil
}
