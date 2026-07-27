package services_test

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"uneg.edu.ve/servicio-sadu-back/internal/testutil"
)

// TestMain deja Gin en modo de prueba para que no imprima su propio registro
// mientras corren las pruebas.
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	codigo := m.Run()
	fmt.Print(testutil.ResumenTecnicas())
	os.Exit(codigo)
}

// idComoTexto convierte un identificador en la cadena que espera un parametro de
// ruta.
func idComoTexto(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}
