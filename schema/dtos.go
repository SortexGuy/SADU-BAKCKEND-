package schema

import (
	"time"

	"github.com/dgrijalva/jwt-go"
)

type LoginDTO struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type ChangePasswordDTO struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8"`
}

// ChangeUsernameDTO cambia el correo con el que se inicia sesion, que es la
// columna username: es la credencial que busca LoginUser. Se pide la contrasena
// actual porque cambiar la credencial de acceso desde una sesion abierta y sin
// confirmar nada dejaria al duenio de la cuenta fuera si el navegador queda solo.
//
// El formato del correo no se valida con `binding:"email"`: esa comprobacion
// corre antes de normalizar, asi que un correo pegado con un espacio delante se
// rechazaba con un mensaje generico. Lo valida el servicio, ya recortado.
type ChangeUsernameDTO struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewUsername     string `json:"newUsername" binding:"required"`
}

// UserProfileDTO es lo que la pantalla de perfil necesita saber de la sesion
// abierta. Nunca incluye la contrasena, ni siquiera cifrada.
type UserProfileDTO struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

type Claims struct {
	UserId   uint   `json:"user_id" binding:"required"`
	Username string `json:"username"`
	jwt.StandardClaims
}

type AthleteDTO struct {
	ID           RegularIDs `json:"ID"`
	GovID        string     `json:"GovID"`
	FirstNames   string     `json:"FirstNames"`
	LastNames    string     `json:"LastNames"`
	PhoneNumber  string     `json:"PhoneNumber"`
	Email        string     `json:"Email"`
	Gender       Gender     `json:"Gender"`
	Enrolled     bool       `json:"Enrolled"`
	Regular      bool       `json:"Regular"`
	Disciplines []DisciplineGetBareDTO `json:"Disciplines"`
}

type MajorGetDTO struct {
	ID   RegularIDs `json:"ID"`
	Name string     `json:"Name"`
}

type DisciplineGetBareDTO struct {
	ID   RegularIDs `json:"ID"`
	Name string     `json:"Name"`
}

type UniversityGetBareDTO struct {
	ID    RegularIDs `json:"ID"`
	Name  string     `json:"Name"`
	Local bool       `json:"Local"`
}

type TeamGetBareDTO struct {
	ID         RegularIDs           `json:"ID"`
	Name       string               `json:"Name" binding:"required"`
	Regular    bool                 `json:"Regular" binding:"required"`
	Category   string               `json:"Category"`
	University UniversityGetBareDTO `json:"University"`
	Athletes   []AthleteDTO         `json:"Athletes" binding:"required"`
}

type TeamGetDTO struct {
	ID           RegularIDs `json:"ID"`
	Name         string     `json:"Name" binding:"required"`
	Regular      bool       `json:"Regular" binding:"required"`
	Category     string     `json:"Category"`
	DisciplineID RegularIDs `json:"DisciplineID"`
	UniversityID RegularIDs `json:"UniversityID"`
	Athletes     []Athlete  `json:"Athletes" binding:"required"`
}

type TeamUpdateDTO struct {
	Name       *string   `json:"Name" binding:"omitempty,min=3"`
	Regular    *bool     `json:"Regular"`
	Category   *string   `json:"Category"`
	AthleteIDs []Athlete `json:"AthleteIDs"`
}

type TeamPostDTO struct {
	ID           RegularIDs   `json:"ID"`
	Name         string       `json:"Name" binding:"required"`
	Regular      bool         `json:"Regular" binding:"required"`
	Category     string       `json:"Category"`
	DisciplineID RegularIDs   `json:"DisciplineID"`
	UniversityID RegularIDs   `json:"UniversityID"`
	AthleteIDs   []RegularIDs `json:"AthleteIDs" binding:"required"`
}

type TeacherGetBareDTO struct {
	ID          RegularIDs             `json:"ID"`
	FirstNames  string                 `json:"FirstNames"`
	LastNames   string                 `json:"LastNames"`
	GovID       string                 `json:"GovID"`
	Disciplines []DisciplineGetBareDTO `json:"Disciplines"`
}

