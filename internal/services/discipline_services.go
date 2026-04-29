package services

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"uneg.edu.ve/servicio-sadu-back/config"
	"uneg.edu.ve/servicio-sadu-back/schema"
)

type DisciplineServices struct {
	DB *gorm.DB
}

func NewDisciplineServices() *DisciplineServices {
	return &DisciplineServices{DB: config.DB}
}

func (d *DisciplineServices) GetAllDisciplines(name string) ([]schema.DisciplineGetBareDTO, error) {
	var disciplines []schema.Discipline
	query := d.DB.Order("name").Preload("Teams", nil)

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	if err := query.Find(&disciplines).Error; err != nil {
		return nil, err
	}
	disciplinesDTO := []schema.DisciplineGetBareDTO{}

	for _, value := range disciplines {
		disciplinesDTO = append(disciplinesDTO, schema.DisciplineGetBareDTO{
			ID:   schema.RegularIDs(value.ID),
			Name: value.Name,
		})
	}
	return disciplinesDTO, nil
}

func (d *DisciplineServices) GetAllDisciplinesByID(ctx *gin.Context) (schema.Discipline, error) {
	var id = ctx.Param("id")
	disciplineID, err := strconv.Atoi(id)
	if err != nil {
		return schema.Discipline{}, fmt.Errorf("ID invalido | %v", err)
	}
	var discipline schema.Discipline
	if result := d.DB.Preload("Teams.University").Preload("Teams").Preload("Events").Preload("Athletes").Preload("Teachers").First(&discipline, disciplineID).Error; result != nil {
		return schema.Discipline{}, fmt.Errorf("Discipline not found | %w", result)
	}

	return discipline, nil
}

func (d *DisciplineServices) CreateDiscipline(dis schema.Discipline) (schema.Discipline, error) {

	if err := d.DB.Omit("Teams", "Events", "Athletes", "Teachers").Create(&dis).Error; err != nil {
		return dis, err
	}
	if len(dis.Athletes) > 0 {
		d.DB.Model(&dis).Association("Athletes").Append(dis.Athletes)
	}

	if len(dis.Teams) > 0 {
		d.DB.Model(&dis).Association("Teams").Append(dis.Teams)
	}

	if len(dis.Events) > 0 {
		d.DB.Model(&dis).Association("Events").Append(dis.Events)
	}
	return dis, nil
}

func (d *DisciplineServices) EditDiscipline(dis schema.Discipline, ctx *gin.Context) (schema.Discipline, error) {

	var id = ctx.Param("id")
	disciplineID, err := strconv.Atoi(id)
	if err != nil {
		return schema.Discipline{}, fmt.Errorf("ID invalido | %w", err)
	}
	var discipline schema.Discipline
	if err := d.DB.First(&discipline, disciplineID).Error; err != nil {
		return schema.Discipline{}, fmt.Errorf("disciplina no encontrada: %w", err)
	}
	d.DB.Model(&discipline).Select("Name").Updates(&dis)

	if len(dis.Teams) > 0 {
		d.DB.Model(&discipline).Association("Teams").Replace(dis.Teams)
	}

	if len(dis.Athletes) > 0 {
		d.DB.Model(&discipline).Association("Athletes").Replace(dis.Athletes)
	}
	if len(dis.Events) > 0 {
		d.DB.Model(&discipline).Association("Events").Replace(dis.Events)
	}

	if len(dis.Teachers) > 0 {
		d.DB.Model(&discipline).Association("Teachers").Replace(dis.Teachers)
	}

	return discipline, d.DB.Preload("Teams").Preload("Athletes").Preload("Events").Preload("Teachers").First(&discipline, disciplineID).Error
}

func (d *DisciplineServices) DeleteDiscipline(ctx *gin.Context) error {
	var id = ctx.Param("id")
	disciplineID, err := strconv.Atoi(id)
	if err != nil {
		return fmt.Errorf("ID invalid: %w", err)
	}
	result := d.DB.Delete(&schema.Discipline{}, disciplineID)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("Column not affected")
	}
	return nil
}
