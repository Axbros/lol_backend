package handler

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	"github.com/smartwalle/alipay/v3"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	wechatUtils "github.com/wechatpay-apiv3/wechatpay-go/utils"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"lol/internal/cache"
	"lol/internal/config"
	"lol/internal/dao"
	"lol/internal/database"
	"lol/internal/ecode"
	"lol/internal/model"
	"lol/internal/payment"
	"lol/internal/types"

	"github.com/go-dev-frame/sponge/pkg/gin/middleware"
	"github.com/go-dev-frame/sponge/pkg/gin/response"
	"github.com/go-dev-frame/sponge/pkg/logger"
	"github.com/go-dev-frame/sponge/pkg/utils"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
)

const (
	pollingInterval    = 10 * time.Second
	maxPollingAttempts = 17 // 最大查询次数
)

var cancelMutex sync.Mutex
var cancel context.CancelFunc
var _ LoanHandler = (*loanHandler)(nil)

// LoanHandler defining the handler interface
type LoanHandler interface {
	Create(c *gin.Context)
	DeleteByID(c *gin.Context)
	UpdateByID(c *gin.Context)
	GetByID(c *gin.Context)
	List(c *gin.Context)
	GetDetail(c *gin.Context)
	Pay(c *gin.Context)
	Notify(c *gin.Context)
}

type loanHandler struct {
	iDao      dao.LoanDao
	alipay    *alipay.Client
	wechatPay *core.Client
}

// NewLoanHandler creating the handler interface
func NewLoanHandler() LoanHandler {
	return &loanHandler{
		iDao: dao.NewLoanDao(
			database.GetDB(), // db driver is mysql
			cache.NewLoanCache(database.GetCacheType()),
		),
		alipay:    payment.GetAlipayClient(),
		wechatPay: payment.GetWechatClient(),
	}
}

