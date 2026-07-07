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

func (s *EventService) GetEvents(id uint, name, status, disciplineID, dateFrom, dateTo, teamName string) ([]schema.EventGetDTO, error) {

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
	if name != " " {
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
	if teamName != "" {
		query = query.Joins("LEFT JOIN teams AS home ON events.home_team_id = home.id").
			Joins("LEFT JOIN teams AS opposite ON events.opposite_team_id = opposite.id").
			Where("home.name LIKE ? OR opposite.name LIKE ?", "%"+teamName+"%", "%"+teamName+"%")
	}
	if err := query.Find(&event).Error; err != nil {
		return nil, nil
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

	if err := tx.Omit("HomeTeam", "OppositeTeam", "Tourney", "ResponsableTeacher", "Discipline", "Athletes").
		Create(&event).Error; err != nil {
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

	tx := s.DB.Begin()
	err := tx.Model(&existingEvent).Omit("Athletes").Updates(map[string]interface{}{
		"Name":                 dto.Name,
		"Date":                 dto.Date,
		"Status":               dto.Status,
		"Observation":          dto.Observation,
		"Ubication":            dto.Ubication,
		"HomePoints":           dto.HomePoints,
		"OppositePoints":       dto.OppositePoints,
		"HomeTeamID":           dto.HomeTeamID,
		"OppositeTeamID":       dto.OppositeTeamID,
		"TourneyID":            dto.TourneyID,
		"ResponsableTeacherID": dto.ResponsableTeacherID,
		"DisciplineID":         dto.DisciplineID,
	}).Error

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
