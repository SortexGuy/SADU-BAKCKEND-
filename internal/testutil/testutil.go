// Package testutil reune los ayudantes que comparten las pruebas: creacion de
// una base de datos aislada, datos base y aserciones cortas.
//
// No se usa en produccion; vive en internal/ para que no forme parte de la
// superficie publica del modulo.
package testutil

import (
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"uneg.edu.ve/servicio-sadu-back/config"
	"uneg.edu.ve/servicio-sadu-back/schema"
)

// NuevaDB crea una base de datos SQLite vacia y aislada para la prueba, con el
// esquema completo y las claves foraneas verificadas, igual que en produccion.
//
// Usa un archivo temporal en lugar de :memory: porque GORM abre varias conexiones
// del pool y una base en memoria sin cache compartida daria una base distinta por
// conexion. El archivo lo borra el propio testing al terminar (t.TempDir).
//
// Tambien asigna config.DB, porque algunos servicios —el listado de atletas— usan
// esa variable global en lugar de recibir la conexion.
func NuevaDB(t *testing.T) *gorm.DB {
	t.Helper()

	ruta := filepath.Join(t.TempDir(), "prueba.db")
	db, err := gorm.Open(sqlite.Open(ruta+"?_foreign_keys=on"), &gorm.Config{
		// Silencioso: las pruebas no deben ensuciar la salida con SQL.
		Logger: gormlogger.Discard,
	})
	if err != nil {
		t.Fatalf("no se pudo abrir la base de prueba: %v", err)
	}

	anterior := config.DB
	config.DB = db
	t.Cleanup(func() { config.DB = anterior })

	if err := config.SyncDB(); err != nil {
		t.Fatalf("no se pudo migrar el esquema de prueba: %v", err)
	}

	verificarClavesForaneas(t, db)
	return db
}

// verificarClavesForaneas confirma que la base de prueba se comporta como la de
// produccion. Si el pragma no quedara activo, las pruebas de integridad pasarian
// por el motivo equivocado.
func verificarClavesForaneas(t *testing.T, db *gorm.DB) {
	t.Helper()

	var activas int
	if err := db.Raw("PRAGMA foreign_keys").Scan(&activas).Error; err != nil {
		t.Fatalf("no se pudo consultar PRAGMA foreign_keys: %v", err)
	}
	if activas != 1 {
		t.Fatal("la base de prueba no esta verificando las claves foraneas: " +
			"las pruebas de integridad no serian validas")
	}
}

// SilenciarLogs descarta la salida del registro durante la prueba. Se usa en las
// pruebas que provocan errores a proposito, para que el ruido esperado no se
// confunda con un fallo real al leer la salida.
func SilenciarLogs(t *testing.T) {
	t.Helper()

	anterior := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(func() { slog.SetDefault(anterior) })
}

// ── Datos base ──────────────────────────────────────────────────────────────
//
// Las entidades del dominio dependen unas de otras (un atleta necesita carrera,
// un equipo necesita disciplina y universidad), y con las claves foraneas activas
// hay que crearlas en orden. Estos ayudantes evitan repetir ese armado.

// Catalogo son los identificadores de los registros base que crea SembrarCatalogo.
type Catalogo struct {
	CarreraID     schema.RegularIDs
	UniversidadID schema.RegularIDs
	Disciplina1ID schema.RegularIDs
	Disciplina2ID schema.RegularIDs
	ProfesorID    schema.RegularIDs
	EquipoLocalID schema.RegularIDs
	EquipoVisitID schema.RegularIDs
	TorneoID      schema.RegularIDs
}

// SembrarCatalogo crea los registros minimos sobre los que se puede montar
// cualquier prueba: una carrera, una universidad, dos disciplinas, un profesor,
// dos equipos y un torneo.
func SembrarCatalogo(t *testing.T, db *gorm.DB) Catalogo {
	t.Helper()

	carrera := schema.Major{Name: "Informatica"}
	universidad := schema.University{Name: "UNEG", Local: true}
	disciplina1 := schema.Discipline{Name: "Voleibol"}
	disciplina2 := schema.Discipline{Name: "Futbol"}

	for _, registro := range []any{&carrera, &universidad, &disciplina1, &disciplina2} {
		if err := db.Create(registro).Error; err != nil {
			t.Fatalf("no se pudo crear un registro del catalogo: %v", err)
		}
	}

	profesor := schema.Teacher{FirstNames: "Luis", LastNames: "Ramirez", GovID: "20000001"}
	if err := db.Create(&profesor).Error; err != nil {
		t.Fatalf("no se pudo crear el profesor del catalogo: %v", err)
	}

	local := schema.Team{
		Name: "UNEG Alfa", Regular: true, Category: schema.GenderM,
		DisciplineID: schema.RegularIDs(disciplina1.ID),
		UniversityID: schema.RegularIDs(universidad.ID),
	}
	visitante := schema.Team{
		Name: "UNEG Beta", Regular: false, Category: schema.GenderF,
		DisciplineID: schema.RegularIDs(disciplina2.ID),
		UniversityID: schema.RegularIDs(universidad.ID),
	}
	for _, equipo := range []*schema.Team{&local, &visitante} {
		if err := db.Create(equipo).Error; err != nil {
			t.Fatalf("no se pudo crear un equipo del catalogo: %v", err)
		}
	}

	torneo := schema.Tourney{
		Name: "Copa UNEG", Status: schema.StatusON,
		DisciplineID: schema.RegularIDs(disciplina1.ID),
	}
	if err := db.Create(&torneo).Error; err != nil {
		t.Fatalf("no se pudo crear el torneo del catalogo: %v", err)
	}

	return Catalogo{
		CarreraID:     schema.RegularIDs(carrera.ID),
		UniversidadID: schema.RegularIDs(universidad.ID),
		Disciplina1ID: schema.RegularIDs(disciplina1.ID),
		Disciplina2ID: schema.RegularIDs(disciplina2.ID),
		ProfesorID:    schema.RegularIDs(profesor.ID),
		EquipoLocalID: schema.RegularIDs(local.ID),
		EquipoVisitID: schema.RegularIDs(visitante.ID),
		TorneoID:      schema.RegularIDs(torneo.ID),
	}
}

// ── Aserciones ──────────────────────────────────────────────────────────────

// SinError falla la prueba si err no es nil.
func SinError(t *testing.T, err error, contexto string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: se esperaba exito y hubo error: %v", contexto, err)
	}
}

// Igual falla la prueba si obtenido y esperado difieren.
func Igual[T comparable](t *testing.T, obtenido, esperado T, contexto string) {
	t.Helper()
	if obtenido != esperado {
		t.Errorf("%s: se obtuvo %v y se esperaba %v", contexto, obtenido, esperado)
	}
}
