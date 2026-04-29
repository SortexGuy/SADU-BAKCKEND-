package services

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"uneg.edu.ve/servicio-sadu-back/config"
	"uneg.edu.ve/servicio-sadu-back/schema"
)

type MajorServices struct {
	DB *gorm.DB
}

func NewMajorServices() *MajorServices {
	return &MajorServices{DB: config.DB}
}

func (s *MajorServices) GetAllMajor(name string) ([]schema.MajorGetDTO, error) {
	dtos := []schema.MajorGetDTO{}
	query := s.DB.Model(&schema.Major{}).Select("id", "name")

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if err := query.Find(&dtos).Error; err != nil {
		return nil, err
	}
	return dtos, nil
}

func (s *MajorServices) GetMajorByID(ctx *gin.Context) (schema.Major, error) {
	var id = ctx.Param("id")
	majorID, err := strconv.Atoi(id)
	if err != nil {
		return schema.Major{}, fmt.Errorf("ERROR: ID INVALID %w", err)
	}
	var major schema.Major

	result := s.DB.Preload("Athletes").First(&major, majorID)
	if result.Error != nil {
		return schema.Major{}, result.Error
	}

	if result.RowsAffected == 0 {
		return schema.Major{}, fmt.Errorf("major %d not found", majorID)
	}
	return major, nil
}

func (s *MajorServices) CreateMajor(m schema.Major) (schema.Major, error) {

	if err := s.DB.Omit("Athletes").Create(&m).Error; err != nil {
		return m, err
	}

	if len(m.Athletes) > 0 {
		s.DB.Model(&m).Association("Athletes").Append(m.Athletes)
	}
	return m, nil
}

func (s *MajorServices) EditMajor(m schema.Major, ctx *gin.Context) (schema.Major, error) {
var id = ctx.Param("id")
	majorID, err := strconv.Atoi(id)

	if err != nil {
		return schema.Major{}, fmt.Errorf("ID invalido: %v", err)
	}
	var major schema.Major

	if err := s.DB.First(&major, majorID).Error; err != nil {
		return schema.Major{}, fmt.Errorf("Major not Found: %w", err)
	}
	s.DB.Model(&major).Select("Name").Updates(&m)

	if len(m.Athletes) > 0 {
		s.DB.Model(&major).Association("Teams").Replace(m.Athletes)
	}
	return major, s.DB.Preload("Athletes").First(&major, majorID).Error
}

func (s *MajorServices) DeleteMajor(ctx *gin.Context) error {
	var id = ctx.Param("id")
	majorID, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	result := s.DB.Delete(&schema.Major{}, majorID)
	if result != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("Athlete not found: ID:%d", majorID)
	}
	return nil

}
