package payment

import (
	"context"
	"log"
	"lol/internal/config"
	"sync"

	"github.com/smartwalle/alipay/v3"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
)

var (
	alipayClient  *alipay.Client
	wechatClient  *core.Client
	gAlipayOnce   sync.Once
	gWchatPayOnce sync.Once
)

//TODO wechatClient *wechat.Client

func InitAliPayment() {
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

func InitWechatPayment() {
	mchID := config.Get().WechatPay.MchID
	mchCertificateSerialNumber := config.Get().WechatPay.MchCertificateSerialNumber
	mchAPIv3Key := config.Get().WechatPay.MchAPIv3Key
	mchPrivateKeyPath := config.Get().WechatPay.MchPrivateKeyPath

	// 读取商户私钥文件
	mchPrivateKey, err := utils.LoadPrivateKeyWithPath(mchPrivateKeyPath)
	if err != nil {
		log.Print("load merchant private key error")
	}
	ctx := context.Background()
	// 使用商户私钥等初始化 client，并使它具有自动定时获取微信支付平台证书的能力
	opts := []core.ClientOption{
		// 按照正确的参数顺序和类型传递
		option.WithWechatPayAutoAuthCipher(mchID, mchCertificateSerialNumber, mchPrivateKey, mchAPIv3Key),
	}
	wechatClient, err = core.NewClient(ctx, opts...)
	if err != nil {
		log.Fatalf("new wechat pay client err: %s", err)
	}
}

func GetAlipayClient() *alipay.Client {
	if alipayClient == nil {
		gAlipayOnce.Do(func() {
			InitAliPayment()
		})
	}
	return alipayClient
}

func GetWechatClient() *core.Client {
	if wechatClient == nil {
		gWchatPayOnce.Do(func() {
			InitWechatPayment()
		})
	}
	return wechatClient
}
