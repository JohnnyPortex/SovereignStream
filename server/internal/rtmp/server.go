package rtmp

import (
	"errors"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/format/rtmp"
)

type PublishCallbacks struct {
	OnStart func(streamKey string) error
	OnStop  func(streamKey string)
}

type Server struct {
	addr      string
	callbacks PublishCallbacks
	server    *rtmp.Server
	hub       *streamHub
}

func NewServer(addr string, callbacks PublishCallbacks) *Server {
	srv := &Server{addr: addr, callbacks: callbacks, hub: newStreamHub()}
	rtmpServer := &rtmp.Server{
		Addr: addr,
	}
	rtmpServer.HandlePublish = srv.handlePublish
	rtmpServer.HandlePlay = srv.handlePlay
	srv.server = rtmpServer
	return srv
}

func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

func (s *Server) handlePublish(conn *rtmp.Conn) {
	streamKey := parseStreamKey(conn.URL)
	if s.callbacks.OnStart != nil {
		if err := s.callbacks.OnStart(streamKey); err != nil {
			_ = conn.Close()
			return
		}
	}
	defer s.hub.reset()

	streams, err := conn.Streams()
	if err != nil {
		_ = conn.Close()
		return
	}
	s.hub.setStreams(streams)
	for {
		pkt, err := conn.ReadPacket()
		if err != nil {
			break
		}
		s.hub.broadcast(clonePacket(pkt))
	}

	if s.callbacks.OnStop != nil {
		s.callbacks.OnStop(streamKey)
	}
}

func (s *Server) handlePlay(conn *rtmp.Conn) {
	streams, err := s.waitForStreams(8 * time.Second)
	if err != nil {
		_ = conn.Close()
		return
	}
	if err := conn.WriteHeader(streams); err != nil {
		_ = conn.Close()
		return
	}

	sub, cancel := s.hub.subscribe()
	defer cancel()

	for pkt := range sub {
		if err := conn.WritePacket(pkt); err != nil {
			break
		}
	}
	_ = conn.Close()
}

func parseStreamKey(rawURL *url.URL) string {
	if rawURL == nil {
		return "default"
	}
	path := strings.Trim(rawURL.Path, "/")
	if path == "" {
		return "default"
	}
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

func clonePacket(pkt av.Packet) av.Packet {
	clone := pkt
	if len(pkt.Data) > 0 {
		clone.Data = append([]byte(nil), pkt.Data...)
	}
	return clone
}

func (s *Server) waitForStreams(timeout time.Duration) ([]av.CodecData, error) {
	deadline := time.Now().Add(timeout)
	for {
		streams := s.hub.getStreams()
		if len(streams) > 0 {
			return streams, nil
		}
		if time.Now().After(deadline) {
			return nil, errors.New("stream metadata not available")
		}
		time.Sleep(100 * time.Millisecond)
	}
}

type streamHub struct {
	mu          sync.RWMutex
	nextID      int
	streams     []av.CodecData
	subscribers map[int]chan av.Packet
}

func newStreamHub() *streamHub {
	return &streamHub{
		subscribers: make(map[int]chan av.Packet),
	}
}

func (h *streamHub) setStreams(streams []av.CodecData) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.streams = streams
}

func (h *streamHub) getStreams() []av.CodecData {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.streams
}

func (h *streamHub) subscribe() (<-chan av.Packet, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan av.Packet, 256)
	id := h.nextID
	h.nextID++
	h.subscribers[id] = ch

	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if c, ok := h.subscribers[id]; ok {
			delete(h.subscribers, id)
			close(c)
		}
	}
}

func (h *streamHub) broadcast(pkt av.Packet) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.subscribers {
		select {
		case ch <- pkt:
		default:
		}
	}
}

func (h *streamHub) reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.streams = nil
	for id, ch := range h.subscribers {
		close(ch)
		delete(h.subscribers, id)
	}
}
