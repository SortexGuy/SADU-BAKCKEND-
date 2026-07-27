package services_test

import (
	"testing"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
	"uneg.edu.ve/servicio-sadu-back/config"
	"uneg.edu.ve/servicio-sadu-back/internal/services"
	"uneg.edu.ve/servicio-sadu-back/internal/testutil"
	"uneg.edu.ve/servicio-sadu-back/schema"
)

// crearUsuario inserta un usuario con la contrasena ya cifrada, que es como lo
// hace EnsureAdminUser.
func crearUsuario(t *testing.T, servicio services.UserService, usuario, contrasena string) schema.User {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(contrasena), bcrypt.DefaultCost)
	testutil.SinError(t, err, "cifrar la contrasena")

	registro := schema.User{Username: usuario, Password: string(hash)}
	testutil.SinError(t, servicio.DB.Create(&registro).Error, "crear el usuario")
	return registro
}

func TestIniciarSesion(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "credenciales correctas e incorrectas, y el token verificado como lo haria un cliente")

	t.Setenv("SECRET_KEY", "clave-de-prueba-para-firmar")

	t.Run("con credenciales correctas devuelve un token valido", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		servicio := services.UserService{DB: db}
		usuario := crearUsuario(t, servicio, "admin", "clave-buena")

		token, err := servicio.LoginUser("admin", "clave-buena")
		testutil.SinError(t, err, "iniciar sesion")
		if token == "" {
			t.Fatal("el token no deberia venir vacio")
		}

		// El token debe estar firmado con la clave del servidor y llevar los datos
		// del usuario, porque el middleware los usa para identificarlo.
		reclamos := jwt.MapClaims{}
		_, err = jwt.ParseWithClaims(token, reclamos, func(*jwt.Token) (any, error) {
			return config.SecretKey(), nil
		})
		testutil.SinError(t, err, "verificar la firma del token")
		testutil.Igual(t, reclamos["username"], "admin", "usuario en el token")
		testutil.Igual(t, uint(reclamos["user_id"].(float64)), usuario.ID, "id en el token")
		if reclamos["exp"] == nil {
			t.Error("el token deberia llevar fecha de expiracion")
		}
	})

	t.Run("con contrasena incorrecta falla", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		testutil.SilenciarLogs(t)
		servicio := services.UserService{DB: db}
		crearUsuario(t, servicio, "admin", "clave-buena")

		if _, err := servicio.LoginUser("admin", "clave-mala"); err == nil {
			t.Fatal("se esperaba error con la contrasena incorrecta")
		}
	})

	t.Run("con usuario inexistente falla con el mismo mensaje", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		testutil.SilenciarLogs(t)
		servicio := services.UserService{DB: db}
		crearUsuario(t, servicio, "admin", "clave-buena")

		_, errUsuario := servicio.LoginUser("fantasma", "clave-buena")
		_, errClave := servicio.LoginUser("admin", "clave-mala")

		if errUsuario == nil || errClave == nil {
			t.Fatal("ambos casos deberian fallar")
		}
		// El mensaje no debe revelar si el usuario existe.
		testutil.Igual(t, errUsuario.Error(), errClave.Error(), "mensaje de credenciales")
	})
}

