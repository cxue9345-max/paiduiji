package store

import "bili-auth-backend/internal/model"

type SessionStore interface {
	Save(session *model.LoginSession) error
	Get(sessionID string) (*model.LoginSession, bool)
	Delete(sessionID string) error
	List() ([]*model.LoginSession, error)
	StartCleanup()
	StopCleanup()
}
