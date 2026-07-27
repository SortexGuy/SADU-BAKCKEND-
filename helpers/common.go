package helpers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"uneg.edu.ve/servicio-sadu-back/internal/logging"
)

type ProblemDetails struct {
	Type     string `json:"type"`
	Tittle   string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance"`
}

// SendError responde con el formato de error de la API y deja constancia en el
// registro. Como todos los errores del servidor pasan por aqui, este es el unico
// punto donde hay que mirar para saber que respuestas de error se estan emitiendo.
//
// El nivel distingue la causa: 5xx es un fallo del servidor y se registra como
// error; 4xx es una peticion invalida del cliente y se registra como advertencia.
func SendError(ctx *gin.Context, code int, title, detail string) {
	problem := ProblemDetails{
		Type:     "about:blank",
		Tittle:   title,
		Status:   code,
		Detail:   detail,
		Instance: ctx.Request.URL.Path,
	}

	registro := logging.Desde(ctx.Request.Context())
	atributos := []any{
		"estado", code,
		"titulo", title,
		"detalle", detail,
		"metodo", ctx.Request.Method,
		"url", ctx.Request.URL.Path,
	}
	if code >= http.StatusInternalServerError {
		registro.Error("respuesta de error", atributos...)
	} else {
		registro.Warn("respuesta de error", atributos...)
	}

	ctx.JSON(code, problem)
}

// SendSucces responde con el formato de exito de la API. El detalle de la
// operacion se registra a nivel de depuracion: el registro de acceso ya deja una
// linea por peticion, asi que repetirlo en informacion seria redundante.
func SendSucces(ctx *gin.Context, operation string, data any) {
	logging.Desde(ctx.Request.Context()).Debug("respuesta correcta",
		"operacion", operation,
		"metodo", ctx.Request.Method,
		"url", ctx.Request.URL.Path,
	)

	ctx.Header("Content-type", "application/json")
	ctx.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("operation from handler %s successfully", operation),
		"data":    data,
	})
}
