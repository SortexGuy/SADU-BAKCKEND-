package routes

import (
	"github.com/gin-gonic/gin"
	"uneg.edu.ve/servicio-sadu-back/internal/handlers"
)

func RegisterUserRoutes(r *gin.RouterGroup, userHandler *handlers.UserHandler) {
	r.POST("/login", userHandler.LoginUserHandler)
}
