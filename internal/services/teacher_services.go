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

type TeacherService struct {
	DB *gorm.DB
}

func NewTeacherService() *TeacherService {
	return &TeacherService{DB: config.DB}
}

func (s *TeacherService) GetTeachers(name, lastname, govID string) ([]schema.TeacherGetDTO, error) {
	var teachers []schema.Teacher
	query := s.DB.Model(&schema.Teacher{})
	if name != "" {
		query = query.Where("first_names LIKE ?", "%"+name+"%")
	}
	if lastname != "" {
		query = query.Where("last_names LIKE ?", "%"+lastname+"%")
	}
	if govID != "" {
		query = query.Where("gov_id LIKE ?", "%"+govID+"%")
	}

	if err := query.Preload("Disciplines").Find(&teachers).Error; err != nil {
		return nil, err
	}

	teachersDTO := make([]schema.TeacherGetDTO, len(teachers))

	for i, teacher := range teachers {
		teachersDTO[i] = schema.TeacherGetDTO{
			ID:          schema.RegularIDs(teacher.ID),
			FirstNames:  teacher.FirstNames,
			LastNames:   teacher.LastNames,
			PhoneNumber: teacher.PhoneNumber,
			Email:       teacher.Email,
			GovID:       teacher.GovID,
			Disciplines: helpers.MapDisciplines(teacher.Disciplines),
		}
	}
	return teachersDTO, nil

}

func (s *TeacherService) GetTeacherById(ctx *gin.Context) (schema.TeacherGetDTO, error) {
	var id = ctx.Param("id")
	teacherId, err := strconv.Atoi(id)

	if err != nil {
		return schema.TeacherGetDTO{}, fmt.Errorf("ID INVALID: %w", err)
	}
	var teacher schema.Teacher
	result := s.DB.Preload("Disciplines").First(&teacher, teacherId)
	if result.Error != nil {
		return schema.TeacherGetDTO{}, fmt.Errorf("profesor %d no encontrado: %w", teacherId, result.Error)
	}
	return schema.TeacherGetDTO{
		ID:          schema.RegularIDs(teacherId),
		FirstNames:  teacher.FirstNames,
		LastNames:   teacher.LastNames,
		PhoneNumber: teacher.PhoneNumber,
		Email:       teacher.Email,
		GovID:       teacher.GovID,
		Disciplines: helpers.MapDisciplines(teacher.Disciplines),
	}, nil
}

// govIDTaken indica si otro profesor distinto de excludeID ya usa esa cedula.
// Las cedulas vacias no se validan, igual que en atletas.
func (s *TeacherService) govIDTaken(govID string, excludeID uint) (bool, error) {
	if strings.TrimSpace(govID) == "" {
		return false, nil
	}

	query := s.DB.Model(&schema.Teacher{}).Where("gov_id = ?", govID)
	if excludeID != 0 {
		query = query.Where("id <> ?", excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *TeacherService) CreateTeacher(t schema.Teacher) (schema.Teacher, error) {

	if strings.TrimSpace(t.GovID) == "" {
		return schema.Teacher{}, ErrMissingGovID
	}

	taken, err := s.govIDTaken(t.GovID, 0)
	if err != nil {
		return schema.Teacher{}, err
	}
	if taken {
		return schema.Teacher{}, ErrDuplicateGovID
	}

	err = s.DB.Transaction(func(tx *gorm.DB) error {

		if err := tx.Set("gorm:save_associations", true).Create(&t).Error; err != nil {
			if isUniqueViolation(err) {
				return ErrDuplicateGovID
			}
			return err
		}
		return tx.Preload("Disciplines").Preload("Events").First(&t, t.ID).Error
	})

	if err != nil {
		return schema.Teacher{}, err
	}
	return t, nil
}

func (s *TeacherService) EditTeacher(ctx *gin.Context, t schema.Teacher) (schema.Teacher, error) {

	id := ctx.Param("id")
	teacherID, err := strconv.Atoi(id)
	var teacher schema.Teacher

	if err != nil {
		return schema.Teacher{}, fmt.Errorf("ID invalido: %v", err)
	}

	if err := s.DB.First(&teacher, teacherID).Error; err != nil {
		return schema.Teacher{}, fmt.Errorf("profesor no encontrado: %d", teacherID)
	}

	taken, err := s.govIDTaken(t.GovID, teacher.ID)
	if err != nil {
		return schema.Teacher{}, err
	}
	if taken {
		return schema.Teacher{}, ErrDuplicateGovID
	}

	//Actualizar campos escalares
	s.DB.Model(&teacher).Updates(&t)

	if len(t.Disciplines) > 0 {
		s.DB.Model(&teacher).Association("Disciplines").Replace(t.Disciplines)
	}
	if len(t.Events) > 0 {
		s.DB.Model(&teacher).Association("Events").Replace(t.Events)
	}

	return teacher, s.DB.Preload("Disciplines").Preload("Events").First(&teacher, teacherID).Error

}

func (s *TeacherService) DeleteTeacher(ctx *gin.Context) error {
	id := ctx.Param("id")
	teacherID, err := strconv.Atoi(id)
	if err != nil {
		return fmt.Errorf("ID inválido: %w", err)
	}

	var teacher schema.Teacher
	result := s.DB.First(&teacher, teacherID)
	if result.Error != nil {
		return fmt.Errorf("profesor %d no encontrado", teacherID)
	}

	result = s.DB.Delete(&schema.Teacher{}, teacherID)
	if result.Error != nil {
		return fmt.Errorf("error eliminando profesor %d: %w", teacherID, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("profesor %d no encontrado", teacherID)
	}
	return nil
}
