package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"uneg.edu.ve/servicio-sadu-back/helpers"
	"uneg.edu.ve/servicio-sadu-back/internal/services"
	"uneg.edu.ve/servicio-sadu-back/schema"
)

type TeamHandler struct {
	service *services.TeamServices
}

func NewTeamHandler(service *services.TeamServices) *TeamHandler {
	return &TeamHandler{service: service}
}

func (h *TeamHandler) GetAllTeamHandler(ctx *gin.Context) {
	name := ctx.Query("name")
	category := ctx.Query("category")
	disciplineID := ctx.Query("discipline_id")
	universityID := ctx.Query("university_id")

	var regular *bool
	if reg := ctx.Query("regular"); reg != "" {
		b := reg == "true"
		regular = &b
	}

	team, err := h.service.GetAllTeam(name, category, disciplineID, universityID, regular)
	if err != nil {

		if strings.Contains(err.Error(), "team") && strings.Contains(err.Error(), "not found") {
			helpers.SendError(ctx, http.StatusNotFound, "Error interno del servidor", "Ocurrió un problema inesperado al procesar la lista de equipos.")
			return
		}

		helpers.SendError(ctx, http.StatusInternalServerError, "Error interno del servidor", "Ocurrió un problema inesperado al procesar la lista de profesores.")
		return
	}
	helpers.SendSucces(ctx, "Listing-Teams-Succesfully", team)
}

func (h *TeamHandler) GetTeamByIdHandler(ctx *gin.Context) {
	team, err := h.service.GetAllTeamByID(ctx)
	if err != nil {

		if strings.Contains(err.Error(), "invalid team ID") {
			helpers.SendError(ctx, http.StatusNotFound, "Error de busqueda", "El ID del equipo esta mal escrito o no se encuentra en la base de datos.")
			return
		}
		if strings.Contains(err.Error(), "team") && strings.Contains(err.Error(), "not found") {
			helpers.SendError(ctx, http.StatusInternalServerError, "Error interno del servidor", "Error obteniendo equipo por su ID de base de datos.")
			return
		}

		helpers.SendError(ctx, http.StatusInternalServerError, "Error interno del servidor", "Error obteniendo equipo por su ID de base de datos.")
		return
	}
	helpers.SendSucces(ctx, "Listing-Team-By-ID-Succesfully", team)
}

func (h *TeamHandler) CreateTeamHandler(ctx *gin.Context) {

	var input schema.Team

	if err := ctx.ShouldBindJSON(&input); err != nil {
		helpers.SendError(ctx, http.StatusBadRequest, "Datos inválidos", "El formato del equipo enviado no es correcto.")
		return
	}

	team, err := h.service.CreateTeam(input)
	if err != nil {
		if services.IsInvalidReference(err) {
			helpers.SendError(ctx, http.StatusBadRequest, "Referencia inválida", "No se pudo guardar el equipo: la disciplina o la universidad indicada no existe.")
			return
		}
		errLower := strings.ToLower(err.Error())
		if strings.Contains(errLower, "not found") ||
			strings.Contains(errLower, "duplicate") ||
			strings.Contains(errLower, "foreign key") {

			helpers.SendError(ctx, http.StatusBadRequest, "Error de validación", err.Error())
			return
		}

		helpers.SendError(ctx, http.StatusInternalServerError, "Error interno del servidor", "El inesperado a la hora de crear equipo.")
		return
	}
	helpers.SendSucces(ctx, "Team-Created-Succesfully.", team)
}

func (h *TeamHandler) EditTeamHandler(ctx *gin.Context) {
  
    var input schema.Team

    if err := ctx.ShouldBindJSON(&input); err != nil {
        helpers.SendError(ctx, http.StatusBadRequest, "Datos de entrada inválidos", "El formato del JSON enviado no coincide con la estructura de un equipo.")
        return
    }

    updatedTeam, err := h.service.EditTeam(input, ctx)
    
    if err != nil {
       
        if strings.Contains(strings.ToLower(err.Error()), "no encontrado") {
            helpers.SendError(ctx, http.StatusNotFound, "Equipo no encontrado", err.Error())
            return
        }

       
        helpers.SendError(ctx, http.StatusInternalServerError, "Error al actualizar equipo", "Ocurrió un error inesperado al procesar la edición.")
        return
    }

    
    helpers.SendSucces(ctx, "Equipo actualizado exitosamente.", updatedTeam)
}

func (h *TeamHandler) DeleteTeam(ctx *gin.Context) {
	err := h.service.DeleteTeam(ctx)
	if err != nil {

		if strings.Contains(err.Error(), "invalid team ID.") {
			helpers.SendError(ctx, http.StatusBadRequest, "Datos de entrada inválidos", "El ID del equipo no fue encontrado en la base de datos, invalido ID.")
			return
		}
		if strings.Contains(err.Error(), "team") && strings.Contains(err.Error(), "not found.") ||
			strings.Contains(err.Error(), "already deleted.") {
			helpers.SendError(ctx, http.StatusNotFound, "Error interno del servidor", "Algun dato del equipo no fue encontrado en la base de datos.")
			return
		}
		if strings.Contains(err.Error(), "deleting team.") {
			helpers.SendError(ctx, http.StatusBadRequest, "Error interno del servidor", "Error eliminando el equipo seleccionado, el equipo ya fue eliminado.")
			return
		}

		helpers.SendError(ctx, http.StatusBadRequest, "Error interno del servidor", "Error eliminando el equipo seleccionado.")
		return
	}
	helpers.SendSucces(ctx, "Team-Deleted-Succesfully.", "")
}
