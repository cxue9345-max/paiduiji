package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	roomInitURL     = "https://api.live.bilibili.com/room/v1/Room/room_init"
	roomNewsURL     = "https://api.live.bilibili.com/room_ex/v1/RoomNews/get"
	danmuInfoURL    = "https://api.live.bilibili.com/xlive/web-room/v1/index/getDanmuInfo"
	roomInfoByUser  = "https://api.live.bilibili.com/xlive/web-room/v1/index/getInfoByUser"
	sendMessageURL  = "https://api.live.bilibili.com/msg/send"
	userNavURL      = "https://api.bilibili.com/x/web-interface/nav"
	relationStatURL = "https://api.bilibili.com/x/relation/stat"
)

type BilibiliLiveClient struct {
	client    *http.Client
	userAgent string
	referer   string
}

func NewBilibiliLiveClient(timeout time.Duration, ua, referer string) *BilibiliLiveClient {
	return &BilibiliLiveClient{
		client:    &http.Client{Timeout: timeout},
		userAgent: ua,
		referer:   referer,
	}
}

func (c *BilibiliLiveClient) RoomInit(ctx context.Context, roomID int, cookies map[string]string) (map[string]any, error) {
	params := url.Values{}
	params.Set("id", fmt.Sprintf("%d", roomID))
	return c.getJSON(ctx, roomInitURL+"?"+params.Encode(), cookies)
}

func (c *BilibiliLiveClient) RoomNews(ctx context.Context, roomID int, cookies map[string]string) (map[string]any, error) {
	params := url.Values{}
	params.Set("roomid", fmt.Sprintf("%d", roomID))
	return c.getJSON(ctx, roomNewsURL+"?"+params.Encode(), cookies)
}

func (c *BilibiliLiveClient) DanmuInfo(ctx context.Context, roomID int, cookies map[string]string) (map[string]any, error) {
	params := url.Values{}
	params.Set("id", fmt.Sprintf("%d", roomID))
	params.Set("type", "0")
	return c.getJSON(ctx, danmuInfoURL+"?"+params.Encode(), cookies)
}

func (c *BilibiliLiveClient) InfoByUser(ctx context.Context, roomID int, cookies map[string]string) (map[string]any, error) {
	params := url.Values{}
	params.Set("room_id", fmt.Sprintf("%d", roomID))
	return c.getJSON(ctx, roomInfoByUser+"?"+params.Encode(), cookies)
}

func (c *BilibiliLiveClient) UserNav(ctx context.Context, cookies map[string]string) (map[string]any, error) {
	return c.getJSON(ctx, userNavURL, cookies)
}

func (c *BilibiliLiveClient) RelationStat(ctx context.Context, vmid int, cookies map[string]string) (map[string]any, error) {
	params := url.Values{}
	params.Set("vmid", fmt.Sprintf("%d", vmid))
	return c.getJSON(ctx, relationStatURL+"?"+params.Encode(), cookies)
}

func (c *BilibiliLiveClient) SendMessage(ctx context.Context, roomID int, message string, cookies map[string]string) (map[string]any, error) {
	csrf := strings.TrimSpace(cookies["bili_jct"])
	if csrf == "" {
		return nil, fmt.Errorf("missing bili_jct in session cookies")
	}
	form := url.Values{}
	form.Set("bubble", "0")
	form.Set("msg", message)
	form.Set("color", "16777215")
	form.Set("mode", "1")
	form.Set("fontsize", "25")
	form.Set("rnd", fmt.Sprintf("%d", time.Now().Unix()))
	form.Set("roomid", fmt.Sprintf("%d", roomID))
	form.Set("csrf", csrf)
	form.Set("csrf_token", csrf)
	return c.postFormJSON(ctx, sendMessageURL, form, cookies)
}

func (c *BilibiliLiveClient) getJSON(ctx context.Context, endpoint string, cookies map[string]string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	c.applyHeaders(req, cookies)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return decodeResponse(resp)
}

func (c *BilibiliLiveClient) postFormJSON(ctx context.Context, endpoint string, form url.Values, cookies map[string]string) (map[string]any, error) {
	body := bytes.NewBufferString(form.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return nil, err
	}
	c.applyHeaders(req, cookies)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return decodeResponse(resp)
}

func (c *BilibiliLiveClient) applyHeaders(req *http.Request, cookies map[string]string) {
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	if c.referer != "" {
		req.Header.Set("Referer", c.referer)
	}
	if cookieHeader := buildCookieHeader(cookies); cookieHeader != "" {
		req.Header.Set("Cookie", cookieHeader)
	}
}

func decodeResponse(resp *http.Response) (map[string]any, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode json failed: %w", err)
	}
	out["_http_status"] = resp.StatusCode
	return out, nil
}

func buildCookieHeader(cookies map[string]string) string {
	if len(cookies) == 0 {
		return ""
	}
	keys := make([]string, 0, len(cookies))
	for k := range cookies {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		if strings.TrimSpace(k) == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", k, cookies[k]))
	}
	return strings.Join(parts, "; ")
}
