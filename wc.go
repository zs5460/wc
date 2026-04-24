package wc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	baseURL     = "https://qyapi.weixin.qq.com/cgi-bin/"
	getTokenURL = baseURL + "gettoken?corpid=%s&corpsecret=%s"
	sendMsgURL  = baseURL + "message/send?access_token=%s"
	// 企业微信 access_token 有效期 7200s，提前刷新避免边界失败
	tokenRefreshInterval = 3600 * time.Second
	httpTimeout          = 10 * time.Second
)

const httpStatusErrFmt = "unexpected HTTP status %d: %s"

type wechat struct {
	appID       string
	appKey      string
	agentID     string
	accessToken string
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	client      *http.Client
}

type result struct {
	Code    int    `json:"errcode"`
	Message string `json:"errmsg"`
	Token   string `json:"access_token"`
}

type textBody struct {
	Content string `json:"content"`
}

type message struct {
	ToUser  string   `json:"touser"`
	MsgType string   `json:"msgtype"`
	AgentID string   `json:"agentid"`
	Text    textBody `json:"text"`
}

func New(appid, appkey, agentid string) (*wechat, error) {
	ctx, cancel := context.WithCancel(context.Background())
	wc := &wechat{
		appID:   appid,
		appKey:  appkey,
		agentID: agentid,
		ctx:     ctx,
		cancel:  cancel,
		client: &http.Client{
			Timeout: httpTimeout,
		},
	}
	if err := wc.getToken(); err != nil {
		cancel()
		return nil, err
	}
	go wc.tokenRefresher()
	return wc, nil
}

func (wc *wechat) tokenRefresher() {
	ticker := time.NewTicker(tokenRefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			_ = wc.getToken()
		case <-wc.ctx.Done():
			return
		}
	}
}

func (wc *wechat) getToken() error {
	var ret result
	url := fmt.Sprintf(getTokenURL, wc.appID, wc.appKey)
	if err := wc.getJSON(url, &ret); err != nil {
		return fmt.Errorf("get token request: %w", err)
	}
	if ret.Code != 0 {
		return fmt.Errorf("get token failed (errcode=%d): %s", ret.Code, ret.Message)
	}
	wc.mu.Lock()
	wc.accessToken = ret.Token
	wc.mu.Unlock()
	return nil
}

func (wc *wechat) Send(to, msg string) error {
	url := fmt.Sprintf(sendMsgURL, wc.token())

	m := message{
		ToUser:  to,
		MsgType: "text",
		AgentID: wc.agentID,
		Text:    textBody{Content: msg},
	}

	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	var ret result
	err = wc.doJSON(http.MethodPost, url, data, &ret)
	if err != nil {
		return fmt.Errorf("send request error: %w", err)
	}

	if ret.Code != 0 {
		return fmt.Errorf("send message failed (errcode=%d): %s", ret.Code, ret.Message)
	}
	return nil
}

func (wc *wechat) Close() {
	wc.cancel()
}

func (wc *wechat) getJSON(url string, v any) error {
	return wc.doJSON(http.MethodGet, url, nil, v)
}

func (wc *wechat) doJSON(method, url string, body []byte, v any) error {
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(wc.ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("build %s request: %w", method, err)
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	}

	resp, err := wc.client.Do(req)
	if err != nil {
		return fmt.Errorf("%s request failed: %w", method, err)
	}
	defer resp.Body.Close()

	reply, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read %s response body: %w", method, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(httpStatusErrFmt, resp.StatusCode, bytes.TrimSpace(reply))
	}
	if err := json.Unmarshal(reply, v); err != nil {
		return fmt.Errorf("decode %s response: %w", method, err)
	}
	return nil
}

func (wc *wechat) token() string {
	wc.mu.RLock()
	defer wc.mu.RUnlock()
	return wc.accessToken
}
