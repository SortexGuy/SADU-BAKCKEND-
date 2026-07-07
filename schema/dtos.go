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
	Name         string       `json:"Name"`
	Status       Status       `json:"Status"`
	Events       []RegularIDs `json:"EventsIDs"`
	StartDate    time.Time    `json:"StartDate"`
	EndDate      time.Time    `json:"EndDate"`
	DisciplineID RegularIDs   `json:"DisciplineID"`
}

type TourneyGetFullDTO struct {
	ID        RegularIDs    `json:"ID"`
	Name      string        `json:"Name"`
	Status    Status        `json:"Status"`
	Events    []EventGetDTO `json:"Events"`
	StartDate time.Time
	EndDate   time.Time
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
	Name                 string     `json:"Name"`
	Date                 time.Time  `json:"Date"`
	Status               string     `json:"Status"`
	Observation          string     `json:"Observation"`
	Ubication            string     `json:"Ubication"`
	HomePoints           uint8      `json:"HomePoints"`
	OppositePoints       uint8      `json:"OppositePoints"`
	HomeTeamID           RegularIDs `json:"HomeTeamID" binding:"required"`
	OppositeTeamID       RegularIDs `json:"OppositeTeamID" binding:"required"`
	TourneyID            RegularIDs `json:"TourneyID"`
	ResponsableTeacherID RegularIDs `json:"ResponsableTeacherID" binding:"required"`
	DisciplineID         RegularIDs `json:"DisciplineID" binding:"required"`
}
