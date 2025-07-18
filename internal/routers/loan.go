package routers

import (
	"github.com/gin-gonic/gin"

	"lol/internal/handler"
)

func init() {
	apiV1RouterFns = append(apiV1RouterFns, func(group *gin.RouterGroup) {
		loanRouter(group, handler.NewLoanHandler())
	})
}

func loanRouter(group *gin.RouterGroup, h handler.LoanHandler) {
	g := group.Group("/loan")

	// All the following routes use jwt authentication, you also can use middleware.Auth(middleware.WithVerify(fn))
	//g.Use(middleware.Auth())

	// If jwt authentication is not required for all routes, authentication middleware can be added
	// separately for only certain routes. In this case, g.Use(middleware.Auth()) above should not be used.

	g.POST("/", h.Create)          // [post] /api/v1/loan
	g.DELETE("/:id", h.DeleteByID) // [delete] /api/v1/loan/:id
	g.PUT("/:id", h.UpdateByID)    // [put] /api/v1/loan/:id
	g.GET("/:id", h.GetByID)       // [get] /api/v1/loan/:id
	g.POST("/list", h.List)        // [post] /api/v1/loan/list
	g.POST("/detail", h.GetDetail)
	g.POST("/pay", h.Pay)
	g.POST("/:bandName/notify", h.Notify)
}
