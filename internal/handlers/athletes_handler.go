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

type AthleteHandler struct {
	service *services.AthleteService
}

func NewAthleteHandler(service *services.AthleteService) *AthleteHandler {
	return &AthleteHandler{
		service: service,
	}
}

func (h *AthleteHandler) GetAthletes(ctx *gin.Context) {
	name := ctx.Query("name")
	lastName := ctx.Query("lastname")
	govID := ctx.Query("govid")


	athletes, err := services.GetAllAthletes(name, lastName, govID)

	if err != nil {
		log.Printf("Error getting athletes: %v", err)

		helpers.SendError(ctx, http.StatusInternalServerError, "Error interno del servidor", "Ocurrió un problema inesperado al procesar la lista de atletas.")
		return
	}
	helpers.SendSucces(ctx, "listing-athletes", athletes)
}

func (h *AthleteHandler) GetAthletesByID(ctx *gin.Context) {

	athletes, err := h.service.GetAthletesByID(ctx)
	if err != nil {
		helpers.SendError(ctx, http.StatusNotFound, "Error de busqueda en la base de datos", "El ID del atleta esta mal escrito o no existe")
		return
	}
	helpers.SendSucces(ctx, "listing-athletes-by-id", athletes)
}


func (h *AthleteHandler) CreateNewAthlete(ctx *gin.Context) {
	// 1. BIND JSON  OBTENER DATOS del request
	var newAthlete schema.Athlete
	if err := ctx.ShouldBindJSON(&newAthlete); err != nil {
		helpers.SendError(ctx, http.StatusNotFound, "Error de busqueda en la base de datos", "Los datos del atleta no cargaron o no se encontraron en la Base de datos")
		return
	}

	createdAthlete, err := h.service.CreateAthlete(newAthlete)
	if err != nil {
		if services.IsInvalidReference(err) {
			helpers.SendError(ctx, http.StatusBadRequest, "Referencia inválida", "No se pudo guardar el atleta: la carrera, el equipo o la disciplina indicada no existe.")
			return
		}
		if errors.Is(err, services.ErrMissingGovID) {
			helpers.SendError(ctx, http.StatusBadRequest, "Cédula obligatoria", "El atleta debe tener una cédula para poder registrarlo.")
			return
		}
		if errors.Is(err, services.ErrMissingMajor) {
			helpers.SendError(ctx, http.StatusBadRequest, "Carrera obligatoria", "Debes indicar la carrera que cursa el atleta.")
			return
		}
		if errors.Is(err, services.ErrDuplicateGovID) {
			helpers.SendError(ctx, http.StatusConflict, "Cédula duplicada", "Ya existe un atleta registrado con esa cédula.")
			return
		}
		helpers.SendError(ctx, http.StatusInternalServerError, "Error interno del servidor", "Dato incorrecto ingresado, el atleta ya fue creado O problema inesperado")
		return
	}

	helpers.SendSucces(ctx, "Atleta creado exitosamente", createdAthlete)
}
func (h *AthleteHandler) EditAthleteByID(ctx *gin.Context) {

	var athlete schema.Athlete
	if err := ctx.ShouldBindJSON(&athlete); err != nil {
		helpers.SendError(ctx, http.StatusNotFound, "Error de busqueda en la base de datos", "Los datos del atleta no cargaron y no se encontraron en la Base de datos")
		return
	}
	updateAthlete, err := h.service.EditAthlete(athlete, ctx)
	if err != nil {
		if services.IsInvalidReference(err) {
			helpers.SendError(ctx, http.StatusBadRequest, "Referencia inválida", "No se pudo guardar el atleta: la carrera, el equipo o la disciplina indicada no existe.")
			return
		}
		if errors.Is(err, services.ErrDuplicateGovID) {
			helpers.SendError(ctx, http.StatusConflict, "Cédula duplicada", "Ya existe otro atleta registrado con esa cédula.")
			return
		}
		helpers.SendError(ctx, http.StatusInternalServerError, "Error interno del servidor", "Datos incorrectos ingresados en la edicion de atletas")
		return
	}
	helpers.SendSucces(ctx, "updated Athletes succesfully", updateAthlete)

}

func (h *AthleteHandler) DeleteAthleteByID(ctx *gin.Context) {

	if err := h.service.DeleteAthlete(ctx); err != nil {
		helpers.SendError(ctx, http.StatusNotFound, "Error de busqueda en la base de datos", "Los datos del atleta no cargaron y no se encontraron en la Base de datos para eliminarlo")
		return
	}

	helpers.SendSucces(ctx, "Deleting-athlete", "")
}
