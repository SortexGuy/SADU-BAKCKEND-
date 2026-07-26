package routes

import (
	"github.com/gin-gonic/gin"
	"uneg.edu.ve/servicio-sadu-back/internal/handlers"
	"uneg.edu.ve/servicio-sadu-back/internal/middlewares"
)

func RegisterUserRoutes(r *gin.RouterGroup, userHandler *handlers.UserHandler) {
	r.POST("/login", userHandler.LoginUserHandler)

	protected := r.Group("/", middlewares.AuthMiddleware())
	protected.PUT("/change-password", userHandler.ChangePasswordHandler)
}
