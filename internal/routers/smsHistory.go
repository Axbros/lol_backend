package routers

import (
	"github.com/gin-gonic/gin"

	"lol/internal/handler"
)

func init() {
	apiV1RouterFns = append(apiV1RouterFns, func(group *gin.RouterGroup) {
		smsHistoryRouter(group, handler.NewSmsHistoryHandler())
	})
}

func smsHistoryRouter(group *gin.RouterGroup, h handler.SmsHistoryHandler) {
	g := group.Group("/smsHistory")

	// All the following routes use jwt authentication, you also can use middleware.Auth(middleware.WithVerify(fn))
	//g.Use(middleware.Auth())

	// If jwt authentication is not required for all routes, authentication middleware can be added
	// separately for only certain routes. In this case, g.Use(middleware.Auth()) above should not be used.

	g.POST("/", h.Create)          // [post] /api/v1/smsHistory
	g.DELETE("/:id", h.DeleteByID) // [delete] /api/v1/smsHistory/:id
	g.PUT("/:id", h.UpdateByID)    // [put] /api/v1/smsHistory/:id
	g.GET("/:id", h.GetByID)       // [get] /api/v1/smsHistory/:id
	g.POST("/list", h.List)        // [post] /api/v1/smsHistory/list
}
