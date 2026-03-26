package auth

import (
	"bili-auth-backend/internal/httpclient"
	"bili-auth-backend/internal/model"
	"bili-auth-backend/internal/store"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestConvertPollCode(t *testing.T) {
	cases := map[int]model.PollResultState{
		86101: model.PollStateWaitingScan,
		86090: model.PollStateWaitingConfirm,
		0:     model.PollStateConfirmed,
		86038: model.PollStateExpired,
		99999: model.PollStateFailed,
	}
	for code, want := range cases {
		if got := convertPollCode(code); got != want {
			t.Fatalf("code %d: got %s want %s", code, got, want)
		}
	}
}

func TestStateToSessionStatus(t *testing.T) {
	if stateToSessionStatus(model.PollStateConfirmed) != model.SessionConfirmed {
		t.Fatal("expected confirmed status")
	}
	if stateToSessionStatus(model.PollStateFailed) != model.SessionFailed {
		t.Fatal("expected failed status")
	}
}

func TestPollRejectsNonZeroTopLevelCode(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"code":-412,"message":"risk control"}`)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	st := store.NewMemorySessionStore(time.Minute, logger)
	client := httpclient.NewBilibiliAuthClient(
		time.Second,
		"ua",
		"ref",
		server.URL+"/generate",
		server.URL,
		0,
		time.Millisecond,
	)
	svc := NewService(st, client, model.Config{SessionTTL: time.Minute}, logger)

	session := &model.LoginSession{
		SessionID:  "sid",
		QRCodeKey:  "qrcode-key",
		Status:     model.SessionCreated,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(time.Minute),
		Cookies:    map[string]string{},
		CookieMeta: map[string]model.CookieMeta{},
	}
	if err := st.Save(session); err != nil {
		t.Fatalf("save session: %v", err)
	}

	gotSession, state, err := svc.Poll(context.Background(), session.SessionID)
	if err != nil {
		t.Fatalf("poll should not return transport error: %v", err)
	}
	if state != model.PollStateFailed {
		t.Fatalf("state = %s, want %s", state, model.PollStateFailed)
	}
	if gotSession.Status != model.SessionFailed {
		t.Fatalf("status = %s, want %s", gotSession.Status, model.SessionFailed)
	}
	if gotSession.ErrorMessage == "" {
		t.Fatal("expected error message for non-zero top-level code")
	}
}
