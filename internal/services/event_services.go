package services

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"uneg.edu.ve/servicio-sadu-back/config"
	"uneg.edu.ve/servicio-sadu-back/helpers"
	"uneg.edu.ve/servicio-sadu-back/schema"
)

// ajustarRangoDelTorneo amplia las fechas del torneo para que cubran la del
// partido: si el partido cae antes del inicio, el inicio se adelanta; si cae
// despues del fin, el fin se atrasa. Un torneo sin fechas toma la del partido en
// las dos, que es el mismo criterio con el que el formulario de torneos las
// sugiere a partir de los partidos elegidos.
//
// El rango solo se amplia, nunca se encoge: recortarlo al mover o desvincular un
// partido borraria fechas puestas a mano, y ademas dejaria fuera a los otros
// partidos del torneo.
//
// Las fechas del torneo se guardan con granularidad de dia (el formulario manda
// un <input type="date">, medianoche UTC), asi que la del partido se trunca antes
// de compararla; si no, la hora del ultimo partido terminaria dentro de EndDate.
func ajustarRangoDelTorneo(tx *gorm.DB, tourneyID schema.RegularIDs, fecha time.Time) error {
	if tourneyID == 0 || fecha.IsZero() {
		return nil
	}

	var torneo schema.Tourney
	if err := tx.First(&torneo, tourneyID).Error; err != nil {
		return fmt.Errorf("torneo del partido no encontrado: %w", err)
	}

	utc := fecha.UTC()
	dia := time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)

	campos := map[string]interface{}{}
	if torneo.StartDate.IsZero() || dia.Before(torneo.StartDate) {
		campos["start_date"] = dia
	}
	if torneo.EndDate.IsZero() || dia.After(torneo.EndDate) {
		campos["end_date"] = dia
	}
	if len(campos) == 0 {
		return nil
	}

	if err := tx.Model(&torneo).Updates(campos).Error; err != nil {
		return fmt.Errorf("ajustando las fechas del torneo: %w", err)
	}

	slog.Info("fechas del torneo ajustadas al partido",
		"torneo_id", torneo.ID,
		"fecha_partido", dia,
		"campos", campos,
	)
	return nil
}

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
		ResponsableTeacherID: dto.ResponsableTeacherID,
		DisciplineID:         dto.DisciplineID,
	}
	if dto.TourneyID != nil {
		event.TourneyID = *dto.TourneyID
	}

	tx := s.DB.Begin()

	// El torneo es opcional: un partido puede existir sin pertenecer a ninguno. Si
	// no viene, o viene en 0, la columna se omite para que quede en NULL, porque un
	// 0 no referencia a ningun torneo y la clave foranea lo rechaza.
	omitir := []string{"HomeTeam", "OppositeTeam", "Tourney", "ResponsableTeacher", "Discipline", "Athletes"}
	if event.TourneyID == 0 {
		omitir = append(omitir, "TourneyID")
	}

	if err := tx.Omit(omitir...).Create(&event).Error; err != nil {
		tx.Rollback()
		return schema.Event{}, err
	}

	// Un partido que entra al torneo lo estira si cae fuera de sus fechas. Va en la
	// misma transaccion que el alta: o quedan los dos escritos o ninguno.
	if err := ajustarRangoDelTorneo(tx, event.TourneyID, event.Date); err != nil {
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

	slog.Info("evento creado",
		"evento_id", event.ID,
		"equipo_local_id", event.HomeTeamID,
		"equipo_visitante_id", event.OppositeTeamID,
		"torneo_id", event.TourneyID,
		"disciplina_id", event.DisciplineID,
	)
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
	// El torneo tiene tres casos, y por eso el DTO lo declara como puntero:
	//
	//   ausente (nil) la peticion no habla del torneo, la columna no se toca. Es lo
	//                 que evita que editar el marcador de un partido lo saque del
	//                 torneo al que pertenece.
	//   0             se pide quitarlo del torneo: se guarda NULL, no 0, porque un 0
	//                 no referencia a ningun torneo y la clave foranea lo rechaza.
	//   valor         se asigna a ese torneo.
	if dto.TourneyID != nil {
		if *dto.TourneyID == 0 {
			campos["TourneyID"] = nil
		} else {
			campos["TourneyID"] = *dto.TourneyID
		}
	}

	// El torneo al que queda vinculado el partido se calcula antes de guardar:
	// despues, `existingEvent` ya trae los valores nuevos y no se distingue de los
	// viejos.
	torneoEfectivo := existingEvent.TourneyID
	if dto.TourneyID != nil {
		torneoEfectivo = *dto.TourneyID
	}

	tx := s.DB.Begin()
	err := tx.Model(&existingEvent).Omit("Athletes").Updates(campos).Error

	if err != nil {
		tx.Rollback()
		return schema.Event{}, err
	}

	// Cambiar la fecha del partido, o moverlo a otro torneo, tambien estira el rango
	// del torneo destino. El torneo del que sale no se toca: encogerlo borraria
	// fechas puestas a mano y dejaria fuera a sus otros partidos.
	if err := ajustarRangoDelTorneo(tx, torneoEfectivo, dto.Date); err != nil {
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
