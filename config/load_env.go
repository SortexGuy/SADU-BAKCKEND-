package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	if err := godotenv.Load(".env"); err != nil {
		log.Print("Please create a .env file in the root directory of the project")
	}
}

var (
	jwtSecret       []byte
	jwtSecretLoaded bool
)

// SecretKey devuelve la clave con la que se firman y validan los JWT.
// Se lee de las variables de entorno la primera vez que se invoca, así que
// debe llamarse siempre despues de LoadEnv(). Si SECRET_KEY no esta definida
// devuelve una clave vacia: quien la use debe verificarlo (main.go aborta el
// arranque en ese caso) en lugar de recurrir a un valor por defecto.
func SecretKey() []byte {
	if !jwtSecretLoaded {
		jwtSecret = []byte(os.Getenv("SECRET_KEY"))
		jwtSecretLoaded = true
	}
	return jwtSecret
}
