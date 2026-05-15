// Package webhook fires moderation/event notifications at an external URL.
// All sends are best-effort and asynchronous — failures are logged but never
// affect the response cycle for the user-facing request.
package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// Notifier sends JSON payloads to a configured URL. A nil/empty URL turns the
// notifier into a no-op so callers don't need to nil-check before every Send.
type Notifier struct {
	URL    string
	Client *http.Client
	Logger *slog.Logger
}

// New returns a Notifier that posts to url. A short timeout keeps slow
// downstream receivers from accumulating goroutines.
func New(url string, logger *slog.Logger) *Notifier {
	if logger == nil {
		logger = slog.Default()
	}
	return &Notifier{
		URL:    url,
		Client: &http.Client{Timeout: 5 * time.Second},
		Logger: logger,
	}
}

// Send fires the payload at the configured URL asynchronously. A nil notifier
// or empty URL is a no-op — that's the supported "webhook disabled" state.
func (n *Notifier) Send(payload map[string]any) {
	if n == nil || n.URL == "" {
		return
	}
	go n.send(payload)
}

func (n *Notifier) send(payload map[string]any) {
	buf, err := json.Marshal(payload)
	if err != nil {
		n.Logger.Warn("webhook marshal failed", "err", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.URL, bytes.NewReader(buf))
	if err != nil {
		n.Logger.Warn("webhook build request failed", "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Errolan-Webhook/1.0")
	resp, err := n.Client.Do(req)
	if err != nil {
		n.Logger.Warn("webhook send failed", "err", err, "url", n.URL)
		return
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		n.Logger.Warn("webhook non-2xx", "status", resp.StatusCode, "url", n.URL)
	}
}
