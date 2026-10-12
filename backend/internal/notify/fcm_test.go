package notify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestComposeUsesLineAndOperatorText(t *testing.T) {
	tests := []struct {
		name        string
		label, desc string
		wantBody    string
	}{
		{"label and description", "Atividade Programada", "Maiores intervalos entre Osasco e Varginha.", "Atividade Programada · Maiores intervalos entre Osasco e Varginha."},
		{"label only", "Paralisada", "", "Paralisada"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Compose(9, "Esmeralda", "#00a88e", tt.label, tt.desc)

			want := Message{Line: 9, Title: "Linha 9 – Esmeralda", Body: tt.wantBody, ColorHex: "#00a88e"}
			if m != want {
				t.Errorf("Compose() = %+v, want %+v", m, want)
			}
		})
	}
}

func TestSendPostsToTheLineTopic(t *testing.T) {
	var got struct {
		method, path, contentType string
		body                      map[string]any
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.method, got.path, got.contentType = r.Method, r.URL.Path, r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&got.body); err != nil {
			t.Error(err)
		}
		_, _ = io.WriteString(w, `{"name":"projects/status-trens-sp/messages/123"}`)
	}))
	defer srv.Close()

	fcm := &FCM{client: srv.Client(), endpoint: srv.URL + "/v1/projects/status-trens-sp/messages:send"}
	err := fcm.Send(context.Background(), Message{Line: 9, Title: "Linha 9 – Esmeralda", Body: "Paralisada", ColorHex: "#00a88e"})
	if err != nil {
		t.Fatal(err)
	}

	if got.method != http.MethodPost || got.path != "/v1/projects/status-trens-sp/messages:send" {
		t.Errorf("request = %s %s", got.method, got.path)
	}
	if got.contentType != "application/json" {
		t.Errorf("Content-Type = %q", got.contentType)
	}
	want := map[string]any{
		"message": map[string]any{
			"topic":        "linha-9",
			"notification": map[string]any{"title": "Linha 9 – Esmeralda", "body": "Paralisada"},
			"android": map[string]any{
				"priority": "high",
				"ttl":      "3600s",
				"notification": map[string]any{
					"icon":       "ic_line_9",
					"color":      "#00a88e",
					"channel_id": "status",
				},
			},
		},
	}
	if gotJSON, wantJSON := mustJSON(t, got.body), mustJSON(t, want); gotJSON != wantJSON {
		t.Errorf("body =\n%s\nwant\n%s", gotJSON, wantJSON)
	}
}

func TestSendOmitsAnEmptyColor(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		_, _ = io.WriteString(w, `{}`)
	}))
	defer srv.Close()

	fcm := &FCM{client: srv.Client(), endpoint: srv.URL}
	if err := fcm.Send(context.Background(), Message{Line: 19, Title: "Linha 19", Body: "Paralisada"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(body, `"color"`) {
		t.Errorf("body has a color: %s", body)
	}
}

func TestSendAttachesTheLineImage(t *testing.T) {
	var got struct {
		Message struct {
			Android struct {
				Notification struct {
					Image string `json:"image"`
				} `json:"notification"`
			} `json:"android"`
		} `json:"message"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Error(err)
		}
		_, _ = io.WriteString(w, `{}`)
	}))
	defer srv.Close()

	fcm := &FCM{client: srv.Client(), endpoint: srv.URL, ImageBaseURL: "https://images.example/notification"}
	if err := fcm.Send(context.Background(), Message{Line: 9, Title: "Linha 9 – Esmeralda", Body: "Paralisada"}); err != nil {
		t.Fatal(err)
	}
	if want := "https://images.example/notification/linha-9.png"; got.Message.Android.Notification.Image != want {
		t.Errorf("image = %q, want %q", got.Message.Android.Notification.Image, want)
	}
}

func TestSendOmitsTheImageWithoutABaseURL(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		_, _ = io.WriteString(w, `{}`)
	}))
	defer srv.Close()

	fcm := &FCM{client: srv.Client(), endpoint: srv.URL}
	if err := fcm.Send(context.Background(), Message{Line: 9, Title: "Linha 9", Body: "Paralisada"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(body, `"image"`) {
		t.Errorf("body has an image: %s", body)
	}
}

func TestSendReportsFCMErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":{"code":400,"message":"Invalid topic","status":"INVALID_ARGUMENT"}}`)
	}))
	defer srv.Close()

	fcm := &FCM{client: srv.Client(), endpoint: srv.URL}
	err := fcm.Send(context.Background(), Message{Line: 9, Title: "t", Body: "b"})

	if err == nil || !strings.Contains(err.Error(), "400") || !strings.Contains(err.Error(), "Invalid topic") {
		t.Errorf("Send() error = %v, want status and FCM message", err)
	}
}

func TestNewFCMRejectsWhatIsNotAServiceAccount(t *testing.T) {
	_, err := NewFCM(context.Background(), []byte(`{"type":"authorized_user"}`))
	if err == nil {
		t.Error("NewFCM() accepted a non service account credential")
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
