package store

import (
	"bili-auth-backend/internal/model"
	"errors"
)

var ErrRedisNotImplemented = errors.New("redis session store not implemented yet")

type RedisSessionStore struct{}

func NewRedisSessionStore(_ model.Config) *RedisSessionStore {
	return &RedisSessionStore{}
}

func (r *RedisSessionStore) Save(_ *model.LoginSession) error         { return ErrRedisNotImplemented }
func (r *RedisSessionStore) Get(_ string) (*model.LoginSession, bool) { return nil, false }
func (r *RedisSessionStore) Delete(_ string) error                    { return ErrRedisNotImplemented }
func (r *RedisSessionStore) List() ([]*model.LoginSession, error)     { return nil, ErrRedisNotImplemented }
func (r *RedisSessionStore) StartCleanup()                            {}
func (r *RedisSessionStore) StopCleanup()                             {}
