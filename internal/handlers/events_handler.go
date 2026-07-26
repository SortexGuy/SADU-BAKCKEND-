package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"uneg.edu.ve/servicio-sadu-back/helpers"
	"uneg.edu.ve/servicio-sadu-back/internal/services"
	"uneg.edu.ve/servicio-sadu-back/schema"
)

type EventHandler struct {
	service *services.EventService
}

func NewEventHandler(service *services.EventService) *EventHandler {
	return &EventHandler{service: service}
}

func (h *EventHandler) GetEventsHandler(ctx *gin.Context) {
	idParam := ctx.Param("id")
	var id uint
	if idParam != "" {
		val, _ := strconv.Atoi(idParam)
		id = uint(val)
	}

	name := ctx.Query("name")
	status := ctx.Query("status")
	disciplineID := ctx.Query("discipline_id")
	dateFrom := ctx.Query("date_from")
	dateTo := ctx.Query("date_to")
	teamName := ctx.Query("team_name")

	events, err := h.service.GetEvents(id, name, status, disciplineID, dateFrom, dateTo, teamName)

	if err != nil {
		helpers.SendError(ctx, http.StatusInternalServerError, "Error interno del servidor", "Ocurrió un problema inesperado al procesar la lista de Eventos.")
		return
	}

	// El detalle devuelve un objeto, no un arreglo de un elemento, igual que el
	// resto de los recursos.
	if idParam != "" {
		if len(events) == 0 {
			helpers.SendError(ctx, http.StatusNotFound, "Error de busqueda en la base de datos", "El ID del evento esta mal escrito o no existe.")
			return
		}
		helpers.SendSucces(ctx, "GET-EVENT-BY-ID", events[0])
		return
	}

	helpers.SendSucces(ctx, "LISTIN-EVENTS-SUCCESFULLY", events)
}

func (h *EventHandler) CreateEventHandler(ctx *gin.Context) {

	var dto schema.EventPOSTandPUTDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		fmt.Printf("Error de binding: %v\n", err)
		helpers.SendError(ctx, http.StatusBadRequest, "Datos inválidos", "El formato del JSON enviado es incorrecto o faltan campos obligatorios")
		return
	}

	createdEvent, err := h.service.CreateEvent(dto)
	if err != nil {
		if services.IsInvalidReference(err) {
			helpers.SendError(ctx, http.StatusBadRequest, "Referencia inválida", "No se pudo guardar el evento: el equipo local, el visitante, el torneo, el profesor o la disciplina indicada no existe.")
			return
		}
		helpers.SendError(ctx, http.StatusInternalServerError, "Error interno del servidor", "No se pudo procesar la creación del evento en el servidor")
		return
	}

	helpers.SendSucces(ctx, "Evento creado exitosamente", createdEvent)
}

func (h *EventHandler) EditEventHandler(ctx *gin.Context) {

	var dto schema.EventPOSTandPUTDTO
	
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		fmt.Printf("Error de binding (JSON vacío o mal formado): %v\n", err)
		helpers.SendError(ctx, http.StatusNotFound, "Error de busqueda en la base de datos", "Los datos del evento no cargaron o no se encontraron en la Base de datos")
		return
	}

	updatedEvent, err := h.service.EditEvent(ctx,dto)
	if err != nil {
		if services.IsInvalidReference(err) {
			helpers.SendError(ctx, http.StatusBadRequest, "Referencia inválida", "No se pudo guardar el evento: el equipo local, el visitante, el torneo, el profesor o la disciplina indicada no existe.")
			return
		}
		fmt.Printf("Error en el servicio EditEvent: %v\n", err)
		helpers.SendError(ctx, http.StatusInternalServerError, "Error interno del servidor", "Datos incorrectos ingresados en la edicion de atletas")
		return
	}

	helpers.SendSucces(ctx, "Evento creado exitosamente", updatedEvent)
}

func (h *EventHandler) DeleteEventHandler(ctx *gin.Context) {

	if err := h.service.DeleteEvent(ctx); err != nil {
		helpers.SendError(ctx, http.StatusNotFound, "Error de busqueda en la base de datos", "Los datos del atleta no cargaron y no se encontraron en la Base de datos para eliminarlo")
		return
	}

	helpers.SendSucces(ctx, "Deleting-Event", "")
}
