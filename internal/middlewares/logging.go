package middlewares

import (
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"uneg.edu.ve/servicio-sadu-back/helpers"
	"uneg.edu.ve/servicio-sadu-back/internal/logging"
)

// RequestID asigna un identificador unico a cada peticion, lo deja disponible
// para el resto de la cadena (contexto de Gin y contexto de la peticion) y lo
// devuelve en la cabecera X-Request-Id.
//
// Sirve para correlacionar: el registro de acceso, el del error y los de la capa
// de servicio de una misma peticion comparten ese identificador.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Si el cliente o un proxy ya envio uno, se respeta.
		id := c.GetHeader("X-Request-Id")
		if id == "" {
			id = uuid.NewString()
		}

		c.Set(logging.CampoIDPeticion, id)
		c.Request = c.Request.WithContext(logging.ConIDPeticion(c.Request.Context(), id))
		c.Header("X-Request-Id", id)

		c.Next()
	}
}

// AccessLog registra una linea por peticion atendida, con el metodo, la ruta, el
// codigo de respuesta, la duracion y el cliente. El nivel depende del resultado:
// 5xx se registra como error, 4xx como advertencia y el resto como informacion,
// de modo que filtrar por nivel ya separa los problemas del trafico normal.
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		inicio := time.Now()
		ruta := c.FullPath()
		if ruta == "" {
			ruta = "(sin ruta)" // peticion a una ruta no registrada
		}

		c.Next()

		estado := c.Writer.Status()
		atributos := []any{
			"metodo", c.Request.Method,
			"ruta", ruta,
			"url", c.Request.URL.Path,
			"estado", estado,
			"duracion_ms", time.Since(inicio).Milliseconds(),
			"ip", c.ClientIP(),
			"bytes", c.Writer.Size(),
		}

		// Si la peticion venia autenticada, el usuario ayuda a auditar.
		if usuario, existe := c.Get("username"); existe {
			atributos = append(atributos, "usuario", usuario)
		}
		if consulta := c.Request.URL.RawQuery; consulta != "" {
			atributos = append(atributos, "filtros", consulta)
		}

		registro := logging.Desde(c.Request.Context())
		switch {
		case estado >= http.StatusInternalServerError:
			registro.Error("peticion atendida", atributos...)
		case estado >= http.StatusBadRequest:
			registro.Warn("peticion atendida", atributos...)
		default:
			registro.Info("peticion atendida", atributos...)
		}
	}
}

// Recovery atrapa cualquier panic de un handler, lo registra con la pila y
// responde con el mismo formato de error que el resto de la API, en lugar de
// cerrar la conexion sin explicacion.
func Recovery() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recuperado any) {
		logging.Desde(c.Request.Context()).Error("panic en un handler",
			"panic", recuperado,
			"metodo", c.Request.Method,
			"url", c.Request.URL.Path,
			"pila", string(debug.Stack()),
		)

		helpers.SendError(c, http.StatusInternalServerError, "Error interno del servidor",
			"Ocurrió un error inesperado al procesar la solicitud.")
		c.Abort()
	})
}
