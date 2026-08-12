package payment

import (
	"context"
	"crypto/md5" //nolint:gosec // XPay's protocol requires MD5 signatures.
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const XPaySignType = "MD5"

var ErrInvalidXPaySignature = errors.New("invalid xpay signature")

// XPaySign creates the signature required by XPay's EPay-compatible protocol.
// Empty values, sign and sign_type are excluded from the signature payload.
func XPaySign(params map[string]string, merchantKey string) string {
	keys := make([]string, 0, len(params))
	for key, value := range params {
		if value == "" || key == "sign" || key == "sign_type" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var payload strings.Builder
	for index, key := range keys {
		if index > 0 {
			payload.WriteByte('&')
		}
		payload.WriteString(key)
		payload.WriteByte('=')
		payload.WriteString(params[key])
	}
	payload.WriteString(merchantKey)

	digest := md5.Sum([]byte(payload.String())) //nolint:gosec // Required by the upstream protocol.
	return hex.EncodeToString(digest[:])
}

// VerifyXPaySignature verifies an XPay callback signature in constant time.
func VerifyXPaySignature(params map[string]string, merchantKey string) error {
	provided := strings.ToLower(params["sign"])
	if provided == "" {
		return ErrInvalidXPaySignature
	}

	expected := XPaySign(params, merchantKey)
	if len(provided) != len(expected) || subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
		return ErrInvalidXPaySignature
	}
	return nil
}

// CreateXPayMAPI posts a signed form to mapi.php and returns the payment URL
// contained in its JSON response.
func CreateXPayMAPI(ctx context.Context, mapiURL string, params map[string]string, merchantKey string) (string, error) {
	if merchantKey == "" || params["pid"] == "" {
		return "", errors.New("xpay merchant credentials are required")
	}
	endpoint, err := url.Parse(mapiURL)
	if err != nil {
		return "", err
	}
	if endpoint.Scheme == "" || endpoint.Host == "" {
		return "", errors.New("invalid xpay MAPI URL")
	}

	signedParams := make(map[string]string, len(params)+2)
	for key, value := range params {
		signedParams[key] = value
	}
	signedParams["sign"] = XPaySign(signedParams, merchantKey)
	signedParams["sign_type"] = XPaySignType

	form := url.Values{}
	for key, value := range signedParams {
		if value != "" {
			form.Set(key, value)
		}
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	requestData, err := json.Marshal(signedParams)
	if err != nil {
		return "", err
	}
	log.Printf("xpay mapi request: method=POST url=%s data=%s form=%s", endpoint.String(), string(requestData), form.Encode())

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", err
	}
	log.Printf("xpay mapi response: status=%d body=%s", resp.StatusCode, string(body))
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("xpay MAPI HTTP status=%d response=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	result, err := parseXPayMAPIResponse(body)
	if err != nil {
		return "", err
	}
	if result.Code != 1 {
		return "", fmt.Errorf("xpay MAPI failed, code=%d msg=%s", result.Code, result.Msg)
	}

	for _, paymentURL := range []string{result.PayURL, result.URLScheme, result.QRCode} {
		if paymentURL != "" {
			return validateXPayPaymentURL(paymentURL)
		}
	}
	return "", errors.New("xpay MAPI response did not contain payurl, urlscheme or qrcode")
}

type xPayMAPIResponse struct {
	Code      int
	Msg       string
	TradeNo   string
	PayURL    string
	QRCode    string
	URLScheme string
	Money     string
}

func parseXPayMAPIResponse(body []byte) (*xPayMAPIResponse, error) {
	var raw struct {
		Code      json.RawMessage `json:"code"`
		Msg       string          `json:"msg"`
		TradeNo   string          `json:"trade_no"`
		PayURL    string          `json:"payurl"`
		QRCode    string          `json:"qrcode"`
		URLScheme string          `json:"urlscheme"`
		Money     string          `json:"money"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode xpay MAPI response: %w", err)
	}

	var code int
	if err := json.Unmarshal(raw.Code, &code); err != nil {
		var codeText string
		if textErr := json.Unmarshal(raw.Code, &codeText); textErr != nil {
			return nil, errors.New("xpay MAPI response contains an invalid code")
		}
		code, err = strconv.Atoi(codeText)
		if err != nil {
			return nil, errors.New("xpay MAPI response contains an invalid code")
		}
	}

	return &xPayMAPIResponse{
		Code: code, Msg: raw.Msg, TradeNo: raw.TradeNo, PayURL: raw.PayURL,
		QRCode: raw.QRCode, URLScheme: raw.URLScheme, Money: raw.Money,
	}, nil
}

func validateXPayPaymentURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	parsed, err := url.Parse(value)
	if err != nil {
		return "", err
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https", "weixin", "alipay", "alipays":
		return value, nil
	default:
		return "", errors.New("xpay returned an invalid payment URL")
	}
}
