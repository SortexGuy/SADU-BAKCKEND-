package services

import (
	"fmt"
	"log/slog"
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

func GetAllAthletes(name, lastname, govID, gender, disciplineID, search string) ([]schema.AthleteDTO, error) {
	var athletes []schema.Athlete
	query := config.DB.Model(&schema.Athlete{})

	// `search` es el buscador unico de la interfaz: coincide con nombre, apellido o
	// cedula. Los parentesis son necesarios para que el OR no se mezcle con los
	// demas filtros, que se combinan con AND.
	if search != "" {
		patron := "%" + search + "%"
		query = query.Where("(first_names LIKE ? OR last_names LIKE ? OR gov_id LIKE ?)", patron, patron, patron)
	}
	if name != "" {
		query = query.Where("first_names LIKE ?", "%"+name+"%")
	}
	if lastname != "" {
		query = query.Where("last_names LIKE ?", "%"+lastname+"%")
	}
	if govID != "" {
		query = query.Where("gov_id LIKE ?", "%"+govID+"%")
	}
	if gender != "" {
		query = query.Where("gender = ?", gender)
	}
	if disciplineID != "" {
		query = query.Joins("JOIN athlete_disciplines ON athlete_disciplines.athlete_id = athletes.id").
			Where("athlete_disciplines.discipline_id = ?", disciplineID)
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

	// El alta y sus asociaciones van en una transaccion: si una vinculacion falla
	// (por ejemplo, un equipo que no existe) no queda un atleta a medio crear.
	// Antes los errores de Association se descartaban, asi que un fallo devolvia 200
	// y la vinculacion simplemente no aparecia.
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Teams", "Events", "Disciplines").Create(&a).Error; err != nil {
			if isUniqueViolation(err) {
				return ErrDuplicateGovID
			}
			return err
		}

		if len(a.Disciplines) > 0 {
			if err := tx.Model(&a).Association("Disciplines").Append(a.Disciplines); err != nil {
				return fmt.Errorf("asociando disciplinas: %w", err)
			}
		}

		if len(a.Events) > 0 {
			if err := tx.Model(&a).Association("Events").Append(a.Events); err != nil {
				return fmt.Errorf("asociando eventos: %w", err)
			}
		}

		if len(a.Teams) > 0 {
			if err := tx.Model(&a).Association("Teams").Append(a.Teams); err != nil {
				return fmt.Errorf("asociando equipos: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return a, err
	}

	slog.Info("atleta creado",
		"atleta_id", a.ID,
		"cedula", a.GovID,
		"carrera_id", a.MajorID,
		"disciplinas", len(a.Disciplines),
		"equipos", len(a.Teams),
		"eventos", len(a.Events),
	)
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

	// Igual que en el alta: si una vinculacion falla hay que enterarse.
	if len(a.Teams) > 0 {
		if err := s.DB.Model(&athlete).Association("Teams").Replace(a.Teams); err != nil {
			return schema.Athlete{}, fmt.Errorf("asociando equipos: %w", err)
		}
	}

	if len(a.Disciplines) > 0 {
		if err := s.DB.Model(&athlete).Association("Disciplines").Replace(a.Disciplines); err != nil {
			return schema.Athlete{}, fmt.Errorf("asociando disciplinas: %w", err)
		}
	}
	if len(a.Events) > 0 {
		if err := s.DB.Model(&athlete).Association("Events").Replace(a.Events); err != nil {
			return schema.Athlete{}, fmt.Errorf("asociando eventos: %w", err)
		}
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

	// Borrado logico: la fila permanece con deleted_at y queda fuera de las consultas.
	slog.Info("atleta eliminado", "atleta_id", athleteID)
	return nil
}
