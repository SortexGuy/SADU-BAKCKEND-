package services

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"uneg.edu.ve/servicio-sadu-back/config"
	"uneg.edu.ve/servicio-sadu-back/helpers"
	"uneg.edu.ve/servicio-sadu-back/schema"
)

type TourneyServices struct {
	DB *gorm.DB
}

func NewTourneyServices() *TourneyServices {
	return &TourneyServices{DB: config.DB}
}

// validarRango comprueba que el fin no sea anterior al inicio. Si alguna de las
// dos fechas no esta definida no hay rango que validar: son opcionales, y los
// torneos cargados antes de que el formulario las pidiera no tienen ninguna.
func validarRango(inicio, fin time.Time) error {
	if inicio.IsZero() || fin.IsZero() {
		return nil
	}
	if fin.Before(inicio) {
		return ErrInvalidDateRange
	}
	return nil
}

func (s *TourneyServices) GetAllTourney(name, status, disciplineID string) ([]schema.TourneyGetBareDTO, error) {
	var tourneys []schema.Tourney
	query := s.DB.Preload("Events").Preload("Discipline")

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if status != "" {
		query = query.Where("status LIKE ?", "%"+status+"%")
	}
	if disciplineID != "" {
		query = query.Where("discipline_id = ?", disciplineID)
	}

	if err := query.Find(&tourneys).Error; err != nil {
		return nil, err
	}
	dtos := []schema.TourneyGetBareDTO{}

	for _, t := range tourneys {

		dto := schema.TourneyGetBareDTO{
			ID:          schema.RegularIDs(t.ID),
			Name:        t.Name,
			Status:      t.Status,
			StartDate:   t.StartDate,
			EndDate:     t.EndDate,
			TotalEvents: uint(len(t.Events)),
			DisciplineID:  schema.RegularIDs(t.DisciplineID),
            DisciplineName: t.Discipline.Name,
		}
		dtos = append(dtos, dto)
	}
	return dtos, nil
}

func (s *TourneyServices) GetTourneyByID(ctx *gin.Context) (schema.TourneyGetFullDTO, error) {
	var id = ctx.Param("id")
	var tourney schema.Tourney
	tourneyID, err := strconv.Atoi(id)
	if err != nil {
		return schema.TourneyGetFullDTO{}, fmt.Errorf("ID INVALID:%w", err)

	}
	result := s.DB.Preload("Discipline").Preload("Events").Preload("Events.HomeTeam.University").
		Preload("Events.HomeTeam.Athletes").
		Preload("Events.OppositeTeam.University").
		Preload("Events.OppositeTeam.Athletes").
		Preload("Events.ResponsableTeacher.Disciplines").
		Preload("Events.Tourney").
		Preload("Events.Discipline").First(&tourney, tourneyID).Error
	if result != nil {
		return schema.TourneyGetFullDTO{}, result
	}

	// La disciplina viaja en el detalle porque el formulario de edicion la precarga
	// y filtra con ella los partidos que se pueden seleccionar.
	return schema.TourneyGetFullDTO{
		ID:             schema.RegularIDs(tourneyID),
		Name:           tourney.Name,
		Status:         tourney.Status,
		Events:         helpers.MapEventsGetDTO(tourney.Events),
		StartDate:      tourney.StartDate,
		EndDate:        tourney.EndDate,
		TotalEvents:    uint(len(tourney.Events)),
		DisciplineID:   tourney.DisciplineID,
		DisciplineName: tourney.Discipline.Name,
	}, nil
}

func (s *TourneyServices) CreateTourney(t schema.Tourney) (schema.Tourney, error) {
	// La disciplina define de que deporte es el torneo y es la que usa el filtro
	// del listado, asi que se exige al crear. Un 0 no referencia ninguna
	// disciplina real y la clave foranea lo rechazaria con un error opaco.
	if t.DisciplineID == 0 {
		return schema.Tourney{}, ErrMissingDiscipline
	}

	if err := validarRango(t.StartDate, t.EndDate); err != nil {
		return schema.Tourney{}, err
	}

	result := s.DB.Omit("Events", "Discipline").Create(&t)
	if result.Error != nil || result.RowsAffected == 0 {
		return schema.Tourney{}, result.Error
	}

	if len(t.Events) > 0 {
		if err := s.DB.Model(&t).Association("Events").Append(t.Events); err != nil {
			return schema.Tourney{}, fmt.Errorf("asociando eventos al torneo: %w", err)
		}
	}

	s.DB.Preload("Events").Preload("Discipline").First(&t, t.ID)

	slog.Info("torneo creado",
		"torneo_id", t.ID,
		"disciplina_id", t.DisciplineID,
		"partidos", len(t.Events),
	)
	return t, nil
}

// UpdateTourney actualiza el torneo indicado en la ruta. eventosDados distingue
// una peticion que trae la lista de partidos (aunque sea vacia: desasocia todos)
// de una que no la menciona, en la que los partidos quedan como estaban.
func (s *TourneyServices) UpdateTourney(t schema.Tourney, eventosDados bool, ctx *gin.Context) (schema.Tourney, error) {
	var id = ctx.Param("id")

	var updateTourney schema.Tourney

	if err := s.DB.First(&updateTourney, id).Error; err != nil {
		return schema.Tourney{}, fmt.Errorf("torneo no encontrado: %w", err)
	}

	// Updates con una struct ignora los campos en cero, asi que el rango se valida
	// sobre los valores que quedarian guardados: lo que trae la peticion y, para lo
	// que no trae, lo que ya estaba.
	inicio, fin := updateTourney.StartDate, updateTourney.EndDate
	if !t.StartDate.IsZero() {
		inicio = t.StartDate
	}
	if !t.EndDate.IsZero() {
		fin = t.EndDate
	}
	if err := validarRango(inicio, fin); err != nil {
		return schema.Tourney{}, err
	}

	if result := s.DB.Model(&updateTourney).Omit("Events").Where("id = ?", id).Updates(&t).Error; result != nil {
		return schema.Tourney{}, result
	}

	if eventosDados {
		asociacion := s.DB.Model(&updateTourney).Association("Events")
		var errEventos error
		if len(t.Events) == 0 {
			// Quedarse sin partidos se pide con Clear, que pone su tourney_id en
			// NULL; Replace con una lista vacia no hace lo mismo.
			errEventos = asociacion.Clear()
		} else {
			errEventos = asociacion.Replace(t.Events)
		}
		if errEventos != nil {
			return schema.Tourney{}, fmt.Errorf("error actualizando eventos: %w", errEventos)
		}
	}

	err := s.DB.Preload("Events").Preload("Discipline").First(&updateTourney, id).Error

	slog.Info("torneo actualizado",
		"torneo_id", updateTourney.ID,
		"disciplina_id", updateTourney.DisciplineID,
		"partidos", len(updateTourney.Events),
		"partidos_reemplazados", eventosDados,
	)
	return updateTourney, err

}

func (s *TourneyServices) DeleteTourney(ctx *gin.Context) error {
	var id = ctx.Param("id")
	tourneyID, err := strconv.Atoi(id)
	if err != nil {
		return fmt.Errorf("Invalid ID")
	}
	var result = s.DB.Delete(&schema.Tourney{}, tourneyID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("No se ha encontrado Torneo")
	}

	return nil
}
