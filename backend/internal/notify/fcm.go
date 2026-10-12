// Package notify sends status change pushes through Firebase Cloud Messaging.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	scope = "https://www.googleapis.com/auth/firebase.messaging"
	// Channel the app creates for status changes (one channel: favoriting is
	// already the user's choice of what to hear about).
	channelID = "status"
	// A status change is stale after an hour; FCM drops it instead of
	// delivering it late to a phone that was offline.
	ttl = "3600s"
)

// Message is one status change push for the favorites of a line.
type Message struct {
	Line     int
	Title    string
	Body     string
	ColorHex string // tints the line icon; empty leaves the default
}

// Compose builds the push for a line from its name and the operator's text.
func Compose(code int, name, colorHex, label, description string) Message {
	body := label
	if description != "" {
		body += " · " + description
	}
	return Message{Line: code, Title: fmt.Sprintf("Linha %d – %s", code, name), Body: body, ColorHex: colorHex}
}

// FCM sends messages to the per line topics through the HTTP v1 API.
type FCM struct {
	client   *http.Client // adds the OAuth token to every request
	endpoint string
	// ImageBaseURL is where the colored line badges are published, as
	// <ImageBaseURL>/linha-<code>.png. Android shows the badge as the
	// notification's picture; empty sends no picture.
	ImageBaseURL string
}

// NewFCM authenticates with a Firebase service account key (JSON). The
// resulting client refreshes its short lived access token by itself.
func NewFCM(ctx context.Context, serviceAccount []byte) (*FCM, error) {
	creds, err := google.CredentialsFromJSONWithType(ctx, serviceAccount, google.ServiceAccount, scope)
	if err != nil {
		return nil, fmt.Errorf("reading firebase service account: %w", err)
	}
	return &FCM{
		client:   oauth2.NewClient(ctx, creds.TokenSource),
		endpoint: fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", creds.ProjectID),
	}, nil
}

type androidNotification struct {
	Icon      string `json:"icon"`
	Color     string `json:"color,omitempty"`
	Image     string `json:"image,omitempty"`
	ChannelID string `json:"channel_id"`
}

type request struct {
	Message struct {
		Topic        string `json:"topic"`
		Notification struct {
			Title string `json:"title"`
			Body  string `json:"body"`
		} `json:"notification"`
		Android struct {
			Priority     string              `json:"priority"`
			TTL          string              `json:"ttl"`
			Notification androidNotification `json:"notification"`
		} `json:"android"`
	} `json:"message"`
}

// Send pushes m to the topic of its line.
func (f *FCM) Send(ctx context.Context, m Message) error {
	var req request
	req.Message.Topic = fmt.Sprintf("linha-%d", m.Line)
	req.Message.Notification.Title = m.Title
	req.Message.Notification.Body = m.Body
	req.Message.Android.Priority = "high"
	req.Message.Android.TTL = ttl
	req.Message.Android.Notification = androidNotification{
		Icon:      fmt.Sprintf("ic_line_%d", m.Line),
		Color:     m.ColorHex,
		ChannelID: channelID,
	}
	if f.ImageBaseURL != "" {
		req.Message.Android.Notification.Image = fmt.Sprintf("%s/linha-%d.png", f.ImageBaseURL, m.Line)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("encoding fcm message: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, f.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building fcm request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("sending to %s: %w", req.Message.Topic, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("sending to %s: fcm returned %d: %s", req.Message.Topic, resp.StatusCode, detail)
	}
	return nil
}
