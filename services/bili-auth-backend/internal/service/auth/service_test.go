package auth

import (
	"bili-auth-backend/internal/model"
	"testing"
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
