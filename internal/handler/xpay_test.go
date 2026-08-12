package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"

	"lol/internal/config"
	"lol/internal/dao"
	"lol/internal/model"
	"lol/internal/payment"
)

type xPayCallbackDAO struct {
	dao.LoanDao
	history       *model.PaymentHistory
	updatedTrade  string
	updatedStatus string
}

func (d *xPayCallbackDAO) GetPaymentHistoryByTradeNo(_ context.Context, _ string) (*model.PaymentHistory, error) {
	return d.history, nil
}

func (d *xPayCallbackDAO) UpdatePaymentStatusByTradeNo(_ context.Context, tradeNo string, status string) error {
	d.updatedTrade = tradeNo
	d.updatedStatus = status
	return nil
}

func TestXPayNotify(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const merchantKey = "test-secret"
	config.Set(&config.Config{XPay: config.XPay{MerchantID: "10414", MerchantKey: merchantKey}})

	params := map[string]string{
		"pid":          "10414",
		"trade_no":     "xpay-order-1",
		"out_trade_no": "merchant-order-1",
		"type":         "alipay",
		"name":         "测试订单",
		"money":        "100.00",
		"trade_status": "TRADE_SUCCESS",
		"sign_type":    payment.XPaySignType,
	}
	params["sign"] = payment.XPaySign(params, merchantKey)
	query := url.Values{}
	for key, value := range params {
		query.Set(key, value)
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/loan/xpay/notify?"+query.Encode(), nil)
	fakeDAO := &xPayCallbackDAO{history: &model.PaymentHistory{
		OutTradeNo: "merchant-order-1",
		TotalMoney: 100,
		Method:     "xpay_alipay",
	}}
	h := &loanHandler{iDao: fakeDAO}

	h.XPayNotify(c)

	if recorder.Code != http.StatusOK || recorder.Body.String() != "success" {
		t.Fatalf("callback response = (%d, %q), want (200, success)", recorder.Code, recorder.Body.String())
	}
	if fakeDAO.updatedTrade != "merchant-order-1" || fakeDAO.updatedStatus != "SUCCESS" {
		t.Fatalf("updated order = (%q, %q)", fakeDAO.updatedTrade, fakeDAO.updatedStatus)
	}
}

func TestBuildXPayNotifyURL(t *testing.T) {
	got, err := buildXPayNotifyURL("https://merchant.example/")
	if err != nil {
		t.Fatal(err)
	}
	const want = "https://merchant.example/api/v1/loan/xpay/notify"
	if got != want {
		t.Fatalf("buildXPayNotifyURL() = %q, want %q", got, want)
	}
}

func TestPaymentMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.Set(&config.Config{Payment: config.Payment{
		Alipay: config.PaymentChannel{Enabled: true, Provider: "xpay"},
		Wechat: config.PaymentChannel{Enabled: false, Provider: "wechatPay"},
	}})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	(&loanHandler{}).PaymentMethods(c)

	var result struct {
		Code int `json:"code"`
		Data struct {
			Alipay bool `json:"alipay"`
			Wechat bool `json:"wechat"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Code != 0 || !result.Data.Alipay || result.Data.Wechat {
		t.Fatalf("unexpected payment methods response: %+v", result)
	}
}

func TestDisabledPaymentMethodIsRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.Set(&config.Config{Payment: config.Payment{
		Alipay: config.PaymentChannel{Enabled: false, Provider: "xpay"},
	}})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	(&loanHandler{}).XPayAlipay(c)

	if recorder.Code != http.StatusOK || recorder.Body.String() == "" {
		t.Fatalf("unexpected disabled payment response: status=%d body=%q", recorder.Code, recorder.Body.String())
	}
}

func TestPaymentChannelProviderSelection(t *testing.T) {
	config.Set(&config.Config{Payment: config.Payment{
		Alipay: config.PaymentChannel{Enabled: true, Provider: "xpay"},
		Wechat: config.PaymentChannel{Enabled: true, Provider: "wechatPay"},
	}})

	alipay, payType, _, directProvider, ok := paymentChannel("alipay")
	if !ok || alipay.Provider != "xpay" || payType != "alipay" || directProvider != "alipay" {
		t.Fatalf("unexpected alipay selection: channel=%+v payType=%q direct=%q ok=%v", alipay, payType, directProvider, ok)
	}
	wechat, payType, _, directProvider, ok := paymentChannel("wechat")
	if !ok || wechat.Provider != "wechatPay" || payType != "wxpay" || directProvider != "wechatpay" {
		t.Fatalf("unexpected wechat selection: channel=%+v payType=%q direct=%q ok=%v", wechat, payType, directProvider, ok)
	}
}