func TestGarantizarAdministrador(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "comportamiento del arranque: crea, no sobrescribe, restablece si se pide")

	t.Run("crea el usuario si no existe", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		servicio := services.UserService{DB: db}
		t.Setenv("ADMIN_USER", "admin")
		t.Setenv("ADMIN_PASS", "clave-inicial")

		testutil.SinError(t, servicio.EnsureAdminUser(), "garantizar el administrador")

		var total int64
		db.Model(&schema.User{}).Where("username = ?", "admin").Count(&total)
		testutil.Igual(t, total, int64(1), "administradores creados")

		_, err := servicio.LoginUser("admin", "clave-inicial")
		testutil.SinError(t, err, "iniciar sesion con la contrasena inicial")
	})

	t.Run("no sobrescribe la contrasena de un administrador existente", func(t *testing.T) {
		// Es la garantia de que un cambio hecho desde change-password sobrevive al
		// reinicio del servidor.
		db := testutil.NuevaDB(t)
		servicio := services.UserService{DB: db}
		t.Setenv("SECRET_KEY", "clave-de-prueba-para-firmar")
		t.Setenv("ADMIN_USER", "admin")
		t.Setenv("ADMIN_PASS", "clave-del-entorno")

		testutil.SinError(t, servicio.EnsureAdminUser(), "primer arranque")

		var usuario schema.User
		testutil.SinError(t, db.Where("username = ?", "admin").First(&usuario).Error, "leer el administrador")
		testutil.SinError(t, servicio.ChangePassword(usuario.ID, "clave-del-entorno", "clave-elegida"), "cambiar la contrasena")

		// Segundo arranque con la misma variable de entorno.
		testutil.SinError(t, servicio.EnsureAdminUser(), "segundo arranque")

		if _, err := servicio.LoginUser("admin", "clave-elegida"); err != nil {
			t.Error("la contrasena elegida por el usuario deberia seguir siendo valida tras reiniciar")
		}
		testutil.SilenciarLogs(t)
		if _, err := servicio.LoginUser("admin", "clave-del-entorno"); err == nil {
			t.Error("la contrasena del entorno no deberia volver a funcionar sola")
		}
	})

	t.Run("con ADMIN_RESET_PASSWORD restablece la contrasena", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		servicio := services.UserService{DB: db}
		t.Setenv("SECRET_KEY", "clave-de-prueba-para-firmar")
		t.Setenv("ADMIN_USER", "admin")
		t.Setenv("ADMIN_PASS", "clave-del-entorno")

		testutil.SinError(t, servicio.EnsureAdminUser(), "primer arranque")

		var usuario schema.User
		testutil.SinError(t, db.Where("username = ?", "admin").First(&usuario).Error, "leer el administrador")
		testutil.SinError(t, servicio.ChangePassword(usuario.ID, "clave-del-entorno", "clave-olvidada"), "cambiar la contrasena")

		t.Setenv("ADMIN_RESET_PASSWORD", "true")
		testutil.SinError(t, servicio.EnsureAdminUser(), "arranque con restablecimiento")

		_, err := servicio.LoginUser("admin", "clave-del-entorno")
		testutil.SinError(t, err, "iniciar sesion con la contrasena restablecida")
	})

	t.Run("sin las variables de entorno no crea nada", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		testutil.SilenciarLogs(t)
		servicio := services.UserService{DB: db}
		t.Setenv("ADMIN_USER", "")
		t.Setenv("ADMIN_PASS", "")

		testutil.SinError(t, servicio.EnsureAdminUser(), "garantizar el administrador sin variables")

		var total int64
		db.Model(&schema.User{}).Count(&total)
		testutil.Igual(t, total, int64(0), "usuarios creados")
	})
}

func TestCambiarContrasena(t *testing.T) {
	testutil.Marcar(t, testutil.CajaNegra, "contrato del cambio de contrasena; una asercion mira el hash guardado")

	t.Setenv("SECRET_KEY", "clave-de-prueba-para-firmar")

	t.Run("con la contrasena actual correcta la reemplaza", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		servicio := services.UserService{DB: db}
		usuario := crearUsuario(t, servicio, "admin", "clave-vieja")

		testutil.SinError(t, servicio.ChangePassword(usuario.ID, "clave-vieja", "clave-nueva"), "cambiar la contrasena")

		_, err := servicio.LoginUser("admin", "clave-nueva")
		testutil.SinError(t, err, "iniciar sesion con la contrasena nueva")

		testutil.SilenciarLogs(t)
		if _, err := servicio.LoginUser("admin", "clave-vieja"); err == nil {
			t.Error("la contrasena vieja no deberia seguir funcionando")
		}
	})

	t.Run("con la contrasena actual incorrecta no cambia nada", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		testutil.SilenciarLogs(t)
		servicio := services.UserService{DB: db}
		usuario := crearUsuario(t, servicio, "admin", "clave-vieja")

		err := servicio.ChangePassword(usuario.ID, "clave-equivocada", "clave-nueva")
		if err == nil {
			t.Fatal("se esperaba error con la contrasena actual incorrecta")
		}
		testutil.Igual(t, err.Error(), "current password is incorrect", "mensaje de error")

		if _, err := servicio.LoginUser("admin", "clave-vieja"); err != nil {
			t.Error("la contrasena original deberia seguir siendo valida")
		}
	})

	t.Run("con un usuario inexistente falla", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		testutil.SilenciarLogs(t)
		servicio := services.UserService{DB: db}

		if err := servicio.ChangePassword(999, "x", "y"); err == nil {
			t.Fatal("se esperaba error con un usuario inexistente")
		}
	})

	t.Run("la contrasena se guarda cifrada", func(t *testing.T) {
		db := testutil.NuevaDB(t)
		servicio := services.UserService{DB: db}
		usuario := crearUsuario(t, servicio, "admin", "clave-vieja")

		testutil.SinError(t, servicio.ChangePassword(usuario.ID, "clave-vieja", "clave-nueva"), "cambiar la contrasena")

		var guardado schema.User
		testutil.SinError(t, db.First(&guardado, usuario.ID).Error, "leer el usuario")
		if guardado.Password == "clave-nueva" {
			t.Fatal("la contrasena quedo guardada en claro")
		}
		testutil.SinError(t,
			bcrypt.CompareHashAndPassword([]byte(guardado.Password), []byte("clave-nueva")),
			"el hash deberia corresponder a la contrasena nueva")
	})
}
