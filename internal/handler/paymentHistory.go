package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"

	"github.com/go-dev-frame/sponge/pkg/gin/middleware"
	"github.com/go-dev-frame/sponge/pkg/gin/response"
	"github.com/go-dev-frame/sponge/pkg/logger"
	"github.com/go-dev-frame/sponge/pkg/utils"

	"lol/internal/cache"
	"lol/internal/dao"
	"lol/internal/database"
	"lol/internal/ecode"
	"lol/internal/model"
	"lol/internal/types"
)

var _ PaymentHistoryHandler = (*paymentHistoryHandler)(nil)

// PaymentHistoryHandler defining the handler interface
type PaymentHistoryHandler interface {
	Create(c *gin.Context)
	DeleteByID(c *gin.Context)
	UpdateByID(c *gin.Context)
	GetByID(c *gin.Context)
	List(c *gin.Context)
}

type paymentHistoryHandler struct {
	iDao dao.PaymentHistoryDao
}

// NewPaymentHistoryHandler creating the handler interface
func NewPaymentHistoryHandler() PaymentHistoryHandler {
	return &paymentHistoryHandler{
		iDao: dao.NewPaymentHistoryDao(
			database.GetDB(), // db driver is mysql
			cache.NewPaymentHistoryCache(database.GetCacheType()),
		),
	}
}

// Create a record
// @Summary create paymentHistory
// @Description submit information to create paymentHistory
// @Tags paymentHistory
// @accept json
// @Produce json
// @Param data body types.CreatePaymentHistoryRequest true "paymentHistory information"
// @Success 200 {object} types.CreatePaymentHistoryReply{}
// @Router /api/v1/paymentHistory [post]
// @Security BearerAuth
func (h *paymentHistoryHandler) Create(c *gin.Context) {
	form := &types.CreatePaymentHistoryRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}

	paymentHistory := &model.PaymentHistory{}
	err = copier.Copy(paymentHistory, form)
	if err != nil {
		response.Error(c, ecode.ErrCreatePaymentHistory)
		return
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	ctx := middleware.WrapCtx(c)
	err = h.iDao.Create(ctx, paymentHistory)
	if err != nil {
		logger.Error("Create error", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	response.Success(c, gin.H{"id": paymentHistory.ID})
}

// DeleteByID delete a record by id
// @Summary delete paymentHistory
// @Description delete paymentHistory by id
// @Tags paymentHistory
// @accept json
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} types.DeletePaymentHistoryByIDReply{}
// @Router /api/v1/paymentHistory/{id} [delete]
// @Security BearerAuth
func (h *paymentHistoryHandler) DeleteByID(c *gin.Context) {
	_, id, isAbort := getPaymentHistoryIDFromPath(c)
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
// @Summary update paymentHistory
// @Description update paymentHistory information by id
// @Tags paymentHistory
// @accept json
// @Produce json
// @Param id path string true "id"
// @Param data body types.UpdatePaymentHistoryByIDRequest true "paymentHistory information"
// @Success 200 {object} types.UpdatePaymentHistoryByIDReply{}
// @Router /api/v1/paymentHistory/{id} [put]
// @Security BearerAuth
func (h *paymentHistoryHandler) UpdateByID(c *gin.Context) {
	_, id, isAbort := getPaymentHistoryIDFromPath(c)
	if isAbort {
		response.Error(c, ecode.InvalidParams)
		return
	}

	form := &types.UpdatePaymentHistoryByIDRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}
	form.ID = id

	paymentHistory := &model.PaymentHistory{}
	err = copier.Copy(paymentHistory, form)
	if err != nil {
		response.Error(c, ecode.ErrUpdateByIDPaymentHistory)
		return
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	ctx := middleware.WrapCtx(c)
	err = h.iDao.UpdateByID(ctx, paymentHistory)
	if err != nil {
		logger.Error("UpdateByID error", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	response.Success(c)
}

// GetByID get a record by id
// @Summary get paymentHistory detail
// @Description get paymentHistory detail by id
// @Tags paymentHistory
// @Param id path string true "id"
// @Accept json
// @Produce json
// @Success 200 {object} types.GetPaymentHistoryByIDReply{}
// @Router /api/v1/paymentHistory/{id} [get]
// @Security BearerAuth
func (h *paymentHistoryHandler) GetByID(c *gin.Context) {
	_, id, isAbort := getPaymentHistoryIDFromPath(c)
	if isAbort {
		response.Error(c, ecode.InvalidParams)
		return
	}

	ctx := middleware.WrapCtx(c)
	paymentHistory, err := h.iDao.GetByID(ctx, id)
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

	data := &types.PaymentHistoryObjDetail{}
	err = copier.Copy(data, paymentHistory)
	if err != nil {
		response.Error(c, ecode.ErrGetByIDPaymentHistory)
		return
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	response.Success(c, gin.H{"paymentHistory": data})
}

// List of records by query parameters
// @Summary list of paymentHistorys by query parameters
// @Description list of paymentHistorys by paging and conditions
// @Tags paymentHistory
// @accept json
// @Produce json
// @Param data body types.Params true "query parameters"
// @Success 200 {object} types.ListPaymentHistorysReply{}
// @Router /api/v1/paymentHistory/list [post]
// @Security BearerAuth
func (h *paymentHistoryHandler) List(c *gin.Context) {
	form := &types.ListPaymentHistorysRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}

	ctx := middleware.WrapCtx(c)
	paymentHistorys, total, err := h.iDao.GetByColumns(ctx, &form.Params)
	if err != nil {
		logger.Error("GetByColumns error", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	data, err := convertPaymentHistorys(paymentHistorys)
	if err != nil {
		response.Error(c, ecode.ErrListPaymentHistory)
		return
	}

	response.Success(c, gin.H{
		"paymentHistorys": data,
		"total":           total,
	})
}

func getPaymentHistoryIDFromPath(c *gin.Context) (string, uint64, bool) {
	idStr := c.Param("id")
	id, err := utils.StrToUint64E(idStr)
	if err != nil || id == 0 {
		logger.Warn("StrToUint64E error: ", logger.String("idStr", idStr), middleware.GCtxRequestIDField(c))
		return "", 0, true
	}

	return idStr, id, false
}

func convertPaymentHistory(paymentHistory *model.PaymentHistory) (*types.PaymentHistoryObjDetail, error) {
	data := &types.PaymentHistoryObjDetail{}
	err := copier.Copy(data, paymentHistory)
	if err != nil {
		return nil, err
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	return data, nil
}

func convertPaymentHistorys(fromValues []*model.PaymentHistory) ([]*types.PaymentHistoryObjDetail, error) {
	toValues := []*types.PaymentHistoryObjDetail{}
	for _, v := range fromValues {
		data, err := convertPaymentHistory(v)
		if err != nil {
			return nil, err
		}
		toValues = append(toValues, data)
	}

	return toValues, nil
}