type TeacherGetDTO struct {
	ID          RegularIDs             `json:"ID"`
	FirstNames  string                 `json:"FirstNames"`
	LastNames   string                 `json:"LastNames"`
	PhoneNumber string                 `json:"PhoneNumber"`
	Email       string                 `json:"Email"`
	GovID       string                 `json:"GovID"`
	Disciplines []DisciplineGetBareDTO `json:"Disciplines"`
	// Events      []Event      `json:"events"`
}
type TeacherCreateDTO struct {
	FirstNames    string       `json:"FirstNames" binding:"required,min=2"`
	LastNames     string       `json:"LastNames" binding:"required,min=2"`
	PhoneNumber   string       `json:"PhoneNumber"`
	Email         string       `json:"Email" binding:"omitempty,email"`
	GovID         string       `json:"GovID" binding:"required,len=8"`
	DisciplineIDs []RegularIDs `json:"DisciplineIDs"`
}

type TourneyGetBareDTO struct {
	ID             RegularIDs `json:"ID"`
	Name           string     `json:"Name"`
	Status         Status     `json:"Status"`
	StartDate      time.Time  `json:"StartDate"`
	EndDate        time.Time  `json:"EndDate"`
	TotalEvents    uint       `json:"TotalEvents"`
	DisciplineID   RegularIDs `json:"DisciplineID"`
	DisciplineName string     `json:"DisciplineName,omitempty"`
}

type TourneyPOSTandPUTDTO struct {
	Name   string `json:"Name"`
	Status Status `json:"Status"`
	// Los partidos del torneo, por identificador. La etiqueta sigue la convencion
	// del resto del archivo (AthleteIDs, DisciplineIDs) y es la que envia el
	// cliente. Una lista vacia en un PUT significa "sin partidos"; su ausencia,
	// "no toques los partidos" (ver UpdateTourney).
	Events       []RegularIDs `json:"EventIDs"`
	StartDate    time.Time    `json:"StartDate"`
	EndDate      time.Time    `json:"EndDate"`
	DisciplineID RegularIDs   `json:"DisciplineID"`
}

type TourneyGetFullDTO struct {
	ID             RegularIDs    `json:"ID"`
	Name           string        `json:"Name"`
	Status         Status        `json:"Status"`
	Events         []EventGetDTO `json:"Events"`
	StartDate      time.Time     `json:"StartDate"`
	EndDate        time.Time     `json:"EndDate"`
	TotalEvents    uint          `json:"TotalEvents"`
	DisciplineID   RegularIDs    `json:"DisciplineID"`
	DisciplineName string        `json:"DisciplineName,omitempty"`
}

type EventGetDTO struct {
	ID                 RegularIDs           `json:"ID"`
	Name               string               `json:"Name"`
	Date               time.Time            `json:"Date"`
	Status             string               `json:"Status"`
	Observation        string               `json:"Observation"`
	Ubication          string               `json:"Ubication"`
	HomePoints         uint8                `json:"HomePoints"`
	OppositePoints     uint8                `json:"OppositePoints"`
	HomeTeam           TeamGetBareDTO       `json:"HomeTeam"`
	OppositeTeam       TeamGetBareDTO       `json:"OppositeTeam"`
	Tourney            TourneyGetBareDTO    `json:"Tourney"`
	ResponsableTeacher TeacherGetBareDTO    `json:"ResponsableTeacher"`
	Discipline         DisciplineGetBareDTO `json:"Discipline"`
}

type EventPOSTandPUTDTO struct {
	Name           string     `json:"Name"`
	Date           time.Time  `json:"Date"`
	Status         string     `json:"Status"`
	Observation    string     `json:"Observation"`
	Ubication      string     `json:"Ubication"`
	HomePoints     uint8      `json:"HomePoints"`
	OppositePoints uint8      `json:"OppositePoints"`
	HomeTeamID     RegularIDs `json:"HomeTeamID" binding:"required"`
	OppositeTeamID RegularIDs `json:"OppositeTeamID" binding:"required"`
	// El torneo es opcional y se distingue por puntero: nil es "no vino en la
	// peticion" (al editar, la columna no se toca) y 0 es "sin torneo" (se guarda
	// NULL). Sin esa distincion, editar un partido lo sacaba de su torneo.
	TourneyID            *RegularIDs `json:"TourneyID"`
	ResponsableTeacherID RegularIDs  `json:"ResponsableTeacherID" binding:"required"`
	DisciplineID         RegularIDs  `json:"DisciplineID" binding:"required"`
}
