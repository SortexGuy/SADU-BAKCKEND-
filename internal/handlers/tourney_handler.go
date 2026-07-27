package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"uneg.edu.ve/servicio-sadu-back/helpers"
	"uneg.edu.ve/servicio-sadu-back/internal/services"
	"uneg.edu.ve/servicio-sadu-back/schema"
)

type TourneyHandler struct {
	service *services.TourneyServices
}

func NewTourneyHandler(service *services.TourneyServices) *TourneyHandler {
	return &TourneyHandler{service: service}
}

// respondioErrorDeTorneo traduce los errores conocidos de guardar un torneo al
// codigo que corresponde y devuelve true si ya escribio la respuesta. Centraliza
// el mapeo porque crear y editar comparten las mismas validaciones.
func respondioErrorDeTorneo(ctx *gin.Context, err error) bool {
	switch {
	case errors.Is(err, services.ErrMissingDiscipline):
		helpers.SendError(ctx, http.StatusBadRequest, "Disciplina requerida", "Indica la disciplina del torneo antes de guardarlo.")
	case errors.Is(err, services.ErrInvalidDateRange):
		helpers.SendError(ctx, http.StatusBadRequest, "Rango de fechas inválido", "La fecha de fin del torneo no puede ser anterior a la de inicio.")
	case errors.Is(err, gorm.ErrRecordNotFound):
		helpers.SendError(ctx, http.StatusNotFound, "Torneo no encontrado", "El ID del torneo no corresponde a ningún registro.")
	case services.IsInvalidReference(err):
		helpers.SendError(ctx, http.StatusBadRequest, "Referencia inválida", "No se pudo guardar el torneo: la disciplina o alguno de los partidos indicados no existe.")
	default:
		return false
	}
	return true
}

func (h *TourneyHandler) GetAllTourneyHandler(ctx *gin.Context) {
	name := ctx.Query("name")
	status := ctx.Query("status")
	disciplineID := ctx.Query("discipline_id")

	dtos, err := h.service.GetAllTourney(name, status, disciplineID)
	if err != nil {
		helpers.SendError(ctx, http.StatusInternalServerError, "Error interno del servidor", "Ocurrió un problema inesperado al procesar la lista de torneos.")
		return
	}

	helpers.SendSucces(ctx, "Listing-Tourneys-Succesfully", dtos)
}

func (h *TourneyHandler) GetTourneyByIdHandler(ctx *gin.Context) {
	tourney, err := h.service.GetTourneyByID(ctx)
	if err != nil {
		helpers.SendError(ctx, http.StatusNotFound, "Error de busqueda", "El ID del equipo esta mal escrito o no se encuentra en la base de datos.")
		return
	}
	helpers.SendSucces(ctx, "Listing-Tourneys-By-ID-Succesfully", tourney)
}

func (h *TourneyHandler) CreateTourneyHandler(ctx *gin.Context) {
	var dto schema.TourneyPOSTandPUTDTO

	if err := ctx.ShouldBindJSON(&dto); err != nil {
		helpers.SendError(ctx, http.StatusBadRequest, "JSON inválido: "+err.Error(), "Los datos enviados para crear el torneo no tienen el formato esperado.")
		return
	}
	newTourney := schema.Tourney{
		Name:         dto.Name,
		Status:       dto.Status,
		StartDate:    dto.StartDate,
		EndDate:      dto.EndDate,
		DisciplineID: dto.DisciplineID,
	}

	for _, id := range dto.Events {
		newTourney.Events = append(newTourney.Events, schema.Event{
			Model: gorm.Model{ID: uint(id)},
		})
	}
	createdTourney, err := h.service.CreateTourney(newTourney)
	if err != nil {
		if respondioErrorDeTorneo(ctx, err) {
			return
		}
		helpers.SendError(ctx, http.StatusInternalServerError, "Error interno del servidor", "Ocurrió un problema inesperado al crear el torneo.")
		return
	}
	helpers.SendSucces(ctx, "CREATING-TOURNEY-SUCCESFULLY", createdTourney)

}

func (h *TourneyHandler) UpdateTourneyHandler(ctx *gin.Context) {

	var dto schema.TourneyPOSTandPUTDTO

	if err := ctx.ShouldBindJSON(&dto); err != nil {
		helpers.SendError(ctx, http.StatusBadRequest, "JSON inválido", err.Error())
		return
	}
	tourneyUpdate := schema.Tourney{
		Name:         dto.Name,
		Status:       dto.Status,
		StartDate:    dto.StartDate,
		EndDate:      dto.EndDate,
		DisciplineID: dto.DisciplineID,
	}


	for _, id := range dto.Events {
		tourneyUpdate.Events = append(tourneyUpdate.Events, schema.Event{
			Model: gorm.Model{ID: uint(id)},
		})
	}

	// La lista de partidos solo se reemplaza si la peticion la trae: encoding/json
	// deja dto.Events en nil cuando la clave no viene, y en una porcion vacia
	// cuando viene como []. Sin esa distincion no habria forma de dejar un torneo
	// sin partidos.
	updatedTourney, err := h.service.UpdateTourney(tourneyUpdate, dto.Events != nil, ctx)
	if err != nil {
		if respondioErrorDeTorneo(ctx, err) {
			return
		}
		helpers.SendError(ctx, http.StatusInternalServerError, "Error interno del servidor", "Ocurrió un problema inesperado al editar el torneo.")
		return

	}
	helpers.SendSucces(ctx, "EDITING-TOURNEY-SUCCESFULLY", updatedTourney)

}

func (h *TourneyHandler) DeleteTourneyHandler(ctx *gin.Context) {
	if err := h.service.DeleteTourney(ctx); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			helpers.SendError(ctx, http.StatusNotFound, "Torneo no encontrada", "El ID del torneo esta mal escrito o es invalido para buscar el torneo en la base de datos")
		} else {
			helpers.SendError(ctx, http.StatusInternalServerError, err.Error(), "Error interno en el servidor: no se encuentra torneo para eliminar")
		}

		
		return
	}

	helpers.SendSucces(ctx, "Deleting Tourney succesfully", "")
}
