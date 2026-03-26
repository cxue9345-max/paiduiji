package auth

import (
	"bili-auth-backend/internal/httpclient"
	"bili-auth-backend/internal/model"
	"bili-auth-backend/internal/store"
	"bili-auth-backend/internal/utils"
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("session not found")

type Service struct {
	store  store.SessionStore
	client *httpclient.BilibiliAuthClient
	cfg    model.Config
	logger *slog.Logger
}

func NewService(st store.SessionStore, client *httpclient.BilibiliAuthClient, cfg model.Config, logger *slog.Logger) *Service {
	return &Service{store: st, client: client, cfg: cfg, logger: logger}
}

func (s *Service) StartSession(ctx context.Context) (*model.LoginSession, error) {
	res, err := s.client.GenerateQRCode(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	session := &model.LoginSession{
		SessionID:  uuid.NewString(),
		QRCodeKey:  res.Data.QRCodeKey,
		QRCodeURL:  res.Data.URL,
		Status:     model.SessionCreated,
		CreatedAt:  now,
		UpdatedAt:  now,
		ExpiresAt:  now.Add(s.cfg.SessionTTL),
		Cookies:    map[string]string{},
		CookieMeta: map[string]model.CookieMeta{},
	}
	if err := s.store.Save(session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Service) GetSession(id string) (*model.LoginSession, error) {
	session, ok := s.store.Get(id)
	if !ok {
		return nil, ErrNotFound
	}
	if time.Now().After(session.ExpiresAt) {
		session.Status = model.SessionExpired
		session.UpdatedAt = time.Now()
		s.store.Save(session)
		return session, nil
	}
	return session, nil
}

func (s *Service) Poll(ctx context.Context, id string) (*model.LoginSession, model.PollResultState, error) {
	session, err := s.GetSession(id)
	if err != nil {
		return nil, model.PollStateFailed, err
	}
	if session.Status == model.SessionExpired {
		return session, model.PollStateExpired, nil
	}
	resp, headers, err := s.client.PollQRCode(ctx, session.QRCodeKey)
	if err != nil {
		session.Status = model.SessionFailed
		session.ErrorMessage = err.Error()
		session.UpdatedAt = time.Now()
		s.store.Save(session)
		return session, model.PollStateFailed, err
	}

	session.LastPollResponse = map[string]any{
		"code":         resp.Code,
		"message":      resp.Msg,
		"data_code":    resp.Data.Code,
		"data_message": resp.Data.Message,
	}

	state := convertPollCode(resp.Data.Code)
	session.Status = stateToSessionStatus(state)
	session.UpdatedAt = time.Now()

	if state == model.PollStateConfirmed {
		jar := utils.NewCookieJar()
		jar.AddFromSetCookieHeader(headers)
		jar.Merge(session.Cookies)
		session.Cookies = jar.Values
		session.CookieMeta = jar.Meta
		session.CookieString = jar.CookieString()

		s.EnrichCookies(ctx, session)

		missing := utils.MissingRequiredCookies(session.Cookies)
		session.MissingKeys = missing
		session.CookieComplete = len(missing) == 0
		if !session.CookieComplete {
			session.ErrorMessage = "cookie captured but key cookies are missing"
		}
		s.logger.Info("poll confirmed",
			"session_id", session.SessionID,
			"cookie_count", len(session.Cookies),
			"SESSDATA", utils.MaskValue(session.Cookies["SESSDATA"]))
	}

	if state == model.PollStateExpired {
		session.ExpiresAt = time.Now()
	}

	if err := s.store.Save(session); err != nil {
		return nil, model.PollStateFailed, err
	}
	return session, state, nil
}

func (s *Service) EnrichCookies(_ context.Context, session *model.LoginSession) {
	// Reserved extension point: enrich incomplete cookie sets.
	// Candidates: buvid3, buvid4, bili_ticket.
	if _, ok := session.Cookies["buvid3"]; !ok {
		session.ErrorMessage = "current cookies may be incomplete; enrichment pipeline placeholder enabled"
	}
}

func (s *Service) Logout(id string) error {
	if _, ok := s.store.Get(id); !ok {
		return ErrNotFound
	}
	return s.store.Delete(id)
}

func convertPollCode(code int) model.PollResultState {
	switch code {
	case 86101:
		return model.PollStateWaitingScan
	case 86090:
		return model.PollStateWaitingConfirm
	case 0:
		return model.PollStateConfirmed
	case 86038:
		return model.PollStateExpired
	default:
		return model.PollStateFailed
	}
}

func stateToSessionStatus(state model.PollResultState) model.SessionStatus {
	switch state {
	case model.PollStateWaitingScan:
		return model.SessionWaitingScan
	case model.PollStateWaitingConfirm:
		return model.SessionWaitingConfirm
	case model.PollStateConfirmed:
		return model.SessionConfirmed
	case model.PollStateExpired:
		return model.SessionExpired
	default:
		return model.SessionFailed
	}
}
