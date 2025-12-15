package wc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	conBaseURL     = "https://qyapi.weixin.qq.com/cgi-bin/"
	conGetTokenURL = conBaseURL + "gettoken?corpid=%s&corpsecret=%s"
	conSendMsgURL  = conBaseURL + "message/send?access_token=%s"
)

type wechat struct {
	AppID       string
	AppKey      string
	AgentID     string
	AccessToken string
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

type result struct {
	Code    int    `json:"errcode"`
	Message string `json:"errmsg"`
	Token   string `json:"access_token"`
}

type message struct {
	ToUser  string `json:"touser"`
	MsgType string `json:"msgtype"`
	AgentID string `json:"agentid"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
}

func New(appid, appkey, agentid string) (*wechat, error) {
	ctx, cancel := context.WithCancel(context.Background())
	wc := &wechat{
		AppID:   appid,
		AppKey:  appkey,
		AgentID: agentid,
		ctx:     ctx,
		cancel:  cancel,
	}
	err := wc.getToken()
	if err != nil {
		return nil, err
	}
	go wc.tokenRefresher()
	return wc, nil
}

func (wc *wechat) tokenRefresher() {
	ticker := time.NewTicker(1600 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			wc.getToken()
		case <-wc.ctx.Done():
			return
		}
	}
}

func (wc *wechat) getToken() error {
	var ret result
	url := fmt.Sprintf(conGetTokenURL, wc.AppID, wc.AppKey)
	err := getJSON(url, &ret)
	if err != nil {
		return err
	}
	if ret.Code != 0 {
		return fmt.Errorf("get token failed: %s", ret.Message)
	}
	wc.mu.Lock()
	wc.AccessToken = ret.Token
	wc.mu.Unlock()
	return nil
}

func (wc *wechat) Send(to, msg string) error {
	var ret result
	wc.mu.RLock()
	url := fmt.Sprintf(conSendMsgURL, wc.AccessToken)
	wc.mu.RUnlock()
	m := message{
		ToUser:  to,
		MsgType: "text",
		AgentID: wc.AgentID,
		Text: struct {
			Content string `json:"content"`
		}{
			Content: msg,
		},
	}

	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal message error: %w", err)
	}

	reply, err := postJSON(url, string(data))
	if err != nil {
		return fmt.Errorf("post URL error: %w", err)
	}

	err = json.Unmarshal(reply, &ret)
	if err != nil {
		return fmt.Errorf("unmarshal reply error: %w", err)
	}
	if ret.Code != 0 {
		return fmt.Errorf("send message failed: %s", ret.Message)
	}
	return nil
}

func (wc *wechat) Close() {
	wc.cancel()
}

func getJSON(url string, v any) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	reply, _ := io.ReadAll(resp.Body)
	return json.Unmarshal(reply, v)
}

func postJSON(url string, params string) (reply []byte, err error) {
	resp, err := http.Post(url,
		"application/json;charset=UTF-8",
		strings.NewReader(params))
	if err != nil {
		return
	}
	defer resp.Body.Close()
	reply, err = io.ReadAll(resp.Body)
	return
}
