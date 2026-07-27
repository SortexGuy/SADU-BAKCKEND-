package middlewares_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"uneg.edu.ve/servicio-sadu-back/config"
	"uneg.edu.ve/servicio-sadu-back/internal/middlewares"
	"uneg.edu.ve/servicio-sadu-back/internal/testutil"
	"uneg.edu.ve/servicio-sadu-back/schema"
)

const claveDePrueba = "clave-de-prueba-del-middleware"

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Setenv("SECRET_KEY", claveDePrueba)
	codigo := m.Run()
	fmt.Print(testutil.ResumenTecnicas())
	os.Exit(codigo)
}

// tokenPara firma un token con los datos indicados, usando la clave del servidor.
func tokenPara(t *testing.T, usuarioID uint, usuario string, expiracion time.Time) string {
	t.Helper()

	reclamos := schema.Claims{
		UserId:         usuarioID,
		Username:       usuario,
		StandardClaims: jwt.StandardClaims{ExpiresAt: expiracion.Unix()},
	}
	firmado, err := jwt.NewWithClaims(jwt.SigningMethodHS256, reclamos).SignedString(config.SecretKey())
	if err != nil {
		t.Fatalf("no se pudo firmar el token de prueba: %v", err)
	}
	return firmado
}

// routerProtegido devuelve un enrutador con una ruta detras del middleware, que
// responde con los datos que el middleware dejo en el contexto.
func routerProtegido() *gin.Engine {
	r := gin.New()
	r.Use(middlewares.RequestID())
	r.GET("/protegido", middlewares.AuthMiddleware(), func(c *gin.Context) {
		usuario, _ := c.Get("username")
		id, _ := c.Get("userId")
		c.JSON(http.StatusOK, gin.H{"username": usuario, "userId": id})
	})
	return r
}

func peticion(r *gin.Engine, autorizacion string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", "/protegido", nil)
	if autorizacion != "" {
		req.Header.Set("Authorization", autorizacion)
	}
	respuesta := httptest.NewRecorder()
	r.ServeHTTP(respuesta, req)
	return respuesta
}

func TestAuthMiddleware(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "presencia y validez del token segun el contrato de autenticacion")

	r := routerProtegido()
	valido := tokenPara(t, 7, "admin", time.Now().Add(time.Hour))

	casos := []struct {
		nombre       string
		autorizacion string
		esperado     int
	}{
		{"sin cabecera", "", http.StatusUnauthorized},
		{"cabecera vacia", "", http.StatusUnauthorized},
		{"token sin prefijo Bearer", valido, http.StatusOK},
		{"token con prefijo Bearer", "Bearer " + valido, http.StatusOK},
		{"prefijo en minusculas", "bearer " + valido, http.StatusOK},
		{"prefijo en mayusculas", "BEARER " + valido, http.StatusOK},
		{"con espacios alrededor", "Bearer   " + valido + "  ", http.StatusOK},
		{"token con basura", "Bearer no-es-un-token", http.StatusUnauthorized},
		{"token vacio tras el prefijo", "Bearer ", http.StatusUnauthorized},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			respuesta := peticion(r, caso.autorizacion)
			if respuesta.Code != caso.esperado {
				t.Errorf("se obtuvo HTTP %d y se esperaba %d", respuesta.Code, caso.esperado)
			}
		})
	}
}

func TestTokenExpirado(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "un token vencido se rechaza")

	r := routerProtegido()
	expirado := tokenPara(t, 7, "admin", time.Now().Add(-time.Hour))

	respuesta := peticion(r, "Bearer "+expirado)
	if respuesta.Code != http.StatusUnauthorized {
		t.Errorf("un token expirado deberia rechazarse: HTTP %d", respuesta.Code)
	}
}

func TestTokenFirmadoConOtraClave(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "un token de otro emisor se rechaza")

	r := routerProtegido()

	reclamos := schema.Claims{
		UserId:         7,
		Username:       "intruso",
		StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(time.Hour).Unix()},
	}
	ajeno, err := jwt.NewWithClaims(jwt.SigningMethodHS256, reclamos).SignedString([]byte("otra-clave"))
	if err != nil {
		t.Fatalf("no se pudo firmar el token ajeno: %v", err)
	}

	respuesta := peticion(r, "Bearer "+ajeno)
	if respuesta.Code != http.StatusUnauthorized {
		t.Errorf("un token firmado con otra clave deberia rechazarse: HTTP %d", respuesta.Code)
	}
}

func TestElMiddlewarePublicaLosDatosDelUsuario(t *testing.T) {
	testutil.Marcar(t, testutil.CajaBlanca, "verifica las claves que el middleware deja en el contexto para los handlers")

	// Los handlers y el registro de acceso leen userId y username del contexto.
	r := routerProtegido()
	respuesta := peticion(r, "Bearer "+tokenPara(t, 42, "coordinador", time.Now().Add(time.Hour)))

	if respuesta.Code != http.StatusOK {
		t.Fatalf("la peticion deberia pasar: HTTP %d", respuesta.Code)
	}
	cuerpo := respuesta.Body.String()
	if !strings.Contains(cuerpo, `"username":"coordinador"`) {
		t.Errorf("el contexto deberia llevar el usuario: %s", cuerpo)
	}
	if !strings.Contains(cuerpo, `"userId":42`) {
		t.Errorf("el contexto deberia llevar el identificador: %s", cuerpo)
	}
}

func TestRequestID(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "la cabecera de respuesta y la unicidad entre peticiones")

	r := gin.New()
	r.Use(middlewares.RequestID())
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	t.Run("genera uno si el cliente no lo envia", func(t *testing.T) {
		respuesta := httptest.NewRecorder()
		r.ServeHTTP(respuesta, httptest.NewRequest("GET", "/x", nil))

		if respuesta.Header().Get("X-Request-Id") == "" {
			t.Error("deberia generarse un identificador de peticion")
		}
	})

	t.Run("respeta el que envia el cliente", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/x", nil)
		req.Header.Set("X-Request-Id", "id-del-cliente")

		respuesta := httptest.NewRecorder()
		r.ServeHTTP(respuesta, req)

		if obtenido := respuesta.Header().Get("X-Request-Id"); obtenido != "id-del-cliente" {
			t.Errorf("se obtuvo %q y se esperaba el identificador del cliente", obtenido)
		}
	})

	t.Run("da uno distinto a cada peticion", func(t *testing.T) {
		primera := httptest.NewRecorder()
		r.ServeHTTP(primera, httptest.NewRequest("GET", "/x", nil))
		segunda := httptest.NewRecorder()
		r.ServeHTTP(segunda, httptest.NewRequest("GET", "/x", nil))

		if primera.Header().Get("X-Request-Id") == segunda.Header().Get("X-Request-Id") {
			t.Error("dos peticiones distintas no deberian compartir identificador")
		}
	})
}

func TestRecovery(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "un panic responde 500 con el formato de error de la API")

	r := gin.New()
	r.Use(middlewares.RequestID(), middlewares.Recovery())
	r.GET("/estalla", func(c *gin.Context) { panic("fallo a proposito") })

	respuesta := httptest.NewRecorder()
	r.ServeHTTP(respuesta, httptest.NewRequest("GET", "/estalla", nil))

	if respuesta.Code != http.StatusInternalServerError {
		t.Errorf("un panic deberia responder 500 y se obtuvo %d", respuesta.Code)
	}
	// Y con el formato de error de la API, no con la conexion cortada.
	if !strings.Contains(respuesta.Body.String(), `"status":500`) {
		t.Errorf("la respuesta deberia usar el formato de error de la API: %s", respuesta.Body.String())
	}
}
