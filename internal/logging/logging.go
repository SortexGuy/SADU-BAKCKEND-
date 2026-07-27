// Package logging centraliza la configuracion del registro de eventos del
// servidor. Usa log/slog de la biblioteca estandar, asi que no agrega
// dependencias, y produce registros estructurados con campos en lugar de texto
// libre: eso permite filtrar por nivel, por recurso o por peticion desde los
// logs del despliegue.
//
// Configuracion por variables de entorno:
//
//	LOG_LEVEL   debug | info | warn | error   (por defecto info)
//	LOG_FORMAT  text | json                   (por defecto json si GIN_MODE=release, text si no)
//
// Con LOG_LEVEL=debug se registran ademas todas las consultas SQL. No conviene
// usarlo contra datos reales: la sentencia puede contener datos personales.
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"
)

// claveIDPeticion es el tipo de la clave con la que el identificador de peticion
// viaja en el contexto. Es un tipo propio para no colisionar con otras claves.
type claveIDPeticion struct{}

// CampoIDPeticion es el nombre del campo con el que el identificador aparece en
// los registros, y tambien el de la cabecera de respuesta.
const CampoIDPeticion = "request_id"

// Setup configura el logger por defecto de slog y lo devuelve. Debe llamarse al
// inicio de main(), antes de cualquier otro registro.
func Setup() *slog.Logger {
	nivel := nivelDesdeEntorno()

	opciones := &slog.HandlerOptions{
		Level: nivel,
		// El origen (archivo:linea) solo se agrega en depuracion: en produccion
		// ocupa espacio sin aportar, porque el mensaje ya identifica la operacion.
		AddSource: nivel == slog.LevelDebug,
	}

	var manejador slog.Handler
	if formatoJSON() {
		manejador = slog.NewJSONHandler(os.Stdout, opciones)
	} else {
		manejador = slog.NewTextHandler(os.Stdout, opciones)
	}

	registro := slog.New(manejador)
	slog.SetDefault(registro)
	return registro
}

func nivelDesdeEntorno() slog.Level {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL"))) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func formatoJSON() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LOG_FORMAT"))) {
	case "json":
		return true
	case "text", "texto":
		return false
	default:
		// En produccion (GIN_MODE=release) el formato JSON es el que entienden
		// los recolectores de logs de las plataformas de despliegue.
		return os.Getenv("GIN_MODE") == "release"
	}
}

// ConIDPeticion devuelve un contexto que lleva el identificador de la peticion,
// para que los registros de las capas siguientes puedan correlacionarse.
func ConIDPeticion(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, claveIDPeticion{}, id)
}

// IDPeticion extrae el identificador de peticion del contexto. Devuelve cadena
// vacia si no lo lleva.
func IDPeticion(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(claveIDPeticion{}).(string); ok {
		return id
	}
	return ""
}

// Desde devuelve un logger que ya incluye el identificador de la peticion, si el
// contexto lo lleva. Sirve para que un registro emitido en la capa de servicio
// pueda seguirse hasta la peticion HTTP que lo origino.
func Desde(ctx context.Context) *slog.Logger {
	if id := IDPeticion(ctx); id != "" {
		return slog.Default().With(CampoIDPeticion, id)
	}
	return slog.Default()
}

var (
	// Enmascara el valor asignado a la columna password, como en un UPDATE.
	patronPassword = regexp.MustCompile(`(?i)("?password"?\s*=\s*)'[^']*'`)
	// Enmascara cualquier hash de bcrypt, que tiene un formato inconfundible
	// ($2a$, $2b$ o $2y$ seguidos del costo y 53 caracteres). Cubre las sentencias
	// donde el valor no esta junto al nombre de la columna, como un INSERT, en las
	// que no se puede saber por regex a que columna corresponde cada valor.
	patronHash = regexp.MustCompile(`\$2[aby]\$\d{2}\$[./A-Za-z0-9]{53}`)
	// Enmascara tokens que pudieran aparecer en una cadena de conexion.
	patronToken = regexp.MustCompile(`(?i)(authToken=)[^&\s]+`)
)

// Sanear oculta los valores sensibles de un texto antes de registrarlo. Se aplica
// a las sentencias SQL y a las cadenas de conexion.
//
// Limitacion conocida: en una sentencia con valores posicionales —un INSERT— no se
// puede asociar por expresion regular cada valor con su columna, asi que solo se
// oculta lo que es reconocible por su forma (los hashes de bcrypt) o por su
// asignacion explicita. Por eso las consultas SQL solo se registran cuando hay un
// error, cuando son lentas, o con LOG_LEVEL=debug.
func Sanear(texto string) string {
	texto = patronPassword.ReplaceAllString(texto, `${1}'[oculto]'`)
	texto = patronHash.ReplaceAllString(texto, "[hash oculto]")
	texto = patronToken.ReplaceAllString(texto, `${1}[oculto]`)
	return texto
}

// escritorSlog reenvia a slog, linea por linea, lo que un componente escriba en
// un io.Writer. Se usa para capturar los mensajes que Gin imprime por su cuenta
// (volcado de rutas y advertencias de arranque) y que no salgan mezclados con el
// registro estructurado.
type escritorSlog struct {
	nivel slog.Level
}

func (e escritorSlog) Write(p []byte) (int, error) {
	for _, linea := range strings.Split(string(p), "\n") {
		linea = strings.TrimSpace(linea)
		if linea == "" {
			continue
		}
		slog.Default().Log(context.Background(), e.nivel, "gin: "+linea)
	}
	return len(p), nil
}

// EscritorDepuracion devuelve un io.Writer que registra lo recibido con nivel de
// depuracion. Lo que Gin escriba solo aparecera con LOG_LEVEL=debug.
func EscritorDepuracion() io.Writer {
	return escritorSlog{nivel: slog.LevelDebug}
}
