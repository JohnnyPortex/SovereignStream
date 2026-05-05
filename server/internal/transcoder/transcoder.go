package transcoder

import (
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

type Transcoder struct {
	mu      sync.RWMutex
	hlsDir  string
	cmd     *exec.Cmd
	running bool
}

func New(hlsDir string) *Transcoder {
	return &Transcoder{hlsDir: hlsDir}
}

func (t *Transcoder) Start(inputURL string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		return nil
	}
	if err := os.MkdirAll(t.hlsDir, 0o755); err != nil {
		return err
	}

	playlist := filepath.Join(t.hlsDir, "index.m3u8")
	_ = os.Remove(playlist)

	args := []string{
		"-y",
		"-i", inputURL,
		// video: low-latency x264 tuning
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-tune", "zerolatency",
		"-g", "30",
		"-sc_threshold", "0",
		"-keyint_min", "30",
		// audio
		"-c:a", "aac",
		"-ar", "44100",
		"-b:a", "128k",
		// HLS low-latency: 1s segments, keep last 3
		"-f", "hls",
		"-hls_time", "1",
		"-hls_list_size", "3",
		"-hls_flags", "delete_segments+append_list+omit_endlist+independent_segments",
		"-hls_segment_type", "mpegts",
		playlist,
	}

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}

	t.cmd = cmd
	t.running = true

	go func() {
		_ = cmd.Wait()
		t.mu.Lock()
		t.running = false
		t.cmd = nil
		t.mu.Unlock()
	}()

	return nil
}

func (t *Transcoder) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running || t.cmd == nil || t.cmd.Process == nil {
		return nil
	}
	if err := t.cmd.Process.Kill(); err != nil {
		return err
	}
	t.running = false
	t.cmd = nil
	return nil
}

func (t *Transcoder) IsRunning() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.running
}

func (t *Transcoder) HLSHandler() http.Handler {
	return http.FileServer(http.Dir(t.hlsDir))
}

func (t *Transcoder) PlaylistPath() (string, error) {
	playlist := filepath.Join(t.hlsDir, "index.m3u8")
	if _, err := os.Stat(playlist); err != nil {
		return "", errors.New("playlist not ready")
	}
	return playlist, nil
}
