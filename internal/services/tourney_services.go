package services

import (
	"fmt"
	"strconv"

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
	var dtos []schema.TourneyGetBareDTO

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

	return schema.TourneyGetFullDTO{
		ID:        schema.RegularIDs(tourneyID),
		Name:      tourney.Name,
		Status:    tourney.Status,
		Events:    helpers.MapEventsGetDTO(tourney.Events),
		StartDate: tourney.StartDate,
		EndDate:   tourney.EndDate,
	}, nil
}

func (s *TourneyServices) CreateTourney(t schema.Tourney) (schema.Tourney, error) {

	result := s.DB.Omit("Events").Omit("Discipline").Create(&t)
	if result.Error != nil || result.RowsAffected == 0 {
		return schema.Tourney{}, result.Error
	}

	if len(t.Events) > 0 {
		s.DB.Model(&t).Preload("Events").Preload("Discipline").Association("Events").Append(t.Events)
	}

	s.DB.Preload("Events").Preload("Discipline").First(&t, t.ID)
	return t, nil
}

func (s *TourneyServices) UpdateTourney(t schema.Tourney, ctx *gin.Context) (schema.Tourney, error) {
	var id = ctx.Param("id")

	var updateTourney schema.Tourney

	if err := s.DB.First(&updateTourney, id).Error; err != nil {
		return schema.Tourney{}, fmt.Errorf("torneo no encontrado: %w", err)
	}

	if result := s.DB.Model(&updateTourney).Omit("Events").Where("id = ?", id).Updates(&t).Error; result != nil {
		return schema.Tourney{}, result
	}

	if t.Events != nil {
		if err := s.DB.Model(&updateTourney).Association("Events").Replace(t.Events); err != nil {
			return schema.Tourney{}, fmt.Errorf("error actualizando eventos: %w", err)
		}
	}


	err := s.DB.Preload("Events").Preload("Discipline").First(&updateTourney, id).Error

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
