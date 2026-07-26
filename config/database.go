package config

import (
	"log"
	"os"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"uneg.edu.ve/servicio-sadu-back/schema"
)

var DB *gorm.DB

func ConnectDB() {
	var err error

	tursoUrl := os.Getenv("TURSO_DATABASE_URL")
	tursoToken := os.Getenv("TURSO_AUTH_TOKEN")

	if tursoUrl != "" && tursoToken != "" {
		dsn := tursoUrl + "?authToken=" + tursoToken
		DB, err = gorm.Open(sqlite.Dialector{
			DriverName: "libsql",
			DSN:        dsn,
		}, &gorm.Config{})
		log.Default().Printf("Using Turso database at: %s\n", tursoUrl)
	} else {
		dbPath := os.Getenv("DATABASE_PATH")
		if dbPath == "" {
			dbPath = "database.db"
		}
		// _foreign_keys=on hace que SQLite verifique las claves foraneas. Sin el,
		// las restricciones existen en el DDL pero no se aplican en ejecucion.
		// Va en el DSN y no como PRAGMA suelto porque el pool abre varias
		// conexiones y el PRAGMA solo afectaria a una.
		DB, err = gorm.Open(sqlite.Open(dbPath+"?_foreign_keys=on"), &gorm.Config{})
		log.Default().Printf("Using local database at: %s\n", dbPath)
	}

	if err != nil {
		log.Fatal(err)
	}

	logForeignKeyEnforcement()
}

// logForeignKeyEnforcement deja constancia en el arranque de si la base esta
// verificando las claves foraneas. En Turso el pragma depende del servidor, asi
// que conviene poder confirmarlo desde los logs del despliegue.
func logForeignKeyEnforcement() {
	var enabled int
	if err := DB.Raw("PRAGMA foreign_keys").Scan(&enabled).Error; err != nil {
		log.Default().Printf("No se pudo consultar PRAGMA foreign_keys: %v\n", err)
		return
	}
	if enabled == 1 {
		log.Default().Println("Integridad referencial: ACTIVA (foreign_keys = ON)")
	} else {
		log.Default().Println("Integridad referencial: INACTIVA (foreign_keys = OFF)")
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

	DB.AutoMigrate(models...)

	if err := DB.SetupJoinTable(&schema.Athlete{}, "Disciplines", &schema.AthleteDiscipline{}); err != nil {
		return err
	}
	log.Println("Setup AthleteDiscilpines seeded successfully")

	if err := DB.SetupJoinTable(&schema.Athlete{}, "Teams", &schema.AthleteTeam{}); err != nil {
		return err
	}
	log.Println("Setup AthleteTeam seeded successfully")

	if err := DB.SetupJoinTable(&schema.Teacher{}, "Disciplines", &schema.TeacherDiscipline{}); err != nil {
		return err
	}
	log.Println("Setup TeacherDisciplines seeded successfully")

	if err := DB.SetupJoinTable(&schema.Athlete{}, "Events", &schema.AthleteEvent{}); err != nil {
		return err
	}
	log.Println("Setup AthleteEvent seeded successfully")
	return nil
}
