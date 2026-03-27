package handler

import (
	"bili-auth-backend/internal/httpclient"
	"bili-auth-backend/internal/service/auth"
	"bili-auth-backend/internal/utils"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type LiveHandler struct {
	authSvc *auth.Service
	client  *httpclient.BilibiliLiveClient
}

func NewLiveHandler(authSvc *auth.Service, client *httpclient.BilibiliLiveClient) *LiveHandler {
	return &LiveHandler{authSvc: authSvc, client: client}
}

func (h *LiveHandler) Register(r *gin.Engine) {
	api := r.Group("/api/live")
	api.GET("/room/init", h.RoomInit)
	api.GET("/room/news", h.RoomNews)
	api.GET("/danmu/info", h.DanmuInfo)
	api.GET("/room/info-by-user", h.InfoByUser)
	api.GET("/user/nav", h.UserNav)
	api.GET("/relation/stat", h.RelationStat)
	api.POST("/msg/send", h.SendMessage)
}

func (h *LiveHandler) RoomInit(c *gin.Context) {
	s, roomID, ok := h.getSessionAndRoomID(c)
	if !ok {
		return
	}
	data, err := h.client.RoomInit(c.Request.Context(), roomID, s.Cookies)
	if err != nil {
		utils.Fail(c, http.StatusBadGateway, 2001, "room init failed", err.Error())
		return
	}
	utils.OK(c, "ok", data)
}

func (h *LiveHandler) RoomNews(c *gin.Context) {
	s, roomID, ok := h.getSessionAndRoomID(c)
	if !ok {
		return
	}
	data, err := h.client.RoomNews(c.Request.Context(), roomID, s.Cookies)
	if err != nil {
		utils.Fail(c, http.StatusBadGateway, 2002, "room news failed", err.Error())
		return
	}
	utils.OK(c, "ok", data)
}

func (h *LiveHandler) DanmuInfo(c *gin.Context) {
	s, roomID, ok := h.getSessionAndRoomID(c)
	if !ok {
		return
	}
	data, err := h.client.DanmuInfo(c.Request.Context(), roomID, s.Cookies)
	if err != nil {
		utils.Fail(c, http.StatusBadGateway, 2003, "getDanmuInfo failed", err.Error())
		return
	}
	utils.OK(c, "ok", data)
}

func (h *LiveHandler) InfoByUser(c *gin.Context) {
	s, roomID, ok := h.getSessionAndRoomID(c)
	if !ok {
		return
	}
	data, err := h.client.InfoByUser(c.Request.Context(), roomID, s.Cookies)
	if err != nil {
		utils.Fail(c, http.StatusBadGateway, 2004, "getInfoByUser failed", err.Error())
		return
	}
	utils.OK(c, "ok", data)
}

func (h *LiveHandler) UserNav(c *gin.Context) {
	s, ok := h.getSession(c)
	if !ok {
		return
	}
	data, err := h.client.UserNav(c.Request.Context(), s.Cookies)
	if err != nil {
		utils.Fail(c, http.StatusBadGateway, 2005, "user nav failed", err.Error())
		return
	}
	utils.OK(c, "ok", data)
}

func (h *LiveHandler) RelationStat(c *gin.Context) {
	s, ok := h.getSession(c)
	if !ok {
		return
	}
	vmid, ok := queryInt(c, "vmid")
	if !ok {
		utils.Fail(c, http.StatusBadRequest, 1400, "invalid vmid", nil)
		return
	}
	data, err := h.client.RelationStat(c.Request.Context(), vmid, s.Cookies)
	if err != nil {
		utils.Fail(c, http.StatusBadGateway, 2006, "relation stat failed", err.Error())
		return
	}
	utils.OK(c, "ok", data)
}

type sendMessageRequest struct {
	SessionID string `json:"session_id"`
	RoomID    int    `json:"room_id"`
	Message   string `json:"message"`
}

func (h *LiveHandler) SendMessage(c *gin.Context) {
	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Fail(c, http.StatusBadRequest, 1400, "invalid request body", err.Error())
		return
	}
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.Message = strings.TrimSpace(req.Message)
	if req.SessionID == "" || req.RoomID <= 0 || req.Message == "" {
		utils.Fail(c, http.StatusBadRequest, 1400, "session_id / room_id / message is required", nil)
		return
	}
	s, err := h.authSvc.GetSession(req.SessionID)
	if err != nil {
		utils.Fail(c, http.StatusNotFound, 1002, "session not found", nil)
		return
	}
	data, err := h.client.SendMessage(c.Request.Context(), req.RoomID, req.Message, s.Cookies)
	if err != nil {
		utils.Fail(c, http.StatusBadGateway, 2007, "send message failed", err.Error())
		return
	}
	utils.OK(c, "ok", data)
}

func (h *LiveHandler) getSessionAndRoomID(c *gin.Context) (*authSession, int, bool) {
	s, ok := h.getSession(c)
	if !ok {
		return nil, 0, false
	}
	roomID, ok := queryInt(c, "room_id")
	if !ok {
		utils.Fail(c, http.StatusBadRequest, 1400, "invalid room_id", nil)
		return nil, 0, false
	}
	return s, roomID, true
}

type authSession struct {
	Cookies map[string]string
}

func (h *LiveHandler) getSession(c *gin.Context) (*authSession, bool) {
	sessionID := strings.TrimSpace(c.Query("session_id"))
	if sessionID == "" {
		utils.Fail(c, http.StatusBadRequest, 1400, "session_id is required", nil)
		return nil, false
	}
	s, err := h.authSvc.GetSession(sessionID)
	if err != nil {
		utils.Fail(c, http.StatusNotFound, 1002, "session not found", nil)
		return nil, false
	}
	if len(s.Cookies) == 0 {
		utils.Fail(c, http.StatusBadRequest, 1401, "session cookies are empty, poll qrcode first", nil)
		return nil, false
	}
	return &authSession{Cookies: s.Cookies}, true
}

func queryInt(c *gin.Context, key string) (int, bool) {
	v := strings.TrimSpace(c.Query(key))
	if v == "" {
		return 0, false
	}
	val, err := strconv.Atoi(v)
	if err != nil || val <= 0 {
		return 0, false
	}
	return val, true
}
