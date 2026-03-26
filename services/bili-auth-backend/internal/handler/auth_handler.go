package handler

import (
	"bili-auth-backend/internal/model"
	"bili-auth-backend/internal/service/auth"
	"bili-auth-backend/internal/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	qrcode "github.com/skip2/go-qrcode"
)

type AuthHandler struct {
	svc   *auth.Service
	debug bool
}

func NewAuthHandler(svc *auth.Service, debug bool) *AuthHandler {
	return &AuthHandler{svc: svc, debug: debug}
}

func (h *AuthHandler) Register(r *gin.Engine) {
	api := r.Group("/api/auth")
	api.POST("/qrcode/start", h.Start)
	api.GET("/qrcode/image/:session_id", h.Image)
	api.GET("/qrcode/poll/:session_id", h.Poll)
	api.GET("/session/:session_id", h.Session)
	api.POST("/logout/:session_id", h.Logout)
}

func (h *AuthHandler) Start(c *gin.Context) {
	s, err := h.svc.StartSession(c.Request.Context())
	if err != nil {
		utils.Fail(c, http.StatusBadGateway, 1001, "failed to start qrcode session", err.Error())
		return
	}
	utils.OK(c, "ok", gin.H{
		"session_id":   s.SessionID,
		"qrcode_key":   s.QRCodeKey,
		"qrcode_url":   s.QRCodeURL,
		"expires_in":   int(time.Until(s.ExpiresAt).Seconds()),
		"qrcode_image": "/api/auth/qrcode/image/" + s.SessionID,
	})
}

func (h *AuthHandler) Image(c *gin.Context) {
	id := c.Param("session_id")
	s, err := h.svc.GetSession(id)
	if err != nil {
		utils.Fail(c, http.StatusNotFound, 1002, "session not found", nil)
		return
	}
	png, err := qrcode.Encode(s.QRCodeURL, qrcode.Medium, 256)
	if err != nil {
		utils.Fail(c, http.StatusInternalServerError, 1003, "failed to generate image", err.Error())
		return
	}
	c.Header("Content-Type", "image/png")
	c.Writer.WriteHeader(http.StatusOK)
	_, _ = c.Writer.Write(png)
}

func (h *AuthHandler) Poll(c *gin.Context) {
	id := c.Param("session_id")
	s, state, err := h.svc.Poll(c.Request.Context(), id)
	if err != nil && err != auth.ErrNotFound {
		utils.Fail(c, http.StatusBadGateway, 1004, "poll request failed", gin.H{"state": state, "error": err.Error()})
		return
	}
	if err == auth.ErrNotFound {
		utils.Fail(c, http.StatusNotFound, 1002, "session not found", nil)
		return
	}
	resp := h.buildSessionData(s)
	resp["poll_state"] = state
	resp["message"] = s.ErrorMessage
	if h.debug {
		resp["cookie_string"] = s.CookieString
		resp["cookie_map"] = s.Cookies
	}
	utils.OK(c, "ok", resp)
}

func (h *AuthHandler) Session(c *gin.Context) {
	id := c.Param("session_id")
	s, err := h.svc.GetSession(id)
	if err != nil {
		utils.Fail(c, http.StatusNotFound, 1002, "session not found", nil)
		return
	}
	data := h.buildSessionData(s)
	data["created_at"] = s.CreatedAt
	data["expires_at"] = s.ExpiresAt
	data["updated_at"] = s.UpdatedAt
	data["error_message"] = s.ErrorMessage
	if h.debug {
		data["cookie_string"] = s.CookieString
		data["cookie_map"] = s.Cookies
		data["cookie_meta"] = s.CookieMeta
	}
	utils.OK(c, "ok", data)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	id := c.Param("session_id")
	if err := h.svc.Logout(id); err != nil {
		utils.Fail(c, http.StatusNotFound, 1002, "session not found", nil)
		return
	}
	utils.OK(c, "ok", gin.H{"session_id": id, "logged_out": true})
}

func (h *AuthHandler) buildSessionData(s *model.LoginSession) gin.H {
	return gin.H{
		"session_id":      s.SessionID,
		"status":          s.Status,
		"cookie_captured": len(s.Cookies) > 0,
		"cookie_count":    len(s.Cookies),
		"cookie_complete": s.CookieComplete,
		"missing_keys":    s.MissingKeys,
	}
}
