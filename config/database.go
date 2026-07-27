package config

import (
	"log/slog"
	"os"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"uneg.edu.ve/servicio-sadu-back/internal/logging"
	"uneg.edu.ve/servicio-sadu-back/schema"
)

var DB *gorm.DB

func ConnectDB() {
	var err error

	// El logger de GORM se enruta a slog: los errores de base de datos y las
	// consultas lentas salen en el mismo flujo estructurado que el resto.
	configuracion := &gorm.Config{Logger: logging.NuevoGORM()}

	tursoUrl := os.Getenv("TURSO_DATABASE_URL")
	tursoToken := os.Getenv("TURSO_AUTH_TOKEN")

	if tursoUrl != "" && tursoToken != "" {
		dsn := tursoUrl + "?authToken=" + tursoToken
		DB, err = gorm.Open(sqlite.Dialector{
			DriverName: "libsql",
			DSN:        dsn,
		}, configuracion)
		slog.Info("base de datos conectada", "destino", "turso", "url", tursoUrl)
	} else {
		dbPath := os.Getenv("DATABASE_PATH")
		if dbPath == "" {
			dbPath = "database.db"
		}
		// _foreign_keys=on hace que SQLite verifique las claves foraneas. Sin el,
		// las restricciones existen en el DDL pero no se aplican en ejecucion.
		// Va en el DSN y no como PRAGMA suelto porque el pool abre varias
		// conexiones y el PRAGMA solo afectaria a una.
		DB, err = gorm.Open(sqlite.Open(dbPath+"?_foreign_keys=on"), configuracion)
		slog.Info("base de datos conectada", "destino", "sqlite local", "ruta", dbPath)
	}

	if err != nil {
		slog.Error("no se pudo conectar a la base de datos", "error", err.Error())
		os.Exit(1)
	}

	logForeignKeyEnforcement()
}

// logForeignKeyEnforcement deja constancia en el arranque de si la base esta
// verificando las claves foraneas. En Turso el pragma depende del servidor, asi
// que conviene poder confirmarlo desde los logs del despliegue.
func logForeignKeyEnforcement() {
	var enabled int
	if err := DB.Raw("PRAGMA foreign_keys").Scan(&enabled).Error; err != nil {
		slog.Warn("no se pudo consultar PRAGMA foreign_keys", "error", err.Error())
		return
	}
	if enabled == 1 {
		slog.Info("integridad referencial activa", "foreign_keys", "on")
	} else {
		slog.Warn("integridad referencial inactiva: la base no verificara las claves foraneas",
			"foreign_keys", "off")
	}
}

func SyncDB() error {
	models := []any{
		&schema.User{},
		&schema.University{},
		&schema.Athlete{},
		&schema.Major{},
		&schema.Discipline{},
		&schema.Event{},
		&schema.Tourney{},
		&schema.Teacher{},
		&schema.Team{},
		&schema.AthleteDiscipline{},
		&schema.AthleteEvent{},
		&schema.AthleteTeam{},
		&schema.TeacherDiscipline{},
	}

	if err := DB.AutoMigrate(models...); err != nil {
		slog.Error("fallo la migracion del esquema", "error", err.Error())
		return err
	}

	// Las cuatro tablas puente llevan columnas propias, asi que se registran
	// explicitamente: si no, GORM crearia una tabla puente vacia.
	puentes := []struct {
		modelo any
		campo  string
		puente any
	}{
		{&schema.Athlete{}, "Disciplines", &schema.AthleteDiscipline{}},
		{&schema.Athlete{}, "Teams", &schema.AthleteTeam{}},
		{&schema.Teacher{}, "Disciplines", &schema.TeacherDiscipline{}},
		{&schema.Athlete{}, "Events", &schema.AthleteEvent{}},
	}

	for _, p := range puentes {
		if err := DB.SetupJoinTable(p.modelo, p.campo, p.puente); err != nil {
			slog.Error("no se pudo registrar la tabla puente", "campo", p.campo, "error", err.Error())
			return err
		}
	}

	slog.Info("esquema sincronizado",
		"modelos", len(models),
		"tablas_puente", len(puentes),
	)
	return nil
}
