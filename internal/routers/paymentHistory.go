package routers

import (
	"github.com/gin-gonic/gin"

	"lol/internal/handler"
)

func init() {
	apiV1RouterFns = append(apiV1RouterFns, func(group *gin.RouterGroup) {
		paymentHistoryRouter(group, handler.NewPaymentHistoryHandler())
	})
}

func paymentHistoryRouter(group *gin.RouterGroup, h handler.PaymentHistoryHandler) {
	g := group.Group("/paymentHistory")

	// All the following routes use jwt authentication, you also can use middleware.Auth(middleware.WithVerify(fn))
	//g.Use(middleware.Auth())

	// If jwt authentication is not required for all routes, authentication middleware can be added
	// separately for only certain routes. In this case, g.Use(middleware.Auth()) above should not be used.

	g.POST("/", h.Create)          // [post] /api/v1/paymentHistory
	g.DELETE("/:id", h.DeleteByID) // [delete] /api/v1/paymentHistory/:id
	g.PUT("/:id", h.UpdateByID)    // [put] /api/v1/paymentHistory/:id
	g.GET("/:id", h.GetByID)       // [get] /api/v1/paymentHistory/:id
	g.POST("/list", h.List)        // [post] /api/v1/paymentHistory/list
}
