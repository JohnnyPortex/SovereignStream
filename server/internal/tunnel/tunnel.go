package tunnel

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os/exec"
	"regexp"
	"sync"
)

var cloudflareURLRegex = regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)

type Manager struct {
	mu    sync.RWMutex
	cmd   *exec.Cmd
	url   string
	alive bool
}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) Start(ctx context.Context, targetURL string) error {
	m.mu.Lock()
	if m.alive {
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()

	cmd := exec.CommandContext(ctx, "cloudflared", "tunnel", "--url", targetURL, "--no-autoupdate")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	m.mu.Lock()
	m.cmd = cmd
	m.alive = true
	m.mu.Unlock()

	go m.captureURL(stdout)
	go m.captureURL(stderr)
	go func() {
		_ = cmd.Wait()
		m.mu.Lock()
		m.alive = false
		m.cmd = nil
		m.mu.Unlock()
	}()

	return nil
}

func (m *Manager) captureURL(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if match := cloudflareURLRegex.FindString(line); match != "" {
			m.mu.Lock()
			m.url = match
			m.mu.Unlock()
		}
	}
}

func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd == nil || m.cmd.Process == nil {
		m.alive = false
		return nil
	}
	if err := m.cmd.Process.Kill(); err != nil {
		return err
	}
	m.cmd = nil
	m.alive = false
	return nil
}

func (m *Manager) URL() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.url == "" {
		return "", errors.New("tunnel url not available yet")
	}
	return m.url, nil
}
