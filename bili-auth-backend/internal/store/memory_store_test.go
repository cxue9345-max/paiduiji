package store

import (
	"bili-auth-backend/internal/model"
	"bili-auth-backend/internal/utils"
	"testing"
	"time"
)

func TestSessionExpiryCleanup(t *testing.T) {
	logger := utils.NewJSONLogger()
	st := NewMemorySessionStore(20*time.Millisecond, logger)
	st.StartCleanup()
	defer st.StopCleanup()

	s := &model.LoginSession{
		SessionID: "s1",
		Status:    model.SessionCreated,
		ExpiresAt: time.Now().Add(50 * time.Millisecond),
	}
	if err := st.Save(s); err != nil {
		t.Fatal(err)
	}
	time.Sleep(120 * time.Millisecond)
	if _, ok := st.Get("s1"); ok {
		t.Fatal("session should have been cleaned up")
	}
}
