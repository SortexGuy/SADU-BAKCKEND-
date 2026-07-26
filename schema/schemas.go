package schema

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username    string
	DisplayName string
	Email       string
	Password    string
	Description string
}

// Status represents the possible states of an item
type Gender string
type Status string

const (
	GenderM Gender = "Masculino"
	GenderF Gender = "Femenino"

	StatusON   Status = "Activo"
	StatusOFF  Status = "Finalizado"
	StatusWait Status = "Pendiente"
)

// ID uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()"`
type Major struct { // Carrera
	gorm.Model
	Name     string
	Athletes []Athlete `gorm:"foreignKey:MajorID"`
}

type University struct {
	gorm.Model
	Name  string
	Local bool
	Teams []Team `gorm:"foreignKey:UniversityID"`
}

type Team struct {
	gorm.Model
	Name          string
	Regular       bool
	Category      Gender `gorm:"index;type:gender"`
	DisciplineID  RegularIDs
	UniversityID  RegularIDs
	Discipline    Discipline     `gorm:"foreignKey:DisciplineID"`
	University    University     `gorm:"foreignKey:UniversityID"`
	AthleteEvents []AthleteEvent `gorm:"foreignKey:TeamID"`
	Athletes      []Athlete      `gorm:"many2many:athlete_teams;"`
}

type Athlete struct {
	gorm.Model
	FirstNames      string
	LastNames       string
	PhoneNumber     string
	Email           string
	Enrolled        bool
	Gender          Gender    `gorm:"index;type:gender"`
	InscriptionDate time.Time // Fecha de inscripcion
	Regular         bool      // Titular
	GovID           string    `gorm:"unique;not null"` // Cedula
	MajorID         RegularIDs
	Teams           []Team       `gorm:"many2many:athlete_teams;"`
	Events          []Event      `gorm:"many2many:athlete_events;"` // foreignKey:AthleteID
	Disciplines     []Discipline `gorm:"many2many:athlete_disciplines;"`
}

// disciplina deportiva
type Discipline struct {
	gorm.Model
	Name     string
	Teams    []Team    `gorm:"foreignKey:DisciplineID"`
	Events   []Event   `gorm:"foreignKey:DisciplineID"`
	Athletes []Athlete `gorm:"many2many:athlete_disciplines;"`
	Teachers []Teacher `gorm:"many2many:teacher_disciplines;"`
	// La relacion con Tourney es uno-a-muchos y vive en Tourney.DisciplineID.
	// Aqui estaba declarada ademas como `many2many:tourney_disicplines`, lo que
	// creaba una tabla puente que nadie consultaba ni escribia. Se elimina sin
	// declarar la relacion inversa para no agregar restricciones nuevas a la
	// migracion. Ver INFORME_TECNICO.md (D9).
}

type Teacher struct {
	gorm.Model
	FirstNames  string
	LastNames   string
	PhoneNumber string
	Email       string
	GovID       string       `gorm:"unique;not null"` // Cedula
	Events      []Event      `gorm:"foreignKey:ResponsableTeacherID"`
	Disciplines []Discipline `gorm:"many2many:teacher_disciplines;"`
}

type Tourney struct {
	gorm.Model
	Name         string
	Status       Status  `gorm:"index;type:status"`
	Events       []Event `gorm:"foreignKey:TourneyID"`
	StartDate    time.Time
	EndDate      time.Time
	DisciplineID RegularIDs `json:"DisciplineID"`
	Discipline   Discipline `gorm:"foreignKey:DisciplineID"`
}

type Event struct {
	gorm.Model
	Name                 string
	Date                 time.Time
	Status               string
	Observation          string
	Ubication            string
	HomePoints           uint8 `gorm:"column:home_points"`
	OppositePoints       uint8 `gorm:"column:opposite_points"`
	HomeTeamID           RegularIDs
	OppositeTeamID       RegularIDs
	TourneyID            RegularIDs `json:"TourneyID"`
	ResponsableTeacherID RegularIDs
	DisciplineID         RegularIDs `json:"DisciplineID"`
	HomeTeam             Team       `gorm:"foreignKey:HomeTeamID"`
	OppositeTeam         Team       `gorm:"foreignKey:OppositeTeamID"`
	Tourney              Tourney    `gorm:"foreignKey:TourneyID"`
	ResponsableTeacher   Teacher    `gorm:"foreignKey:ResponsableTeacherID"`
	Discipline           Discipline `gorm:"foreignKey:DisciplineID"`
	Athletes             []Athlete  `gorm:"many2many:athlete_events;"`
}

type AthleteDiscipline struct {
	AthleteID    RegularIDs `gorm:"primaryKey;"`
	DisciplineID RegularIDs `gorm:"primaryKey;"`
	Regular      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time `gorm:"index"`
	DeletedAt    gorm.DeletedAt
}

// err := db.SetupJoinTable(&Athlete{}, "Discipline", &AthleteDisciplines{})

type AthleteTeam struct {
	AthleteID RegularIDs `gorm:"primaryKey"`
	TeamID    RegularIDs `gorm:"primaryKey"`
	StartDate time.Time
	EndDate   time.Time `gorm:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
}

// err := db.SetupJoinTable(&Athlete{}, "Team", &AthleteTeam{})

type TeacherDiscipline struct {
	TeacherID    RegularIDs `gorm:"primaryKey"`
	DisciplineID RegularIDs `gorm:"primaryKey"`
	StartDate    time.Time
	EndDate      time.Time `gorm:"index"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt
}

// err := db.SetupJoinTable(&Teacher{}, "Discipline", &TeacherDisciplines{})

type AthleteEvent struct {
	AthleteID RegularIDs `gorm:"primaryKey"`
	EventID   RegularIDs `gorm:"primaryKey;index"`
	TeamID    RegularIDs
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
}

// err := db.SetupJoinTable(&Athlete{}, "Event", &AthleteEvent{})
