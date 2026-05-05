package signaling

import (
	"errors"
	"fmt"
	"sync"

	pusher "github.com/pusher/pusher-http-go/v5"
)

type Client struct {
	mu      sync.RWMutex
	enabled bool
	client  *pusher.Client
}

func New(appID, key, secret, cluster string) (*Client, error) {
	if appID == "" || key == "" || secret == "" || cluster == "" {
		return &Client{}, nil
	}
	c := &pusher.Client{
		AppID:   appID,
		Key:     key,
		Secret:  secret,
		Cluster: cluster,
		Secure:  true,
	}
	return &Client{enabled: true, client: c}, nil
}

func (c *Client) IsEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled && c.client != nil
}

func (c *Client) PublishLiveStatus(isOnline bool, streamURL string) error {
	if !c.IsEnabled() {
		return nil
	}
	payload := map[string]any{
		"online":    isOnline,
		"streamUrl": streamURL,
	}
	return c.client.Trigger("live-status", "state-updated", payload)
}

func (c *Client) PublishSignal(roomID string, payload map[string]any) error {
	if !c.IsEnabled() {
		return nil
	}
	if roomID == "" {
		return errors.New("roomID is required")
	}
	channel := fmt.Sprintf("webrtc-%s", roomID)
	return c.client.Trigger(channel, "signal", payload)
}

func (c *Client) PublishChatMessage(roomID, sender, message string) error {
	if !c.IsEnabled() {
		return nil
	}
	if roomID == "" {
		return errors.New("roomID is required")
	}
	if message == "" {
		return errors.New("message is required")
	}
	channel := fmt.Sprintf("chat-%s", roomID)
	payload := map[string]any{
		"sender":  sender,
		"message": message,
	}
	return c.client.Trigger(channel, "new-message", payload)
}
