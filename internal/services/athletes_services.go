package services

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"uneg.edu.ve/servicio-sadu-back/config"
	"uneg.edu.ve/servicio-sadu-back/helpers"
	"uneg.edu.ve/servicio-sadu-back/schema"
)

// conexión a la DB
type AthleteService struct {
	DB *gorm.DB
}

func NewAthleteService() *AthleteService {
	return &AthleteService{DB: config.DB}
}

//GET  METHOD

func GetAllAthletes(name, lastname, govID string) ([]schema.AthleteDTO, error) {
	var athletes []schema.Athlete
	query := config.DB.Model(&schema.Athlete{})

	if name != "" {
		query = query.Where("first_names LIKE ?", "%"+name+"%")
	}
	if lastname != "" {
		query = query.Where("last_names LIKE ?", "%"+lastname+"%")
	}
	if govID != "" {
		query = query.Where("gov_id LIKE ?", "%"+govID+"%")
	}

	if err := query.Preload("Disciplines").Find(&athletes).Error; err != nil {
		return nil, err
	}
	athleteDTO := make([]schema.AthleteDTO, len(athletes))
	for i, value := range athletes {
		athleteDTO[i] = schema.AthleteDTO{
			ID:          schema.RegularIDs(value.ID),
			GovID:       value.GovID,
			FirstNames:  value.FirstNames,
			LastNames:   value.LastNames,
			PhoneNumber: value.PhoneNumber,
			Gender:      value.Gender,
			Email:       value.Email,
			Enrolled:    value.Enrolled,
			Regular:     value.Regular,
			Disciplines: helpers.MapDisciplines(value.Disciplines),
		}
	}
	return athleteDTO, nil
}

// GET BY ID
func (s *AthleteService) GetAthletesByID(ctx *gin.Context) (schema.Athlete, error) {
	var id = ctx.Param("id")
	athleteID, err := strconv.Atoi(id)
	
	//.Preload("Teams.Discipline.Teams")
	query := s.DB.Preload("Teams.University", nil).Preload("Teams.Discipline", nil).Preload("Teams", nil).Preload("Disciplines").Preload("Events")

	if err != nil {
		return schema.Athlete{}, fmt.Errorf("ID invalido: %v", err)
	}
	var athlete schema.Athlete

	result := query.First(&athlete, athleteID)
	if result.Error != nil {
		return schema.Athlete{}, result.Error
	}

	return athlete, nil
}

// govIDTaken indica si otro atleta distinto de excludeID ya usa esa cedula.
// Las cedulas vacias no se validan: la base admite historicamente registros sin
// cedula y rechazarlos aqui cambiaria el comportamiento de los formularios.
func (s *AthleteService) govIDTaken(govID string, excludeID uint) (bool, error) {
	if strings.TrimSpace(govID) == "" {
		return false, nil
	}

	query := s.DB.Model(&schema.Athlete{}).Where("gov_id = ?", govID)
	if excludeID != 0 {
		query = query.Where("id <> ?", excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// POST METHOD
func (s *AthleteService) CreateAthlete(a schema.Athlete) (schema.Athlete, error) {
	if strings.TrimSpace(a.GovID) == "" {
		return a, ErrMissingGovID
	}
	if a.MajorID == 0 {
		return a, ErrMissingMajor
	}

	taken, err := s.govIDTaken(a.GovID, 0)
	if err != nil {
		return a, err
	}
	if taken {
		return a, ErrDuplicateGovID
	}

	if err := s.DB.Omit("Teams", "Events", "Disciplines").Create(&a).Error; err != nil {
		if isUniqueViolation(err) {
			return a, ErrDuplicateGovID
		}
		return a, err
	}

	if len(a.Disciplines) > 0 {
		s.DB.Model(&a).Preload("Disciplines").Association("Disciplines").Append(a.Disciplines)
	}

	if len(a.Events) > 0 {
		s.DB.Model(&a).Preload("Events").Association("Events").Append(a.Events)
	}

	if len(a.Teams) > 0 {
		s.DB.Model(&a).Preload("Teams").Association("Teams").Append(a.Teams)
	}
	return a, nil
}

// PUT METHOD
func (s *AthleteService) EditAthlete(a schema.Athlete, ctx *gin.Context) (schema.Athlete, error) {
	var id = ctx.Param("id")
	athleteID, err := strconv.Atoi(id)

	if err != nil {
		return schema.Athlete{}, fmt.Errorf("ID invalido: %v", err)
	}

	var athlete schema.Athlete
	if err := s.DB.First(&athlete, athleteID).Error; err != nil {
		return schema.Athlete{}, fmt.Errorf("atleta no encontrado: %d", athleteID)
	}

	taken, err := s.govIDTaken(a.GovID, athlete.ID)
	if err != nil {
		return schema.Athlete{}, err
	}
	if taken {
		return schema.Athlete{}, ErrDuplicateGovID
	}

	//Actualizar campos escalares
	s.DB.Model(&athlete).Updates(&a)

	if len(a.Teams) > 0 {
		s.DB.Model(&athlete).Association("Teams").Replace(a.Teams)
	}

	if len(a.Disciplines) > 0 {
		s.DB.Model(&athlete).Association("Disciplines").Replace(a.Disciplines)
	}
	if len(a.Events) > 0 {
		s.DB.Model(&athlete).Association("Events").Replace(a.Events)
	}

	return athlete, s.DB.Preload("Teams").Preload("Disciplines").Preload("Events").First(&athlete, athleteID).Error
}

// DELETE METHOD
func (s *AthleteService) DeleteAthlete(ctx *gin.Context) error {
	var id = ctx.Param("id")
	athleteID, err := strconv.Atoi(id)

	if err != nil {
		return fmt.Errorf("ID inválido: %w", err)
	}
	result := s.DB.Delete(&schema.Athlete{}, athleteID)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("atleta no encontrado: %d", athleteID)
	}
	return nil
}
