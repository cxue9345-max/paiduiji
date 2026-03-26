package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type BilibiliAuthClient struct {
	client         *http.Client
	userAgent      string
	referer        string
	generateURL    string
	pollURL        string
	pollMaxRetries int
	retryInterval  time.Duration
}

type GenerateResp struct {
	Code int    `json:"code"`
	Msg  string `json:"message"`
	Data struct {
		URL       string `json:"url"`
		QRCodeKey string `json:"qrcode_key"`
	} `json:"data"`
}

type PollResp struct {
	Code int    `json:"code"`
	Msg  string `json:"message"`
	Data struct {
		Code         int    `json:"code"`
		Message      string `json:"message"`
		RefreshToken string `json:"refresh_token"`
		Timestamp    int64  `json:"timestamp"`
		URL          string `json:"url"`
	} `json:"data"`
}

func NewBilibiliAuthClient(timeout time.Duration, ua, referer, generateURL, pollURL string, retries int, retryInterval time.Duration) *BilibiliAuthClient {
	return &BilibiliAuthClient{
		client:         &http.Client{Timeout: timeout},
		userAgent:      ua,
		referer:        referer,
		generateURL:    generateURL,
		pollURL:        pollURL,
		pollMaxRetries: retries,
		retryInterval:  retryInterval,
	}
}

func (c *BilibiliAuthClient) GenerateQRCode(ctx context.Context) (*GenerateResp, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.generateURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Referer", c.referer)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var out GenerateResp
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode generate response: %w", err)
	}
	if out.Code != 0 {
		return nil, fmt.Errorf("bilibili generate failed: %d %s", out.Code, out.Msg)
	}
	return &out, nil
}

func (c *BilibiliAuthClient) PollQRCode(ctx context.Context, qrcodeKey string) (*PollResp, http.Header, error) {
	params := url.Values{}
	params.Set("qrcode_key", qrcodeKey)
	params.Set("source", "main-fe-header")
	endpoint := c.pollURL + "?" + params.Encode()

	var lastErr error
	for i := 0; i <= c.pollMaxRetries; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, nil, err
		}
		req.Header.Set("User-Agent", c.userAgent)
		req.Header.Set("Referer", c.referer)
		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			if i < c.pollMaxRetries {
				time.Sleep(c.retryInterval)
				continue
			}
			break
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		var out PollResp
		if err := json.Unmarshal(body, &out); err != nil {
			lastErr = fmt.Errorf("decode poll response: %w", err)
			continue
		}
		return &out, resp.Header, nil
	}
	return nil, nil, fmt.Errorf("poll failed after retries: %w", lastErr)
}
