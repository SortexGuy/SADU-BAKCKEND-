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

func (h *TourneyHandler) GetAllTourneyHandler(ctx *gin.Context) {
	name := ctx.Query("name")
	status := ctx.Query("status")

	dtos, err := h.service.GetAllTourney(name, status)
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
		helpers.SendError(ctx, http.StatusBadRequest, "JSON inválido: "+err.Error(), "El torneo ya fue creado anteriormente o no fue encontrado.")
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
		if services.IsInvalidReference(err) {
			helpers.SendError(ctx, http.StatusBadRequest, "Referencia inválida", "No se pudo guardar el torneo: la disciplina indicada no existe.")
			return
		}
		helpers.SendError(ctx, http.StatusInternalServerError, "Error interno del servidor", "El inesperado a la hora de crear torneo.")
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
	
	updatedTourney, err := h.service.UpdateTourney(tourneyUpdate, ctx)
	if err != nil {
		if services.IsInvalidReference(err) {
			helpers.SendError(ctx, http.StatusBadRequest, "Referencia inválida", "No se pudo guardar el torneo: la disciplina indicada no existe.")
			return
		}
		helpers.SendError(ctx, http.StatusInternalServerError, "Error interno del servidor", "El inesperado a la hora de editar torneo.")
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
