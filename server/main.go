package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"sovereignstream/server/internal/config"
	"sovereignstream/server/internal/dashboard"
	"sovereignstream/server/internal/rtmp"
	"sovereignstream/server/internal/signaling"
	"sovereignstream/server/internal/transcoder"
	"sovereignstream/server/internal/tunnel"
)

type liveState struct {
	mu        sync.RWMutex
	online    bool
	streamKey string
}

func (s *liveState) SetOnline(streamKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.online = true
	s.streamKey = streamKey
}

func (s *liveState) SetOffline() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.online = false
	s.streamKey = ""
}

func (s *liveState) Snapshot() (bool, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.online, s.streamKey
}

func main() {
	httpAddr := envOrDefault("SOVEREIGNSTREAM_HTTP_ADDR", ":8080")
	rtmpAddr := envOrDefault("SOVEREIGNSTREAM_RTMP_ADDR", ":1935")
	dashboardAddr := envOrDefault("SOVEREIGNSTREAM_DASHBOARD_ADDR", ":3000")
	hlsDir := envOrDefault("SOVEREIGNSTREAM_HLS_DIR", "./hls")
	dbPath := envOrDefault("SOVEREIGNSTREAM_DB_PATH", "./sovereignstream.db")

	cfgStore, err := config.Open(dbPath)
	if err != nil {
		log.Fatalf("failed to open config store: %v", err)
	}
	defer cfgStore.Close()

	cfg, err := cfgStore.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	signalClient, err := signaling.New(cfg.PusherAppID, cfg.PusherKey, cfg.PusherSecret, cfg.PusherCluster)
	if err != nil {
		log.Fatalf("failed to init signaling client: %v", err)
	}

	trans := transcoder.New(hlsDir)
	tunnelManager := tunnel.NewManager()
	state := &liveState{}

	rtmpServer := rtmp.NewServer(rtmpAddr, rtmp.PublishCallbacks{
		OnStart: func(streamKey string) error {
			state.SetOnline(streamKey)
			inputURL := fmt.Sprintf("rtmp://127.0.0.1%s/live/%s", rtmpAddr, streamKey)
			if err := trans.Start(inputURL); err != nil {
				return err
			}
			if err := tunnelManager.Start(context.Background(), "http://127.0.0.1"+httpAddr); err != nil {
				log.Printf("tunnel start error: %v", err)
			}
			url, _ := tunnelManager.URL()
			_ = signalClient.PublishLiveStatus(true, url)
			return nil
		},
		OnStop: func(string) {
			state.SetOffline()
			_ = trans.Stop()
			_ = signalClient.PublishLiveStatus(false, "")
		},
	})

	apiMux := http.NewServeMux()
	apiMux.Handle("/hls/", http.StripPrefix("/hls/", trans.HLSHandler()))
	apiMux.HandleFunc("/api/stream-url", func(w http.ResponseWriter, r *http.Request) {
		url, err := tunnelManager.URL()
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]string{"url": ""})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"url": url})
	})
	apiMux.HandleFunc("/api/live-status", func(w http.ResponseWriter, r *http.Request) {
		online, streamKey := state.Snapshot()
		writeJSON(w, http.StatusOK, map[string]any{"online": online, "streamKey": streamKey})
	})
	apiMux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			cfg, err := cfgStore.Load()
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, cfg)
		case http.MethodPut:
			var incoming config.StreamConfig
			if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
				return
			}
			current, err := cfgStore.Load()
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			mergeConfig(&current, incoming)
			if err := cfgStore.Save(current); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, current)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	apiMux.HandleFunc("/api/webrtc/signal", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			RoomID string         `json:"roomId"`
			Data   map[string]any `json:"data"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
			return
		}
		if err := signalClient.PublishSignal(payload.RoomID, payload.Data); err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
	})
	apiMux.HandleFunc("/api/chat/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			RoomID  string `json:"roomId"`
			Sender  string `json:"sender"`
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
			return
		}
		if err := signalClient.PublishChatMessage(payload.RoomID, payload.Sender, payload.Message); err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
	})
	apiMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	httpServer := &http.Server{Addr: httpAddr, Handler: withCORS(apiMux)}
	dashboardServer := &http.Server{Addr: dashboardAddr, Handler: dashboard.Handler()}

	go func() {
		log.Printf("HTTP API listening on %s", httpAddr)
		if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server error: %v", err)
		}
	}()

	go func() {
		log.Printf("Dashboard listening on %s", dashboardAddr)
		if err := dashboardServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("dashboard server error: %v", err)
		}
	}()

	go func() {
		log.Printf("RTMP listener on %s", rtmpAddr)
		if err := rtmpServer.Start(); err != nil {
			log.Fatalf("rtmp server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	_ = trans.Stop()
	_ = tunnelManager.Stop()
	_ = httpServer.Shutdown(ctx)
	_ = dashboardServer.Shutdown(ctx)
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func envOrDefault(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func mergeConfig(current *config.StreamConfig, incoming config.StreamConfig) {
	if incoming.Title != "" {
		current.Title = incoming.Title
	}
	if incoming.AccentColor != "" {
		current.AccentColor = incoming.AccentColor
	}
	if incoming.BannerURL != "" {
		current.BannerURL = incoming.BannerURL
	}
	if incoming.RTMPStreamKey != "" {
		current.RTMPStreamKey = incoming.RTMPStreamKey
	}
	if incoming.PusherAppID != "" {
		current.PusherAppID = incoming.PusherAppID
	}
	if incoming.PusherKey != "" {
		current.PusherKey = incoming.PusherKey
	}
	if incoming.PusherSecret != "" {
		current.PusherSecret = incoming.PusherSecret
	}
	if incoming.PusherCluster != "" {
		current.PusherCluster = incoming.PusherCluster
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
