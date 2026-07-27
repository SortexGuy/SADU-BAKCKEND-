package logging_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"uneg.edu.ve/servicio-sadu-back/internal/logging"
	"uneg.edu.ve/servicio-sadu-back/internal/testutil"
)

func TestMain(m *testing.M) {
	codigo := m.Run()
	fmt.Print(testutil.ResumenTecnicas())
	os.Exit(codigo)
}

func TestSanear(t *testing.T) {
	testutil.Marcar(t, testutil.CajaBlanca, "ataca cada patron de ocultacion, incluido el hash en un INSERT posicional")

	// Lo que se registra puede acabar en los logs del despliegue, asi que las
	// contrasenas y los tokens no deben aparecer ni siquiera cifrados.
	casos := []struct {
		nombre    string
		entrada   string
		noDebeVer string
		debeVer   string
	}{
		{
			nombre:    "hash de contrasena en un UPDATE",
			entrada:   `UPDATE "users" SET "password"='$2a$10$abcdefghijklmnop',"updated_at"='2026-01-01' WHERE id = 1`,
			noDebeVer: "$2a$10$abcdefghijklmnop",
			debeVer:   "[oculto]",
		},
		{
			// La aplicacion siempre cifra antes de guardar, asi que lo que puede
			// aparecer en un INSERT es un hash. Se reconoce por su formato, no por
			// la columna, porque en un INSERT los valores son posicionales.
			nombre: "hash en un INSERT con valores posicionales",
			entrada: `INSERT INTO "users" ("username","password","email") ` +
				`VALUES ('admin','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy','')`,
			noDebeVer: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
			debeVer:   "[hash oculto]",
		},
		{
			nombre:    "token de la cadena de conexion",
			entrada:   `libsql://base.turso.io?authToken=eyJhbGciOiJFZERTQSJ9.token-larguisimo`,
			noDebeVer: "eyJhbGciOiJFZERTQSJ9.token-larguisimo",
			debeVer:   "[oculto]",
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			salida := logging.Sanear(caso.entrada)

			if strings.Contains(salida, caso.noDebeVer) {
				t.Errorf("el valor sensible sigue visible: %s", salida)
			}
			if !strings.Contains(salida, caso.debeVer) {
				t.Errorf("deberia aparecer la marca de ocultado: %s", salida)
			}
		})
	}

	t.Run("no altera un texto sin secretos", func(t *testing.T) {
		consulta := `SELECT * FROM athletes WHERE gov_id = '30000001'`
		if salida := logging.Sanear(consulta); salida != consulta {
			t.Errorf("una consulta sin secretos no deberia cambiar:\n  entrada: %s\n  salida:  %s", consulta, salida)
		}
	})
}

func TestIDPeticionEnElContexto(t *testing.T) {
	testutil.Marcar(t, testutil.CajaBlanca, "el mecanismo interno de propagacion por el contexto")

	t.Run("va y vuelve del contexto", func(t *testing.T) {
		ctx := logging.ConIDPeticion(context.Background(), "abc-123")
		if obtenido := logging.IDPeticion(ctx); obtenido != "abc-123" {
			t.Errorf("se obtuvo %q y se esperaba abc-123", obtenido)
		}
	})

	t.Run("un contexto sin identificador devuelve vacio", func(t *testing.T) {
		if obtenido := logging.IDPeticion(context.Background()); obtenido != "" {
			t.Errorf("se esperaba cadena vacia y se obtuvo %q", obtenido)
		}
	})

	t.Run("un contexto nulo no provoca panic", func(t *testing.T) {
		if obtenido := logging.IDPeticion(nil); obtenido != "" {
			t.Errorf("se esperaba cadena vacia y se obtuvo %q", obtenido)
		}
	})
}

func TestDesdeDevuelveUnLoggerUtilizable(t *testing.T) {
	testutil.Marcar(t, testutil.CajaBlanca, "que la funcion nunca devuelva nil, con y sin identificador")

	// Con y sin identificador, Desde() nunca debe devolver nil: cualquier capa lo
	// usa sin comprobar.
	if logging.Desde(context.Background()) == nil {
		t.Error("Desde() no deberia devolver nil sin identificador")
	}
	if logging.Desde(logging.ConIDPeticion(context.Background(), "abc")) == nil {
		t.Error("Desde() no deberia devolver nil con identificador")
	}
}

func TestSetupRespetaElNivel(t *testing.T) {
	testutil.Marcar(t, testutil.CajaBlanca, "las ramas que traducen LOG_LEVEL a un nivel de slog")

	t.Run("con LOG_LEVEL=debug habilita depuracion", func(t *testing.T) {
		t.Setenv("LOG_LEVEL", "debug")
		registro := logging.Setup()

		if !registro.Enabled(context.Background(), -4) { // slog.LevelDebug
			t.Error("con LOG_LEVEL=debug deberia registrarse el nivel de depuracion")
		}
	})

	t.Run("por defecto no registra depuracion", func(t *testing.T) {
		t.Setenv("LOG_LEVEL", "")
		registro := logging.Setup()

		if registro.Enabled(context.Background(), -4) {
			t.Error("sin LOG_LEVEL el nivel de depuracion deberia quedar apagado")
		}
		if !registro.Enabled(context.Background(), 0) { // slog.LevelInfo
			t.Error("el nivel de informacion deberia estar habilitado")
		}
	})

	t.Run("con LOG_LEVEL=error silencia las advertencias", func(t *testing.T) {
		t.Setenv("LOG_LEVEL", "error")
		registro := logging.Setup()

		if registro.Enabled(context.Background(), 4) { // slog.LevelWarn
			t.Error("con LOG_LEVEL=error las advertencias deberian quedar fuera")
		}
		if !registro.Enabled(context.Background(), 8) { // slog.LevelError
			t.Error("el nivel de error deberia estar habilitado")
		}
	})
}
