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

var _ SmsHistoryHandler = (*smsHistoryHandler)(nil)

// SmsHistoryHandler defining the handler interface
type SmsHistoryHandler interface {
	Create(c *gin.Context)
	DeleteByID(c *gin.Context)
	UpdateByID(c *gin.Context)
	GetByID(c *gin.Context)
	List(c *gin.Context)
}

type smsHistoryHandler struct {
	iDao dao.SmsHistoryDao
}

// NewSmsHistoryHandler creating the handler interface
func NewSmsHistoryHandler() SmsHistoryHandler {
	return &smsHistoryHandler{
		iDao: dao.NewSmsHistoryDao(
			database.GetDB(), // db driver is mysql
			cache.NewSmsHistoryCache(database.GetCacheType()),
		),
	}
}

// Create a record
// @Summary create smsHistory
// @Description submit information to create smsHistory
// @Tags smsHistory
// @accept json
// @Produce json
// @Param data body types.CreateSmsHistoryRequest true "smsHistory information"
// @Success 200 {object} types.CreateSmsHistoryReply{}
// @Router /api/v1/smsHistory [post]
// @Security BearerAuth
func (h *smsHistoryHandler) Create(c *gin.Context) {
	form := &types.CreateSmsHistoryRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}

	smsHistory := &model.SmsHistory{}
	err = copier.Copy(smsHistory, form)
	if err != nil {
		response.Error(c, ecode.ErrCreateSmsHistory)
		return
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	ctx := middleware.WrapCtx(c)
	err = h.iDao.Create(ctx, smsHistory)
	if err != nil {
		logger.Error("Create error", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	response.Success(c, gin.H{"id": smsHistory.ID})
}

// DeleteByID delete a record by id
// @Summary delete smsHistory
// @Description delete smsHistory by id
// @Tags smsHistory
// @accept json
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} types.DeleteSmsHistoryByIDReply{}
// @Router /api/v1/smsHistory/{id} [delete]
// @Security BearerAuth
func (h *smsHistoryHandler) DeleteByID(c *gin.Context) {
	_, id, isAbort := getSmsHistoryIDFromPath(c)
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
// @Summary update smsHistory
// @Description update smsHistory information by id
// @Tags smsHistory
// @accept json
// @Produce json
// @Param id path string true "id"
// @Param data body types.UpdateSmsHistoryByIDRequest true "smsHistory information"
// @Success 200 {object} types.UpdateSmsHistoryByIDReply{}
// @Router /api/v1/smsHistory/{id} [put]
// @Security BearerAuth
func (h *smsHistoryHandler) UpdateByID(c *gin.Context) {
	_, id, isAbort := getSmsHistoryIDFromPath(c)
	if isAbort {
		response.Error(c, ecode.InvalidParams)
		return
	}

	form := &types.UpdateSmsHistoryByIDRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}
	form.ID = id

	smsHistory := &model.SmsHistory{}
	err = copier.Copy(smsHistory, form)
	if err != nil {
		response.Error(c, ecode.ErrUpdateByIDSmsHistory)
		return
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	ctx := middleware.WrapCtx(c)
	err = h.iDao.UpdateByID(ctx, smsHistory)
	if err != nil {
		logger.Error("UpdateByID error", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	response.Success(c)
}

// GetByID get a record by id
// @Summary get smsHistory detail
// @Description get smsHistory detail by id
// @Tags smsHistory
// @Param id path string true "id"
// @Accept json
// @Produce json
// @Success 200 {object} types.GetSmsHistoryByIDReply{}
// @Router /api/v1/smsHistory/{id} [get]
// @Security BearerAuth
func (h *smsHistoryHandler) GetByID(c *gin.Context) {
	_, id, isAbort := getSmsHistoryIDFromPath(c)
	if isAbort {
		response.Error(c, ecode.InvalidParams)
		return
	}

	ctx := middleware.WrapCtx(c)
	smsHistory, err := h.iDao.GetByID(ctx, id)
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

	data := &types.SmsHistoryObjDetail{}
	err = copier.Copy(data, smsHistory)
	if err != nil {
		response.Error(c, ecode.ErrGetByIDSmsHistory)
		return
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	response.Success(c, gin.H{"smsHistory": data})
}

// List of records by query parameters
// @Summary list of smsHistorys by query parameters
// @Description list of smsHistorys by paging and conditions
// @Tags smsHistory
// @accept json
// @Produce json
// @Param data body types.Params true "query parameters"
// @Success 200 {object} types.ListSmsHistorysReply{}
// @Router /api/v1/smsHistory/list [post]
// @Security BearerAuth
func (h *smsHistoryHandler) List(c *gin.Context) {
	form := &types.ListSmsHistorysRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}

	ctx := middleware.WrapCtx(c)
	smsHistorys, total, err := h.iDao.GetByColumns(ctx, &form.Params)
	if err != nil {
		logger.Error("GetByColumns error", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	data, err := convertSmsHistorys(smsHistorys)
	if err != nil {
		response.Error(c, ecode.ErrListSmsHistory)
		return
	}

	response.Success(c, gin.H{
		"smsHistorys": data,
		"total":       total,
	})
}

func getSmsHistoryIDFromPath(c *gin.Context) (string, uint64, bool) {
	idStr := c.Param("id")
	id, err := utils.StrToUint64E(idStr)
	if err != nil || id == 0 {
		logger.Warn("StrToUint64E error: ", logger.String("idStr", idStr), middleware.GCtxRequestIDField(c))
		return "", 0, true
	}

	return idStr, id, false
}

func convertSmsHistory(smsHistory *model.SmsHistory) (*types.SmsHistoryObjDetail, error) {
	data := &types.SmsHistoryObjDetail{}
	err := copier.Copy(data, smsHistory)
	if err != nil {
		return nil, err
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	return data, nil
}

func convertSmsHistorys(fromValues []*model.SmsHistory) ([]*types.SmsHistoryObjDetail, error) {
	toValues := []*types.SmsHistoryObjDetail{}
	for _, v := range fromValues {
		data, err := convertSmsHistory(v)
		if err != nil {
			return nil, err
		}
		toValues = append(toValues, data)
	}

	return toValues, nil
}