// Create a record
// @Summary create loan
// @Description submit information to create loan
// @Tags loan
// @accept json
// @Produce json
// @Param data body types.CreateLoanRequest true "loan information"
// @Success 200 {object} types.CreateLoanReply{}
// @Router /api/v1/loan [post]
// @Security BearerAuth
func (h *loanHandler) Create(c *gin.Context) {
	form := &types.CreateLoanRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}

	loan := &model.Loan{}

	err = copier.Copy(loan, form)
	if err != nil {
		response.Error(c, ecode.ErrCreateLoan)
		return
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	ctx := middleware.WrapCtx(c)

	// Ensure the calculation is done with floating-point numbers
	// Convert form.LoanMoney to float64
	loan.MonthlyPayment = float64(form.LoanMoney)*2/100 + float64(form.LoanMoney)/float64(form.LoanPeriod)

	loan.Status = 0

	now := time.Now()
	loan.CreateAt = &now
	err = h.iDao.Create(ctx, loan)
	if err != nil {
		logger.Error("Create error", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	response.Success(c, gin.H{"id": loan.ID})
}

// DeleteByID delete a record by id
// @Summary delete loan
// @Description delete loan by id
// @Tags loan
// @accept json
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} types.DeleteLoanByIDReply{}
// @Router /api/v1/loan/{id} [delete]
// @Security BearerAuth
func (h *loanHandler) DeleteByID(c *gin.Context) {
	_, id, isAbort := getLoanIDFromPath(c)
	if isAbort {
		response.Error(c, ecode.InvalidParams)
		return
	}

	ctx := middleware.WrapCtx(c)
	err := h.iDao.DeleteByID(ctx, id)
	if err != nil {
		logger.Error("DeleteByID error", logger.Err(err), logger.Any("id", id), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	response.Success(c)
}

// UpdateByID update information by id
// @Summary update loan
// @Description update loan information by id
// @Tags loan
// @accept json
// @Produce json
// @Param id path string true "id"
// @Param data body types.UpdateLoanByIDRequest true "loan information"
// @Success 200 {object} types.UpdateLoanByIDReply{}
// @Router /api/v1/loan/{id} [put]
// @Security BearerAuth
func (h *loanHandler) UpdateByID(c *gin.Context) {
	_, id, isAbort := getLoanIDFromPath(c)
	if isAbort {
		response.Error(c, ecode.InvalidParams)
		return
	}

	form := &types.UpdateLoanByIDRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}
	form.ID = id

	loan := &model.Loan{}
	err = copier.Copy(loan, form)
	if err != nil {
		response.Error(c, ecode.ErrUpdateByIDLoan)
		return
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	ctx := middleware.WrapCtx(c)
	err = h.iDao.UpdateByID(ctx, loan)
	if err != nil {
		logger.Error("UpdateByID error", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	response.Success(c)
}

// GetByID get a record by id
// @Summary get loan detail
// @Description get loan detail by id
// @Tags loan
// @Param id path string true "id"
// @Accept json
// @Produce json
// @Success 200 {object} types.GetLoanByIDReply{}
// @Router /api/v1/loan/{id} [get]
// @Security BearerAuth
func (h *loanHandler) GetByID(c *gin.Context) {
	_, id, isAbort := getLoanIDFromPath(c)
	if isAbort {
		response.Error(c, ecode.InvalidParams)
		return
	}

	ctx := middleware.WrapCtx(c)
	loan, err := h.iDao.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, database.ErrRecordNotFound) {
			logger.Warn("GetByID not found", logger.Err(err), logger.Any("id", id), middleware.GCtxRequestIDField(c))
			response.Error(c, ecode.NotFound)
		} else {
			logger.Error("GetByID error", logger.Err(err), logger.Any("id", id), middleware.GCtxRequestIDField(c))
			response.Output(c, ecode.InternalServerError.ToHTTPCode())
		}
		return
	}

	data := &types.LoanObjDetail{}
	err = copier.Copy(data, loan)
	if err != nil {
		response.Error(c, ecode.ErrGetByIDLoan)
		return
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	response.Success(c, gin.H{"loan": data})
}

// List of records by query parameters
// @Summary list of loans by query parameters
// @Description list of loans by paging and conditions
// @Tags loan
// @accept json
// @Produce json
// @Param data body types.Params true "query parameters"
// @Success 200 {object} types.ListLoansReply{}
// @Router /api/v1/loan/list [post]
// @Security BearerAuth
func (h *loanHandler) List(c *gin.Context) {
	form := &types.ListLoansRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}

	ctx := middleware.WrapCtx(c)
	loans, total, err := h.iDao.GetByColumns(ctx, &form.Params)
	if err != nil {
		logger.Error("GetByColumns error", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	data, err := convertLoans(loans)
	if err != nil {
		response.Error(c, ecode.ErrListLoan)
		return
	}

	response.Success(c, gin.H{
		"loans": data,
		"total": total,
	})
}

func (h *loanHandler) GetDetail(c *gin.Context) {
	form := &types.GetDetailRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}
	ctx := middleware.WrapCtx(c)
	loan, err := h.iDao.GetByMobileAndCode(ctx, form.Mobile, form.Code)
	if err != nil {
		logger.Warnf("failed to get user loan detail ,user mobile:%s,user code:%s", form.Mobile, form.Code)
		response.Error(c, ecode.ErrListLoan)
		return
	}
	response.Success(c, gin.H{
		"detail": loan,
	})
}

func (h *loanHandler) Pay(c *gin.Context) {
	form := &types.PayRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}
	ctx := middleware.WrapCtx(c)
	loan, err := h.iDao.GetByMobileAndCode(ctx, form.Mobile, form.Code)
	if err != nil {

		response.Error(c, ecode.ErrListLoan)
		return
	}
	if loan.Status == 1 {
		//表示已經還完
		response.Error(c, ecode.ErrLoanStatus)
		return
	}
	//開始調用支付寶/微信網頁支付接口
	var subject string

	baseMoney := loan.MonthlyPayment + float64(loan.OverDueMoney)

	totalMoney := baseMoney + baseMoney*(6.0/1000)
	money := fmt.Sprintf("%.2f", totalMoney)

	extInfo := ""
	if loan.OverDueMoney > 0 {
		extInfo = "（逾期费用：" + strconv.Itoa(loan.OverDueMoney) + "元）"
	}
	subject = loan.Name + "支付【" + loan.CarModel + "】月租" + money + "元" + extInfo

	var url string
	tradeNo := generateTradeNo()
	if form.Method == "alipay" {
		//支付寶
		url = WapAlipay(h.alipay, subject, money, tradeNo)
	} else {
		url = WechatNativePay(h.wechatPay, subject, totalMoney, tradeNo)
		//ctxForTracking, c := context.WithTimeout(context.Background(), 10*time.Minute)
		//cancel = c
		//go func() {
		//	trackWechatOrder(h, ctxForTracking, h.wechatPay, tradeNo)
		//	cancelMutex.Lock()
		//	if cancel != nil {
		//		cancel()
		//		cancel = nil
		//	}
		//	cancelMutex.Unlock()
		//}()
	}
	now := time.Now()
	payments := &model.PaymentHistory{
		UserPhone:  form.Mobile,
		OutTradeNo: tradeNo,
		Status:     "PAYING",
		Method:     form.Method,
		CreateAt:   &now,
	}
	err = h.iDao.CreatePaymentHistory(ctx, payments)
	if err != nil {
		response.Error(c, ecode.ErrCreatePayment)
		return
	}
	response.Success(c, gin.H{
		"url": url,
	})
}

func (h *loanHandler) Notify(c *gin.Context) {
	// 解析表单数据并检查错误
	if err := c.Request.ParseForm(); err != nil {
		log.Printf("解析表单数据失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "解析请求参数失败"})
		return
	}

	// 获取路径参数
	bandName := c.Param("bandName")
	ctx := middleware.WrapCtx(c)

	// 仅处理支付宝通知
	if bandName == "alipay" {
		// 解析支付宝通知
		result, err := h.alipay.DecodeNotification(c.Request.Form)
		if err != nil {
			log.Printf("解析支付宝异步通知参数失败: %v, 表单数据: %+v", err, c.Request.Form)
			c.String(http.StatusBadRequest, "fail") // 支付宝要求失败返回"fail"
			return
		}

		// 记录通知接收情况
		log.Printf("收到支付宝通知 - 订单号: %s, 交易状态: %s", result.OutTradeNo, result.TradeStatus)

		// 根据交易状态处理
		var status string
		switch result.TradeStatus {
		case "TRADE_SUCCESS":
			status = "SUCCESS"
			log.Printf("订单 %s 支付成功", result.OutTradeNo)
		case "TRADE_CLOSED":
			status = "CLOSED"
			log.Printf("订单 %s 已关闭", result.OutTradeNo)
		case "TRADE_FINISHED":
			log.Printf("订单 %s 交易完成", result.OutTradeNo)
			// 无需更新状态，直接返回成功
			c.String(http.StatusOK, "success")
			return
		default:
			log.Printf("订单 %s 收到未处理的支付状态: %s", result.OutTradeNo, result.TradeStatus)
			c.String(http.StatusOK, "success")
			return
		}

		// 更新订单状态
		if err := h.iDao.UpdatePaymentStatusByTradeNo(ctx, result.OutTradeNo, status); err != nil {
			log.Printf("更新订单 %s 状态失败: %v", result.OutTradeNo, err)
			c.String(http.StatusInternalServerError, "fail")
			return
		}
	} else {
		// 记录未支持的支付渠道
		log.Printf("收到未支持的支付渠道通知: %s", bandName)
	}

	if bandName == "wechat" {
		dir, err := os.Getwd()
		if err != nil {
			log.Println("Get Current Workplace Direction Error::", err)
			return
		}
		mchPrivateKeyPath := dir + config.Get().WechatPay.MchPrivateKeyPath

		mchPrivateKey, err := wechatUtils.LoadPrivateKeyWithPath(mchPrivateKeyPath)

		mchID := config.Get().WechatPay.MchID
		mchAPIv3Key := config.Get().WechatPay.MchAPIv3Key

		err = downloader.MgrInstance().RegisterDownloaderWithPrivateKey(ctx, mchPrivateKey, config.Get().WechatPay.MchCertificateSerialNumber, mchID, mchAPIv3Key)
		if err != nil {
			logger.Warn("downloader.MgrInstance().RegisterDownloaderWithPrivateKey(ctx, mchPrivateKey, config.Get().WechatPay.MchCertificateSerialNumber,  config.Get().WechatPay.MchID, config.Get().WechatPay.MchAPIv3Key) error")
			return
		}
		// 2. 获取商户号对应的微信支付平台证书访问器
		certificateVisitor := downloader.MgrInstance().GetCertificateVisitor(mchID)
		// 3. 使用证书访问器初始化 `notify.Handler`
		handler := notify.NewNotifyHandler(mchAPIv3Key, verifiers.NewSHA256WithRSAVerifier(certificateVisitor))
		transaction := new(payments.Transaction)
		notifyReq, err := handler.ParseNotifyRequest(context.Background(), c.Request, transaction)
		// 如果验签未通过，或者解密失败
		if err != nil {
			fmt.Println(err)
			return
		}
		// 处理通知内容
		if notifyReq.Summary != "支付成功" {
			logger.Warnf("微信支付失败：%s", notifyReq.Summary)
			h.iDao.UpdatePaymentStatusByTradeNo(ctx, *transaction.OutTradeNo, "FAILED")
		} else {
			logger.Warnf("微信支付成功：%s", notifyReq.Summary)
			h.iDao.UpdatePaymentStatusByTradeNo(ctx, *transaction.OutTradeNo, "SUCCESS")
		}
		logger.Infof("微信交易单号 %s 交易状态 %s", transaction.TransactionId, transaction.TradeState)
	}

	// 返回成功响应（支付宝要求成功返回"success"）
	c.String(http.StatusOK, "success")
}

// 订单查询函数
func queryWechatOrderStatus(ctx context.Context, client *core.Client, outTradeNo string) (bool, error) {
	mchid := config.Get().WechatPay.MchID
	svc := native.NativeApiService{Client: client}
	resp, result, err := svc.QueryOrderByOutTradeNo(ctx,
		native.QueryOrderByOutTradeNoRequest{
			OutTradeNo: core.String(outTradeNo),
			Mchid:      core.String(mchid),
		},
	)

	if err != nil {
		// 处理错误
		log.Printf("call QueryOrderByOutTradeNo err: %s", err)
		return false, err
	}

	if result.Response.StatusCode != 200 {
		// 非 200 状态码，认为查询异常
		log.Printf("QueryOrderByOutTradeNo unexpected status code: %d", result.Response.StatusCode)
		return false, nil
	}

	// 检查订单状态
	if resp.TradeState != nil && *resp.TradeState == "SUCCESS" {
		log.Printf("Order %s has been paid successfully", outTradeNo)
		return true, nil
	}

	log.Printf("Order %s is not paid yet", outTradeNo)
	return false, nil
}

// 订单跟踪函数
func trackWechatOrder(h *loanHandler, ctx context.Context, client *core.Client, outTradeNo string) {
	attempts := 0
	ticker := time.NewTicker(pollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():

			log.Printf("订单跟踪被取消，订单号: %s，原因: %v", outTradeNo, ctx.Err())
			CloseOrder(h.wechatPay, outTradeNo, ctx)
			err := h.iDao.UpdatePaymentStatusByTradeNo(ctx, outTradeNo, "CANCEL")
			if err != nil {
				log.Printf("更新订单状态失败: %v", err)
			}
			return
		case <-ticker.C:
			attempts++
			if attempts > maxPollingAttempts {
				log.Printf("达到最大查询次数，停止跟踪订单，订单号: %s", outTradeNo)
				err := h.iDao.UpdatePaymentStatusByTradeNo(ctx, outTradeNo, "TIME_OUT")
				if err != nil {
					log.Printf("更新订单状态失败: %v", err)
				}
				CloseOrder(h.wechatPay, outTradeNo, ctx)
				return
			}
			paid, err := queryWechatOrderStatus(ctx, client, outTradeNo)
			if err != nil {
				log.Printf("查询订单状态出错，订单号: %s, 错误信息: %v", outTradeNo, err)
				CloseOrder(h.wechatPay, outTradeNo, ctx)
				continue
			}
			if paid {
				log.Printf("订单已支付，结束跟踪，订单号: %s", outTradeNo)
				err := h.iDao.UpdatePaymentStatusByTradeNo(ctx, outTradeNo, "SUCCESS")
				if err != nil {
					log.Printf("更新订单状态失败: %v", err)
				}
				return
			}
			log.Printf("订单未支付，继续跟踪，订单号: %s，第 %d 次查询", outTradeNo, attempts)
		}
	}
}
func getLoanIDFromPath(c *gin.Context) (string, uint64, bool) {
	idStr := c.Param("id")
	id, err := utils.StrToUint64E(idStr)
	if err != nil || id == 0 {
		logger.Warn("StrToUint64E error: ", logger.String("idStr", idStr), middleware.GCtxRequestIDField(c))
		return "", 0, true
	}

	return idStr, id, false
}

func convertLoan(loan *model.Loan) (*types.LoanObjDetail, error) {
	data := &types.LoanObjDetail{}
	err := copier.Copy(data, loan)
	if err != nil {
		return nil, err
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	return data, nil
}

func convertLoans(fromValues []*model.Loan) ([]*types.LoanObjDetail, error) {
	toValues := []*types.LoanObjDetail{}
	for _, v := range fromValues {
		data, err := convertLoan(v)
		if err != nil {
			return nil, err
		}
		toValues = append(toValues, data)
	}

	return toValues, nil
}

// var alipayClient *alipay.Client

func WapAlipay(alipayClient *alipay.Client, subject string, totalAmount string, tradeNo string) string {
	notifyURL := config.Get().Alipay.NotifyURL
	returnURL := config.Get().Alipay.ReturnURL

	// 构建手机网页支付请求参数
	var p = alipay.TradeWapPay{}
	p.NotifyURL = notifyURL
	p.ReturnURL = returnURL
	p.Subject = subject
	p.OutTradeNo = tradeNo
	p.TotalAmount = totalAmount
	p.ProductCode = "QUICK_WAP_PAY"

	// 发起支付请求
	url, err := alipayClient.TradeWapPay(p)
	if err != nil {
		log.Fatalf("发起支付请求失败: %v", err)
	}

	return url.String()
}

func WechatNativePay(wechatClient *core.Client, subject string, totalAmount float64, tradeNo string) string {
	ctx := context.Background()
	appid := config.Get().WechatPay.AppID
	mchid := config.Get().WechatPay.MchID
	notifyURL := config.Get().WechatPay.NotifyURL
	totalAmountInFen := int64(totalAmount * 100)
	// 创建 Native 支付服务实例
	service := native.NativeApiService{Client: wechatClient}
	// 构建支付请求参数
	req := native.PrepayRequest{
		Appid:       core.String(appid),
		Mchid:       core.String(mchid),
		Description: core.String(subject),
		OutTradeNo:  core.String(tradeNo),
		GoodsTag:    core.String("用户偿还车款"),
		NotifyUrl:   core.String(notifyURL),
		Amount: &native.Amount{
			Total: core.Int64(totalAmountInFen), // 订单总金额，单位为分
		},
	}
	resp, _, err := service.Prepay(ctx, req)
	if err != nil {
		log.Fatalf("prepay error: %v", err)
	}

	return *resp.CodeUrl
}

// 以下情况需要调用关单接口：
// 1. 商户订单支付失败需要生成新单号重新发起支付，要对原订单号调用关单，避免重复支付；
// 2. 系统下单后，用户支付超时，系统退出不再受理，避免用户继续，请调用关单接口。
func CloseOrder(wechatClient *core.Client, outTradeNo string, ctx context.Context) bool {
	svc := native.NativeApiService{Client: wechatClient}
	mchid := config.Get().WechatPay.MchID
	result, err := svc.CloseOrder(ctx,
		native.CloseOrderRequest{
			OutTradeNo: core.String(outTradeNo),
			Mchid:      core.String(mchid),
		},
	)

	if err != nil {
		// 处理错误
		log.Printf("call CloseOrder err:%s", err)
		return false
	} else {
		// 处理返回结果
		log.Printf("status=%d", result.Response.StatusCode)
		return true
	}
}

func generateTradeNo() string {
	return time.Now().Format("20060102150405")
}
