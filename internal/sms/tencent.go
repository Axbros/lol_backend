package sms

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tencentsms "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"

	"lol/internal/config"
)

type Sender interface {
	Send(ctx context.Context, mobile string, username string, code string) error
}

type TencentSender struct {
	client *tencentsms.Client
	cfg    config.SMS
}

func NewTencentSender(cfg config.SMS) (*TencentSender, error) {
	if cfg.SecretID == "" || cfg.SecretKey == "" || cfg.SDKAppID == "" || cfg.SignName == "" || cfg.TemplateID == "" {
		return nil, errors.New("tencent SMS configuration is incomplete")
	}
	credential := common.NewCredential(cfg.SecretID, cfg.SecretKey)
	httpProfile := profile.NewHttpProfile()
	httpProfile.ReqMethod = "POST"
	httpProfile.ReqTimeout = 30
	if cfg.Endpoint != "" {
		httpProfile.Endpoint = cfg.Endpoint
	}
	clientProfile := profile.NewClientProfile()
	clientProfile.SignMethod = "TC3-HMAC-SHA256"
	clientProfile.Language = "en-US"
	clientProfile.HttpProfile = httpProfile

	client, err := tencentsms.NewClient(credential, cfg.Region, clientProfile)
	if err != nil {
		return nil, err
	}
	return &TencentSender{client: client, cfg: cfg}, nil
}

func (s *TencentSender) Send(ctx context.Context, mobile string, username string, code string) error {
	request := tencentsms.NewSendSmsRequest()
	request.PhoneNumberSet = common.StringPtrs([]string{FormatMobile(mobile, s.cfg.CountryCode)})
	request.SmsSdkAppId = common.StringPtr(s.cfg.SDKAppID)
	request.SignName = common.StringPtr(s.cfg.SignName)
	request.TemplateId = common.StringPtr(s.cfg.TemplateID)
	request.TemplateParamSet = common.StringPtrs([]string{username, code})
	request.SessionContext = common.StringPtr("")
	request.ExtendCode = common.StringPtr("")
	request.SenderId = common.StringPtr("")

	response, err := s.client.SendSmsWithContext(ctx, request)
	if err != nil {
		return err
	}
	if response == nil || response.Response == nil || len(response.Response.SendStatusSet) == 0 {
		return errors.New("tencent SMS returned an empty response")
	}
	status := response.Response.SendStatusSet[0]
	if status == nil || status.Code == nil || *status.Code != "Ok" {
		statusCode, message := "", ""
		if status != nil {
			if status.Code != nil {
				statusCode = *status.Code
			}
			if status.Message != nil {
				message = *status.Message
			}
		}
		return fmt.Errorf("tencent SMS rejected request: code=%s message=%s", statusCode, message)
	}
	return nil
}

func FormatMobile(mobile string, countryCode string) string {
	mobile = strings.TrimSpace(mobile)
	if strings.HasPrefix(mobile, "+") || strings.HasPrefix(mobile, "00") {
		return mobile
	}
	countryCode = strings.TrimSpace(countryCode)
	if countryCode == "" {
		countryCode = "+86"
	}
	return strings.TrimRight(countryCode, " ") + mobile
}
