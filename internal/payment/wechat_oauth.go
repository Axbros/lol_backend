package payment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const wechatOAuthAccessTokenURL = "https://api.weixin.qq.com/sns/oauth2/access_token"

type wechatOAuthResponse struct {
	OpenID     string `json:"openid"`
	ErrCode    int    `json:"errcode"`
	ErrMessage string `json:"errmsg"`
}

// ExchangeWechatOAuthCode 使用公众号网页授权 code 换取 JSAPI 下单所需的 openid。
func ExchangeWechatOAuthCode(ctx context.Context, appID, appSecret, code string) (string, error) {
	if strings.TrimSpace(appID) == "" || strings.TrimSpace(appSecret) == "" || strings.TrimSpace(code) == "" {
		return "", errors.New("wechat oauth appID, appSecret and code are required")
	}

	u, err := url.Parse(wechatOAuthAccessTokenURL)
	if err != nil {
		return "", err
	}
	query := u.Query()
	query.Set("appid", appID)
	query.Set("secret", appSecret)
	query.Set("code", code)
	query.Set("grant_type", "authorization_code")
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("wechat oauth returned status %d", resp.StatusCode)
	}
	result := &wechatOAuthResponse{}
	if err = json.NewDecoder(resp.Body).Decode(result); err != nil {
		return "", err
	}
	if result.ErrCode != 0 {
		return "", fmt.Errorf("wechat oauth failed: errcode=%d errmsg=%s", result.ErrCode, result.ErrMessage)
	}
	if result.OpenID == "" {
		return "", errors.New("wechat oauth response did not contain openid")
	}
	return result.OpenID, nil
}
