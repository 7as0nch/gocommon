package identityprovider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type QQOAuthHTTPAuthorizer struct {
	appID       string
	appKey      string
	redirectURI string
	httpClient  *http.Client
}

func NewQQOAuthHTTPAuthorizer(appID, appKey, redirectURI string) *QQOAuthHTTPAuthorizer {
	return &QQOAuthHTTPAuthorizer{
		appID:       appID,
		appKey:      appKey,
		redirectURI: redirectURI,
		httpClient:  &http.Client{Timeout: 8 * time.Second},
	}
}

func (a *QQOAuthHTTPAuthorizer) Authorize(ctx context.Context, code string) (*QQAuthResult, error) {
	if strings.TrimSpace(a.appID) == "" || strings.TrimSpace(a.appKey) == "" || strings.TrimSpace(a.redirectURI) == "" {
		return nil, ErrProviderNotConfigured
	}
	accessToken, err := a.exchangeQQToken(ctx, code)
	if err != nil {
		return nil, err
	}
	openID, err := a.fetchQQOpenID(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	return &QQAuthResult{OpenID: openID}, nil
}

func (a *QQOAuthHTTPAuthorizer) exchangeQQToken(ctx context.Context, code string) (string, error) {
	u := "https://graph.qq.com/oauth2.0/token?" + url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {a.appID},
		"client_secret": {a.appKey},
		"code":          {code},
		"redirect_uri":  {a.redirectURI},
	}.Encode()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(values.Get("access_token"))
	if token == "" {
		return "", fmt.Errorf("qq access_token为空: %s", string(body))
	}
	return token, nil
}

func (a *QQOAuthHTTPAuthorizer) fetchQQOpenID(ctx context.Context, accessToken string) (string, error) {
	u := "https://graph.qq.com/oauth2.0/me?access_token=" + url.QueryEscape(accessToken)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	raw := string(body)
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return "", errors.New("qq openid返回格式异常")
	}
	var parsed struct {
		OpenID string `json:"openid"`
	}
	if err := json.Unmarshal([]byte(raw[start:end+1]), &parsed); err != nil {
		return "", err
	}
	openID := strings.TrimSpace(parsed.OpenID)
	if openID == "" {
		return "", errors.New("qq openid为空")
	}
	return openID, nil
}

type WechatOAuthHTTPAuthorizer struct {
	appID      string
	appKey     string
	httpClient *http.Client
}

func NewWechatOAuthHTTPAuthorizer(appID, appKey string) *WechatOAuthHTTPAuthorizer {
	return &WechatOAuthHTTPAuthorizer{
		appID:      appID,
		appKey:     appKey,
		httpClient: &http.Client{Timeout: 8 * time.Second},
	}
}

func (a *WechatOAuthHTTPAuthorizer) Authorize(ctx context.Context, code string) (*WechatAuthResult, error) {
	if strings.TrimSpace(a.appID) == "" || strings.TrimSpace(a.appKey) == "" {
		return nil, ErrProviderNotConfigured
	}
	u := "https://api.weixin.qq.com/sns/oauth2/access_token?" + url.Values{
		"appid":      {a.appID},
		"secret":     {a.appKey},
		"code":       {code},
		"grant_type": {"authorization_code"},
	}.Encode()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var parsed struct {
		OpenID  string `json:"openid"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if parsed.ErrCode != 0 {
		return nil, fmt.Errorf("wx oauth error: %d %s", parsed.ErrCode, parsed.ErrMsg)
	}
	openID := strings.TrimSpace(parsed.OpenID)
	if openID == "" {
		return nil, errors.New("wx openid为空")
	}
	return &WechatAuthResult{OpenID: openID}, nil
}
