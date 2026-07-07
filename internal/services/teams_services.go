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

type TeamServices struct {
	DB *gorm.DB
}

func NewTeamServices() *TeamServices {
	return &TeamServices{DB: config.DB}
}

func (s *TeamServices) GetAllTeam(name, category, disciplineID, universityID string, regular *bool) ([]schema.TeamGetDTO, error) {
	var teams []schema.Team
	query := s.DB.Preload("Athletes")

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if disciplineID != "" {
		query = query.Where("discipline_id = ?", disciplineID)
	}
	if universityID != "" {
		query = query.Where("university_id = ?", universityID)
	}
	if regular != nil {
		query = query.Where("regular = ?", *regular)
	}

	if err := query.Find(&teams).Error; err != nil {
		return nil, fmt.Errorf("listing teams: %w", err)
	}

	// Mapear a DTOs
	dtos := make([]schema.TeamGetDTO, len(teams))
	for i, team := range teams {
		dtos[i] = schema.TeamGetDTO{
			ID:           schema.RegularIDs(team.ID),
			Name:         team.Name,
			Regular:      team.Regular,
			Category:     string(team.Category),
			DisciplineID: schema.RegularIDs(team.DisciplineID),
			UniversityID: schema.RegularIDs(team.UniversityID),
			Athletes:     team.Athletes,
		}
	}
	return dtos, nil
}

func (s *TeamServices) GetAllTeamByID(ctx *gin.Context) (schema.TeamGetBareDTO, error) {
	id := ctx.Param("id")
	teamID, err := strconv.Atoi(id)
	if err != nil {
		return schema.TeamGetBareDTO{}, fmt.Errorf("invalid team ID: %w", err)
	}

	var team schema.Team
	if err := s.DB.Preload("University").Preload("Athletes").First(&team, teamID).Error; err != nil {
		return schema.TeamGetBareDTO{}, fmt.Errorf("team %d not found: %w", teamID, err)
	}

	return schema.TeamGetBareDTO{
		ID:       schema.RegularIDs(team.ID),
		Name:     team.Name,
		Regular:  team.Regular,
		Category: string(team.Category),
		University: schema.UniversityGetBareDTO{
			ID:   schema.RegularIDs(team.University.ID),
			Name: team.University.Name,
			Local: team.University.Local,
		},
		Athletes: helpers.MapAthletes(team.Athletes),
	}, nil
}

func (s *TeamServices) CreateTeam(t schema.Team) (schema.Team, error) {
err := s.DB.Transaction(func(tx *gorm.DB) error {

		if err := tx.Omit("Discipline", "University").Create(&t).Error; err != nil {
            return err
        }
		return tx.Preload("Athletes").Preload("Discipline").Preload("University").First(&t, t.ID).Error
	})

	if err != nil {
		return schema.Team{}, err
	}
	return t, nil
}

func (s *TeamServices) EditTeam(t schema.Team, ctx *gin.Context) (schema.Team, error){
id := ctx.Param("id")
    var team schema.Team

    err := s.DB.Transaction(func(tx *gorm.DB) error {
       
        if err := tx.First(&team, id).Error; err != nil {
            return fmt.Errorf("equipo no encontrado: %w", err)
        }

        if err := tx.Model(&team).Omit("Athletes", "Discipline", "University").Updates(&t).Error; err != nil {
            return err
        }

        if t.Athletes != nil {
          
            if err := tx.Model(&team).Association("Athletes").Replace(t.Athletes); err != nil {
                return err
            }
        }

        return tx.Preload("Athletes").
                  Preload("Discipline").
                  Preload("University").
                  First(&team, id).Error
    })

    return team, err
}

func (s *TeamServices) DeleteTeam(ctx *gin.Context) error {
	id := ctx.Param("id")
	teamID, err := strconv.Atoi(id)
	if err != nil {
		return fmt.Errorf("invalid team ID %q: %w", id, err)
	}

	var team schema.Team
	if err := s.DB.First(&team, teamID).Error; err != nil {
		return fmt.Errorf("team %d not found: %w", teamID, err)
	}

	result := s.DB.Delete(&team, teamID)
	if result.Error != nil {
		return fmt.Errorf("deleting team %d: %w", teamID, result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("team %d not found or already deleted", teamID)
	}

	return nil
}
