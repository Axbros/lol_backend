package payment

import (
	"log"
	"lol/internal/config"
	"sync"

	"github.com/smartwalle/alipay/v3"
)

var (
	alipayClient *alipay.Client
	gAlipayOnce  sync.Once
)

//TODO wechatClient *wechat.Client

func InitPayment() {
	appID := config.Get().Alipay.AppID
	privateKey := config.Get().Alipay.PrivateKey
	publicKey := config.Get().Alipay.PublicKey
	isProd := config.Get().Alipay.IsProd

	var err error
	alipayClient, err = alipay.New(appID, privateKey, isProd)
	if err != nil {
		log.Fatalf("初始化支付宝客户端失败: %v", err)
	}

	// 设置支付宝公钥
	err = alipayClient.LoadAliPayPublicKey(publicKey)
	if err != nil {
		log.Fatalf("加载支付宝公钥失败: %v", err)
	}
}

func GetAlipayClient() *alipay.Client {
	if alipayClient == nil {
		gAlipayOnce.Do(func() {
			InitPayment()
		})
	}
	return alipayClient
}
