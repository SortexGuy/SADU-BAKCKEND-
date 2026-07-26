package services

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"uneg.edu.ve/servicio-sadu-back/config"
	"uneg.edu.ve/servicio-sadu-back/helpers"
	"uneg.edu.ve/servicio-sadu-back/schema"
)

type EventService struct {
	DB *gorm.DB
}

func NewEventService() *EventService {
	return &EventService{DB: config.DB}
}

func (s *EventService) GetEvents(id uint, name, status, disciplineID, dateFrom, dateTo, teamName, universityID string) ([]schema.EventGetDTO, error) {

	var event []schema.Event

	query := s.DB.Preload("HomeTeam").
		Preload("HomeTeam.University").
		Preload("OppositeTeam.University").
		Preload("Tourney").
		Preload("ResponsableTeacher.Disciplines").
		Preload("ResponsableTeacher").
		Preload("Discipline")

	if id != 0 {
		query = query.Where("events.id = ?", id)
	}
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	if status != "" {
		query = query.Where("status LIKE ?", "%"+status+"%")
	}
	if disciplineID != "" {
		query = query.Where("discipline_id = ?", disciplineID)
	}
	if dateFrom != "" {
		query = query.Where("date >= ?", dateFrom)
	}
	if dateTo != "" {
		query = query.Where("date <= ?", dateTo+" 23:59:59")
	}
	// Los filtros por nombre de equipo y por universidad necesitan las mismas dos
	// uniones, asi que se agregan una sola vez si alguno esta activo.
	if teamName != "" || universityID != "" {
		query = query.Joins("LEFT JOIN teams AS home ON events.home_team_id = home.id").
			Joins("LEFT JOIN teams AS opposite ON events.opposite_team_id = opposite.id")
	}
	if teamName != "" {
		// Los parentesis evitan que el OR se mezcle con los filtros combinados con AND.
		query = query.Where("(home.name LIKE ? OR opposite.name LIKE ?)", "%"+teamName+"%", "%"+teamName+"%")
	}
	if universityID != "" {
		query = query.Where("(home.university_id = ? OR opposite.university_id = ?)", universityID, universityID)
	}
	if err := query.Find(&event).Error; err != nil {
		return nil, fmt.Errorf("listando eventos: %w", err)
	}

	dto := helpers.MapEventsGetDTO(event)

	return dto, nil
}

func (s *EventService) CreateEvent(dto schema.EventPOSTandPUTDTO) (schema.Event, error) {
	// var tourney schema.Tourney

	// if err := s.DB.First(&tourney, dto.TourneyID).Error; err != nil {
	// 	return schema.Event{}, fmt.Errorf("torneo no encontrado: %w", err)
	// }

	// if dto.DisciplineID != tourney.DisciplineID {
	// 	return schema.Event{}, fmt.Errorf("la disciplina del evento no coincide con la del torneo")
	// }

	event := schema.Event{
		Name:                 dto.Name,
		Date:                 dto.Date,
		Status:               dto.Status,
		Observation:          dto.Observation,
		Ubication:            dto.Ubication,
		HomePoints:           dto.HomePoints,
		OppositePoints:       dto.OppositePoints,
		HomeTeamID:           dto.HomeTeamID,
		OppositeTeamID:       dto.OppositeTeamID,
		TourneyID:            dto.TourneyID,
		ResponsableTeacherID: dto.ResponsableTeacherID,
		DisciplineID:         dto.DisciplineID,
	}

	tx := s.DB.Begin()

	// El torneo es opcional (el formulario de eventos no lo pide). Si no viene, la
	// columna se omite para que quede en NULL: un 0 no referencia a ningun torneo y
	// la clave foranea lo rechaza.
	omitir := []string{"HomeTeam", "OppositeTeam", "Tourney", "ResponsableTeacher", "Discipline", "Athletes"}
	if dto.TourneyID == 0 {
		omitir = append(omitir, "TourneyID")
	}

	if err := tx.Omit(omitir...).Create(&event).Error; err != nil {
		tx.Rollback()
		return schema.Event{}, err
	}

	tx.Commit()

	s.DB.Preload("HomeTeam").
		Preload("OppositeTeam").
		Preload("Tourney").
		Preload("ResponsableTeacher").
		Preload("Discipline").
		First(&event, event.ID)

	return event, nil
}
func (s *EventService) EditEvent(ctx *gin.Context, dto schema.EventPOSTandPUTDTO) (schema.Event, error) {

	id := ctx.Param("id")

	var existingEvent schema.Event
	if err := s.DB.First(&existingEvent, id).Error; err != nil {
		return schema.Event{}, fmt.Errorf("evento no encontrado: %w", err)
	}

	campos := map[string]interface{}{
		"Name":                 dto.Name,
		"Date":                 dto.Date,
		"Status":               dto.Status,
		"Observation":          dto.Observation,
		"Ubication":            dto.Ubication,
		"HomePoints":           dto.HomePoints,
		"OppositePoints":       dto.OppositePoints,
		"HomeTeamID":           dto.HomeTeamID,
		"OppositeTeamID":       dto.OppositeTeamID,
		"ResponsableTeacherID": dto.ResponsableTeacherID,
		"DisciplineID":         dto.DisciplineID,
	}
	// Sin torneo se guarda NULL, no 0. Ver CreateEvent.
	if dto.TourneyID == 0 {
		campos["TourneyID"] = nil
	} else {
		campos["TourneyID"] = dto.TourneyID
	}

	tx := s.DB.Begin()
	err := tx.Model(&existingEvent).Omit("Athletes").Updates(campos).Error

	if err != nil {
		tx.Rollback()
		return schema.Event{}, err
	}

	if err := tx.Commit().Error; err != nil {
		return schema.Event{}, err
	}

	s.DB.Preload("HomeTeam").
		Preload("OppositeTeam").
		Preload("Tourney").
		Preload("ResponsableTeacher").
		Preload("Discipline").
		First(&existingEvent, id)

	return existingEvent, nil

}
func (s *EventService) DeleteEvent(ctx *gin.Context) error {
	id := ctx.Param("id")

	result := s.DB.Delete(&schema.Event{}, id)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
