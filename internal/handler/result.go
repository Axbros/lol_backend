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

var _ ResultHandler = (*resultHandler)(nil)

// ResultHandler defining the handler interface
type ResultHandler interface {
	Create(c *gin.Context)
	DeleteByID(c *gin.Context)
	UpdateByID(c *gin.Context)
	GetByID(c *gin.Context)
	List(c *gin.Context)
}

type resultHandler struct {
	iDao dao.ResultDao
}

// NewResultHandler creating the handler interface
func NewResultHandler() ResultHandler {
	return &resultHandler{
		iDao: dao.NewResultDao(
			database.GetDB(), // db driver is mysql
			cache.NewResultCache(database.GetCacheType()),
		),
	}
}

// Create a record
// @Summary create result
// @Description submit information to create result
// @Tags result
// @accept json
// @Produce json
// @Param data body types.CreateResultRequest true "result information"
// @Success 200 {object} types.CreateResultReply{}
// @Router /api/v1/result [post]
// @Security BearerAuth
func (h *resultHandler) Create(c *gin.Context) {
	form := &types.CreateResultRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}

	result := &model.Result{}
	err = copier.Copy(result, form)
	if err != nil {
		response.Error(c, ecode.ErrCreateResult)
		return
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	ctx := middleware.WrapCtx(c)
	err = h.iDao.Create(ctx, result)
	if err != nil {
		logger.Error("Create error", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	response.Success(c, gin.H{"id": result.ID})
}

// DeleteByID delete a record by id
// @Summary delete result
// @Description delete result by id
// @Tags result
// @accept json
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} types.DeleteResultByIDReply{}
// @Router /api/v1/result/{id} [delete]
// @Security BearerAuth
func (h *resultHandler) DeleteByID(c *gin.Context) {
	_, id, isAbort := getResultIDFromPath(c)
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
// @Summary update result
// @Description update result information by id
// @Tags result
// @accept json
// @Produce json
// @Param id path string true "id"
// @Param data body types.UpdateResultByIDRequest true "result information"
// @Success 200 {object} types.UpdateResultByIDReply{}
// @Router /api/v1/result/{id} [put]
// @Security BearerAuth
func (h *resultHandler) UpdateByID(c *gin.Context) {
	_, id, isAbort := getResultIDFromPath(c)
	if isAbort {
		response.Error(c, ecode.InvalidParams)
		return
	}

	form := &types.UpdateResultByIDRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}
	form.ID = id

	result := &model.Result{}
	err = copier.Copy(result, form)
	if err != nil {
		response.Error(c, ecode.ErrUpdateByIDResult)
		return
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	ctx := middleware.WrapCtx(c)
	err = h.iDao.UpdateByID(ctx, result)
	if err != nil {
		logger.Error("UpdateByID error", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	response.Success(c)
}

// GetByID get a record by id
// @Summary get result detail
// @Description get result detail by id
// @Tags result
// @Param id path string true "id"
// @Accept json
// @Produce json
// @Success 200 {object} types.GetResultByIDReply{}
// @Router /api/v1/result/{id} [get]
// @Security BearerAuth
func (h *resultHandler) GetByID(c *gin.Context) {
	_, id, isAbort := getResultIDFromPath(c)
	if isAbort {
		response.Error(c, ecode.InvalidParams)
		return
	}

	ctx := middleware.WrapCtx(c)
	result, err := h.iDao.GetByID(ctx, id)
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

	data := &types.ResultObjDetail{}
	err = copier.Copy(data, result)
	if err != nil {
		response.Error(c, ecode.ErrGetByIDResult)
		return
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	response.Success(c, gin.H{"result": data})
}

// List of records by query parameters
// @Summary list of results by query parameters
// @Description list of results by paging and conditions
// @Tags result
// @accept json
// @Produce json
// @Param data body types.Params true "query parameters"
// @Success 200 {object} types.ListResultsReply{}
// @Router /api/v1/result/list [post]
// @Security BearerAuth
func (h *resultHandler) List(c *gin.Context) {
	form := &types.ListResultsRequest{}
	err := c.ShouldBindJSON(form)
	if err != nil {
		logger.Warn("ShouldBindJSON error: ", logger.Err(err), middleware.GCtxRequestIDField(c))
		response.Error(c, ecode.InvalidParams)
		return
	}

	ctx := middleware.WrapCtx(c)
	results, total, err := h.iDao.GetByColumns(ctx, &form.Params)
	if err != nil {
		logger.Error("GetByColumns error", logger.Err(err), logger.Any("form", form), middleware.GCtxRequestIDField(c))
		response.Output(c, ecode.InternalServerError.ToHTTPCode())
		return
	}

	data, err := convertResults(results)
	if err != nil {
		response.Error(c, ecode.ErrListResult)
		return
	}

	response.Success(c, gin.H{
		"results": data,
		"total":   total,
	})
}

func getResultIDFromPath(c *gin.Context) (string, uint64, bool) {
	idStr := c.Param("id")
	id, err := utils.StrToUint64E(idStr)
	if err != nil || id == 0 {
		logger.Warn("StrToUint64E error: ", logger.String("idStr", idStr), middleware.GCtxRequestIDField(c))
		return "", 0, true
	}

	return idStr, id, false
}

func convertResult(result *model.Result) (*types.ResultObjDetail, error) {
	data := &types.ResultObjDetail{}
	err := copier.Copy(data, result)
	if err != nil {
		return nil, err
	}
	// Note: if copier.Copy cannot assign a value to a field, add it here

	return data, nil
}

func convertResults(fromValues []*model.Result) ([]*types.ResultObjDetail, error) {
	toValues := []*types.ResultObjDetail{}
	for _, v := range fromValues {
		data, err := convertResult(v)
		if err != nil {
			return nil, err
		}
		toValues = append(toValues, data)
	}

	return toValues, nil
}
