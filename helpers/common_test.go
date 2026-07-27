package helpers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"uneg.edu.ve/servicio-sadu-back/helpers"
	"uneg.edu.ve/servicio-sadu-back/internal/testutil"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	codigo := m.Run()
	fmt.Print(testutil.ResumenTecnicas())
	os.Exit(codigo)
}

// contextoDePrueba devuelve un contexto de Gin listo para escribir una respuesta.
func contextoDePrueba(ruta string) (*gin.Context, *httptest.ResponseRecorder) {
	grabadora := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(grabadora)
	ctx.Request = httptest.NewRequest("GET", ruta, nil)
	return ctx, grabadora
}

func TestSendSucces(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "el contrato de exito que validan los esquemas del frontend")

	// El contrato de exito es lo que validan los esquemas del frontend: si cambia
	// la envoltura, el cliente deja de poder leer la respuesta.
	ctx, grabadora := contextoDePrueba("/atletas")
	helpers.SendSucces(ctx, "listing-athletes", []string{"a", "b"})

	if grabadora.Code != http.StatusOK {
		t.Errorf("se esperaba HTTP 200 y se obtuvo %d", grabadora.Code)
	}

	var envoltura struct {
		Message string   `json:"message"`
		Data    []string `json:"data"`
	}
	if err := json.Unmarshal(grabadora.Body.Bytes(), &envoltura); err != nil {
		t.Fatalf("la respuesta deberia ser JSON con data y message: %v", err)
	}
	if len(envoltura.Data) != 2 {
		t.Errorf("data deberia traer los dos elementos, y trajo %d", len(envoltura.Data))
	}
	if envoltura.Message == "" {
		t.Error("message no deberia venir vacio")
	}
}

func TestSendSuccesConDataNula(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "la envoltura sigue siendo valida sin datos")

	// El cambio de contrasena responde sin datos: la envoltura debe seguir siendo
	// valida.
	ctx, grabadora := contextoDePrueba("/users/change-password")
	helpers.SendSucces(ctx, "password changed", nil)

	var envoltura map[string]any
	if err := json.Unmarshal(grabadora.Body.Bytes(), &envoltura); err != nil {
		t.Fatalf("la respuesta deberia ser JSON valido: %v", err)
	}
	if _, existe := envoltura["data"]; !existe {
		t.Error("el campo data deberia estar presente aunque sea nulo")
	}
}

func TestSendError(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "el formato de problema y la correspondencia con el codigo HTTP")

	casos := []struct {
		nombre  string
		codigo  int
		titulo  string
		detalle string
		ruta    string
	}{
		{"peticion invalida", http.StatusBadRequest, "Cédula obligatoria", "El atleta debe tener una cédula.", "/athletes/create"},
		{"conflicto", http.StatusConflict, "Cédula duplicada", "Ya existe un atleta con esa cédula.", "/athletes/create"},
		{"no encontrado", http.StatusNotFound, "No encontrado", "El ID no existe.", "/athletes/99"},
		{"error del servidor", http.StatusInternalServerError, "Error interno", "Algo inesperado.", "/athletes"},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			ctx, grabadora := contextoDePrueba(caso.ruta)
			helpers.SendError(ctx, caso.codigo, caso.titulo, caso.detalle)

			if grabadora.Code != caso.codigo {
				t.Errorf("se esperaba HTTP %d y se obtuvo %d", caso.codigo, grabadora.Code)
			}

			var problema helpers.ProblemDetails
			if err := json.Unmarshal(grabadora.Body.Bytes(), &problema); err != nil {
				t.Fatalf("el error deberia ser JSON con el formato de problema: %v", err)
			}

			if problema.Status != caso.codigo {
				t.Errorf("el status del cuerpo (%d) deberia coincidir con el HTTP (%d)", problema.Status, caso.codigo)
			}
			if problema.Tittle != caso.titulo {
				t.Errorf("titulo: se obtuvo %q y se esperaba %q", problema.Tittle, caso.titulo)
			}
			if problema.Detail != caso.detalle {
				t.Errorf("detalle: se obtuvo %q y se esperaba %q", problema.Detail, caso.detalle)
			}
			if problema.Instance != caso.ruta {
				t.Errorf("instancia: se obtuvo %q y se esperaba la ruta %q", problema.Instance, caso.ruta)
			}
			if problema.Type == "" {
				t.Error("el campo type no deberia venir vacio")
			}
		})
	}
}
