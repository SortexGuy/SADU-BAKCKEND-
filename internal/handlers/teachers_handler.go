package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"uneg.edu.ve/servicio-sadu-back/helpers"
	"uneg.edu.ve/servicio-sadu-back/internal/services"
	"uneg.edu.ve/servicio-sadu-back/schema"
)

type TeacherHandler struct {
	service *services.TeacherService
}

func NewTeacherHandler(service *services.TeacherService) *TeacherHandler {
	return &TeacherHandler{service: service}
}
func (h *TeacherHandler) GetAllTeachersHandler(ctx *gin.Context) {

	name := ctx.Query("name")
	lastName := ctx.Query("lastName")
	govID := ctx.Query("govID")
	disciplineID := ctx.Query("discipline_id")

	teachersDTO, err := h.service.GetTeachers(name, lastName, govID, disciplineID)

	if err != nil {
		helpers.SendError(ctx, http.StatusInternalServerError, "Error interno del servidor", "Ocurrió un problema inesperado al procesar la lista de profesores.")
		return
	}
	helpers.SendSucces(ctx, "Listing-Teachers-Succesfully", teachersDTO)
}

func (h *TeacherHandler) GetTeacherByIdHandler(ctx *gin.Context) {
	dto, err := h.service.GetTeacherById(ctx)
	if err != nil {

		if strings.Contains(err.Error(), "no encontrado") {

			helpers.SendError(ctx, http.StatusNotFound, "Error de busqueda en la base de datos", "El ID del profesor esta mal escrito o no se encuentra en la base de datos.")
		} else {
			helpers.SendError(ctx, http.StatusNotFound, "Error interno del servidor", "Error obteniendo profesor por su ID de base de datos.")
		}
		return
	}
	helpers.SendSucces(ctx, "Listing-Teachers-By-ID-Succesfully", dto)
}

func (h *TeacherHandler) CreateTeacherHandler(ctx *gin.Context) {

	var teacher schema.Teacher

	if err := ctx.ShouldBindJSON(&teacher); err != nil {
		helpers.SendError(ctx, http.StatusNotFound, "Error de busqueda en la base de datos", "El profesor ya fue creado anteriormente o no fue encontrado.")
		return
	}

	dto, err := h.service.CreateTeacher(teacher)
	if err != nil {
		helpers.SendError(ctx, http.StatusInternalServerError, "Error interno del servidor", "Datos incorrectos ingresados en la creacion del profesor.")
		return
	}
	helpers.SendSucces(ctx, "Teacher-Created-Succesfully", dto)
}

func (h *TeacherHandler) UpdateTeacherHandler(ctx *gin.Context) {

	var teacher schema.Teacher

	if err := ctx.ShouldBindJSON(&teacher); err != nil {
		helpers.SendError(ctx, http.StatusInternalServerError, "Error interno del servidor", "Los datos del profesor no fueron encontrado o el profesor fue eliminado anteriormente.")
		return
	}

	teacher, err := h.service.EditTeacher(ctx, teacher)
	if err != nil {
		if strings.Contains(err.Error(), "no encontrado") {
			helpers.SendError(ctx, http.StatusNotFound, "Error de busqueda en la base de datos", "Los datos del profesor no fueron encontrado o el profesor fue eliminado anteriormente.")
		} else {
			helpers.SendError(ctx, http.StatusInternalServerError, "Error interno del servidor", "Datos incorrectos ingresados en la actualizacion del profesor.")
		}
		return
	}
	helpers.SendSucces(ctx, "Updated-Teacher-Succesfully", teacher)
}

func (h *TeacherHandler) DeleteTeacherHandler(ctx *gin.Context) {
	if err := h.service.DeleteTeacher(ctx); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			helpers.SendError(ctx, http.StatusNotFound, "Error de busqueda en la base de datos","El ID del profesor no se encuentra, profesor no encontrado.")
		} else {
			helpers.SendError(ctx, http.StatusInternalServerError, "Error interno del servidor","Profesor no encontrado.")
		}
		return
	}

	helpers.SendSucces(ctx, "Deleting Teacher succesfully.", "")
}
