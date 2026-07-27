package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"uneg.edu.ve/servicio-sadu-back/helpers"
	"uneg.edu.ve/servicio-sadu-back/internal/services"
	"uneg.edu.ve/servicio-sadu-back/schema"
)

type UserHandler struct {
	service *services.UserService
}

func NewUserHandler(service *services.UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (u *UserHandler) LoginUserHandler(ctx *gin.Context) {
	var loginData schema.LoginDTO
	if err := ctx.ShouldBindJSON(&loginData); err != nil {
		log.Printf("Error binding JSON: %v\n", err)
		helpers.SendError(ctx, http.StatusBadRequest, "INVALID_INPUT", "Invalid login data")
		return
	}

	token, err := u.service.LoginUser(loginData.Username, loginData.Password)
	if err != nil {
		helpers.SendError(ctx, http.StatusUnauthorized, "AUTH_FAILED", "Invalid credentials")
		return
	}

	helpers.SendSucces(ctx, "Successfully logged in", token)
}

// ProfileHandler devuelve la cuenta de la sesion abierta. El id se toma del
// token que dejo el middleware, nunca de la ruta ni del cuerpo.
func (u *UserHandler) ProfileHandler(ctx *gin.Context) {
	userID, exists := ctx.Get("userId")
	if !exists {
		helpers.SendError(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	perfil, err := u.service.GetProfile(userID.(uint))
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			helpers.SendError(ctx, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		helpers.SendError(ctx, http.StatusInternalServerError, "ERROR", err.Error())
		return
	}

	helpers.SendSucces(ctx, "profile", perfil)
}

// ChangeUsernameHandler cambia el correo con el que se inicia sesion.
func (u *UserHandler) ChangeUsernameHandler(ctx *gin.Context) {
	userID, exists := ctx.Get("userId")
	if !exists {
		helpers.SendError(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	var data schema.ChangeUsernameDTO
	if err := ctx.ShouldBindJSON(&data); err != nil {
		helpers.SendError(ctx, http.StatusBadRequest, "INVALID_INPUT", "Correo o contrasena invalidos")
		return
	}

	err := u.service.ChangeUsername(userID.(uint), data.CurrentPassword, data.NewUsername)
	switch {
	case err == nil:
		helpers.SendSucces(ctx, "correo actualizado", nil)
	case errors.Is(err, services.ErrInvalidCurrentPassword):
		helpers.SendError(ctx, http.StatusBadRequest, "INVALID_PASSWORD", err.Error())
	case errors.Is(err, services.ErrMissingUsername), errors.Is(err, services.ErrInvalidEmail):
		helpers.SendError(ctx, http.StatusBadRequest, "INVALID_INPUT", err.Error())
	case errors.Is(err, services.ErrUsernameTaken):
		helpers.SendError(ctx, http.StatusConflict, "USERNAME_TAKEN", err.Error())
	case errors.Is(err, services.ErrUserNotFound):
		helpers.SendError(ctx, http.StatusNotFound, "NOT_FOUND", err.Error())
	default:
		helpers.SendError(ctx, http.StatusInternalServerError, "ERROR", err.Error())
	}
}

func (u *UserHandler) ChangePasswordHandler(ctx *gin.Context) {
	userID, exists := ctx.Get("userId")
	if !exists {
		helpers.SendError(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	var data schema.ChangePasswordDTO
	if err := ctx.ShouldBindJSON(&data); err != nil {
		helpers.SendError(ctx, http.StatusBadRequest, "INVALID_INPUT", "Invalid data")
		return
	}

	if err := u.service.ChangePassword(userID.(uint), data.OldPassword, data.NewPassword); err != nil {
		if err.Error() == "current password is incorrect" {
			helpers.SendError(ctx, http.StatusBadRequest, "INVALID_PASSWORD", err.Error())
			return
		}
		helpers.SendError(ctx, http.StatusInternalServerError, "ERROR", err.Error())
		return
	}

	helpers.SendSucces(ctx, "password changed successfully", nil)
}
