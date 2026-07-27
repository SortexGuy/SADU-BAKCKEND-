package services

import (
	"errors"
	"strings"
)

// ErrDuplicateGovID indica que la cedula ya esta registrada en otro registro del
// mismo tipo (atleta o profesor). Los handlers la traducen a HTTP 409.
//
// La unicidad se valida en dos niveles: en la capa de servicio, para poder dar un
// mensaje claro, y con una restriccion UNIQUE en la base, que es la garantia real.
var ErrDuplicateGovID = errors.New("la cedula ya esta registrada")

// ErrMissingGovID indica que se intento crear un registro sin cedula. La columna
// es NOT NULL y UNIQUE, asi que dos registros sin cedula colisionarian entre si.
// Los handlers la traducen a HTTP 400.
var ErrMissingGovID = errors.New("la cedula es obligatoria")

// ErrMissingMajor indica que se intento crear un atleta sin carrera. La columna
// major_id no admite NULL y tiene clave foranea, asi que un 0 no referencia a
// ninguna carrera real. Los handlers la traducen a HTTP 400.
var ErrMissingMajor = errors.New("la carrera es obligatoria")

// ErrMissingDiscipline indica que se intento crear un torneo sin disciplina. La
// columna discipline_id no admite NULL y tiene clave foranea, asi que un 0 no
// referencia a ninguna disciplina real. Los handlers la traducen a HTTP 400.
var ErrMissingDiscipline = errors.New("la disciplina es obligatoria")

// ErrInvalidDateRange indica que la fecha de fin es anterior a la de inicio. Un
// rango invertido dejaria el torneo sin duracion representable. Los handlers la
// traducen a HTTP 400.
var ErrInvalidDateRange = errors.New("la fecha de fin no puede ser anterior a la de inicio")

// isUniqueViolation reconoce el error que devuelve SQLite cuando se viola la
// restriccion UNIQUE. Actua como red de seguridad: si una cedula duplicada se
// escapa de la validacion previa, el cliente recibe 409 en lugar de un 500 opaco.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// IsInvalidReference indica que la operacion violo una clave foranea: se apunto a
// un registro que no existe, o se omitio una relacion obligatoria y quedo en 0.
// Los handlers la traducen a HTTP 400 en lugar de un 500 opaco.
//
// Se comprueba sobre el mensaje porque el driver de SQLite no expone un tipo de
// error especifico, y se centraliza aqui para no repetir la cadena en cada handler.
func IsInvalidReference(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToUpper(err.Error()), "FOREIGN KEY CONSTRAINT FAILED")
}
