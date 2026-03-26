package model

import "time"

type SessionStatus string

const (
	SessionCreated        SessionStatus = "created"
	SessionWaitingScan    SessionStatus = "waiting_scan"
	SessionWaitingConfirm SessionStatus = "waiting_confirm"
	SessionConfirmed      SessionStatus = "confirmed"
	SessionExpired        SessionStatus = "expired"
	SessionFailed         SessionStatus = "failed"
)

type PollResultState string

const (
	PollStateWaitingScan    PollResultState = "waiting_scan"
	PollStateWaitingConfirm PollResultState = "waiting_confirm"
	PollStateConfirmed      PollResultState = "confirmed"
	PollStateExpired        PollResultState = "expired"
	PollStateFailed         PollResultState = "failed"
)

type LoginSession struct {
	SessionID        string                `json:"session_id"`
	QRCodeKey        string                `json:"qrcode_key"`
	QRCodeURL        string                `json:"qrcode_url"`
	Status           SessionStatus         `json:"status"`
	CreatedAt        time.Time             `json:"created_at"`
	UpdatedAt        time.Time             `json:"updated_at"`
	ExpiresAt        time.Time             `json:"expires_at"`
	Cookies          map[string]string     `json:"cookies,omitempty"`
	CookieString     string                `json:"cookie_string,omitempty"`
	CookieMeta       map[string]CookieMeta `json:"cookie_meta,omitempty"`
	CookieComplete   bool                  `json:"cookie_complete"`
	MissingKeys      []string              `json:"missing_keys,omitempty"`
	LastPollResponse map[string]any        `json:"last_poll_response,omitempty"`
	ErrorMessage     string                `json:"error_message,omitempty"`
}

type CookieMeta struct {
	Path     string    `json:"path,omitempty"`
	Domain   string    `json:"domain,omitempty"`
	Expires  time.Time `json:"expires,omitempty"`
	Secure   bool      `json:"secure,omitempty"`
	HTTPOnly bool      `json:"http_only,omitempty"`
	Raw      string    `json:"raw,omitempty"`
}

type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    any         `json:"data,omitempty"`
	Details interface{} `json:"details,omitempty"`
}
