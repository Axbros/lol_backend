package payment

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestXPaySign(t *testing.T) {
	params := map[string]string{
		"pid":          "1000",
		"type":         "alipay",
		"out_trade_no": "20240101123456",
		"notify_url":   "http://www.example.com/notify_url.php",
		"return_url":   "http://www.example.com/return_url.php",
		"name":         "测试商品",
		"money":        "100.00",
		"empty":        "",
		"sign":         "ignored",
		"sign_type":    "MD5",
	}

	const want = "f026e82760c50c6babf1198a8ff09c83"
	if got := XPaySign(params, "your_key_here"); got != want {
		t.Fatalf("XPaySign() = %q, want %q", got, want)
	}
}

func TestCreateXPayMAPIPostsSignedFormAndReturnsPayURL(t *testing.T) {
	params := map[string]string{
		"pid":          "10414",
		"type":         "wxpay",
		"out_trade_no": "202608120001",
		"notify_url":   "https://merchant.example/api/v1/loan/xpay/notify",
		"name":         "月租 测试",
		"money":        "100.00",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
			t.Errorf("Content-Type = %q", r.Header.Get("Content-Type"))
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		received := make(map[string]string, len(r.PostForm))
		for key := range r.PostForm {
			received[key] = r.PostForm.Get(key)
		}
		if err := VerifyXPaySignature(received, "secret"); err != nil {
			t.Errorf("VerifyXPaySignature() error = %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":1,"msg":"success","trade_no":"xpay-1","payurl":"https://pay.example/order-1","money":"100.00"}`))
	}))
	defer server.Close()

	payURL, err := CreateXPayMAPI(context.Background(), server.URL+"/mapi.php", params, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if payURL != "https://pay.example/order-1" {
		t.Fatalf("CreateXPayMAPI() = %q", payURL)
	}
}

func TestCreateXPayMAPIReturnsUpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"code":-1,"msg":"没有找到可用支付账号"}`))
	}))
	defer server.Close()

	_, err := CreateXPayMAPI(context.Background(), server.URL+"/mapi.php", map[string]string{
		"pid": "10414", "type": "alipay", "out_trade_no": "order-1",
	}, "secret")
	if err == nil || !strings.Contains(err.Error(), "没有找到可用支付账号") {
		t.Fatalf("CreateXPayMAPI() error = %v", err)
	}
}

func TestCreateXPayMAPIAcceptsStringCodeAndURLScheme(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"code":"1","urlscheme":"weixin://dl/business/?ticket=test"}`))
	}))
	defer server.Close()

	got, err := CreateXPayMAPI(context.Background(), server.URL+"/mapi.php", map[string]string{
		"pid": "10414", "type": "wxpay", "out_trade_no": "order-1",
	}, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if got != "weixin://dl/business/?ticket=test" {
		t.Fatalf("CreateXPayMAPI() = %q", got)
	}
}

func TestVerifyXPaySignatureRejectsModifiedParameters(t *testing.T) {
	params := map[string]string{"pid": "10414", "money": "100.00"}
	params["sign"] = XPaySign(params, "secret")
	params["money"] = "0.01"
	if err := VerifyXPaySignature(params, "secret"); err == nil {
		t.Fatal("VerifyXPaySignature() accepted modified parameters")
	}
}
