package services

import (
	"errors"
	"log/slog"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"uneg.edu.ve/servicio-sadu-back/config"
	"uneg.edu.ve/servicio-sadu-back/schema"
)

type UserService struct {
	DB *gorm.DB
}

func NewUserService() *UserService {
	return &UserService{DB: config.DB}
}

func (s *UserService) LoginUser(username, password string) (string, error) {
	var user schema.User

	// Busca al usuario por el username
	if err := s.DB.Where("username = ?", username).First(&user).Error; err != nil {
		// Usuario inexistente: se registra el intento sin revelar si el nombre
		// existe en la respuesta al cliente.
		slog.Warn("intento de inicio de sesion fallido", "usuario", username, "motivo", "usuario no encontrado")
		return "", errors.New("invalid credentials")
	}

	// Verifica el password usando bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		slog.Warn("intento de inicio de sesion fallido", "usuario", username, "motivo", "contrasena incorrecta")
		return "", errors.New("invalid credentials")
	}

	expirationTime := time.Now().Add(24 * time.Hour) // 24h
	var claims = schema.Claims{
		UserId:   user.ID,
		Username: user.Username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(config.SecretKey())
	if err != nil {
		slog.Error("no se pudo firmar el token", "usuario", username, "error", err.Error())
		return "", err
	}

	slog.Info("inicio de sesion correcto",
		"usuario", user.Username,
		"usuario_id", user.ID,
		"expira", expirationTime.Format(time.RFC3339),
	)
	return tokenString, nil
}

func (s *UserService) EnsureAdminUser() error {
	username := os.Getenv("ADMIN_USER")
	password := os.Getenv("ADMIN_PASS")

	if username == "" || password == "" {
		slog.Warn("ADMIN_USER o ADMIN_PASS sin definir: no se creara ningun usuario, " +
			"asi que no sera posible iniciar sesion")
		return nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	var user schema.User
	err = s.DB.Where("username = ?", username).First(&user).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// No hay nadie con ese correo, y eso tiene dos lecturas distintas.
		var cuentas int64
		if err := s.DB.Model(&schema.User{}).Count(&cuentas).Error; err != nil {
			return err
		}

		// Tabla vacia: primer arranque, se crea el administrador.
		if cuentas == 0 {
			user = schema.User{
				Username: username,
				Password: string(hashedPassword),
			}
			if err := s.DB.Create(&user).Error; err != nil {
				return err
			}
			slog.Info("usuario administrador creado", "usuario", username)
			return nil
		}

		// Ya hay una cuenta con otro correo: es la misma de siempre, cambiado desde
		// el perfil. Crear una segunda con ADMIN_USER dejaria abierta una credencial
		// paralela con la contrasena del entorno, que es justo lo que el cambio de
		// correo pretendia cerrar.
		if os.Getenv("ADMIN_RESET_PASSWORD") != "true" {
			slog.Info("el administrador usa un correo distinto de ADMIN_USER: no se crea otra cuenta",
				"admin_user", username, "cuentas", cuentas)
			return nil
		}

		// Recuperacion: ADMIN_RESET_PASSWORD devuelve la cuenta mas antigua a las
		// credenciales del entorno, correo incluido. Es la unica forma de volver a
		// entrar si el correo nuevo se perdio.
		if err := s.DB.Order("id").First(&user).Error; err != nil {
			return err
		}
		slog.Warn("restableciendo correo y contrasena del administrador por ADMIN_RESET_PASSWORD=true",
			"usuario_id", user.ID, "usuario", username)
		return s.DB.Model(&user).Updates(map[string]interface{}{
			"username": username,
			"password": string(hashedPassword),
		}).Error
	}

	// El administrador ya existe. No se sobrescribe su contraseña en cada arranque:
	// hacerlo deshacia en silencio cualquier cambio hecho desde /users/change-password,
	// que quedaba revertido al siguiente reinicio del servidor.
	//
	// Para recuperar el acceso si se pierde la contraseña, arrancar una vez con
	// ADMIN_RESET_PASSWORD=true: eso restablece la contraseña al valor de ADMIN_PASS.
	if os.Getenv("ADMIN_RESET_PASSWORD") != "true" {
		slog.Info("usuario administrador ya existe: no se toca su contrasena", "usuario", username)
		return nil
	}

	slog.Warn("restableciendo la contrasena del administrador por ADMIN_RESET_PASSWORD=true", "usuario", username)
	return s.DB.Model(&user).Update("password", string(hashedPassword)).Error
}

// esCorreo comprueba que la cadena sea una direccion de correo a secas.
//
// mail.ParseAddress por si solo acepta cosas que no sirven como credencial:
// "Nombre <correo@dominio>", y dominios sin punto como "admin@uneg", que son
// validos en la norma pero casi siempre son una direccion escrita a medias. Se
// exige la direccion pelada y un dominio con punto, que es lo mismo que valida el
// formulario del cliente.
func esCorreo(valor string) bool {
	direccion, err := mail.ParseAddress(valor)
	if err != nil || direccion.Address != valor {
		return false
	}

	dominio := valor[strings.LastIndex(valor, "@")+1:]
	return strings.Contains(dominio, ".") && !strings.HasSuffix(dominio, ".")
}

// GetProfile devuelve los datos de la cuenta que abrio la sesion. El id sale del
// token, no de la peticion, asi que nadie puede pedir el perfil de otro.
func (s *UserService) GetProfile(userID uint) (schema.UserProfileDTO, error) {
	var user schema.User
	if err := s.DB.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return schema.UserProfileDTO{}, ErrUserNotFound
		}
		return schema.UserProfileDTO{}, err
	}

	return schema.UserProfileDTO{ID: user.ID, Username: user.Username}, nil
}

// ChangeUsername cambia el correo con el que se inicia sesion. Guarda el valor en
// minusculas y sin espacios porque LoginUser compara la columna tal cual: un
// correo escrito con mayusculas al guardarlo no volveria a encontrarse al entrar.
//
// El token que ya tiene el cliente sigue valido: identifica al usuario por id, y
// el correo que lleva dentro solo se usa para registrar quien hizo cada cosa. No
// hace falta volver a iniciar sesion.
func (s *UserService) ChangeUsername(userID uint, currentPassword, newUsername string) error {
	nuevo := strings.ToLower(strings.TrimSpace(newUsername))
	if nuevo == "" {
		return ErrMissingUsername
	}
	if !esCorreo(nuevo) {
		return ErrInvalidEmail
	}

	var user schema.User
	if err := s.DB.First(&user, userID).Error; err != nil {
		slog.Warn("cambio de correo rechazado", "usuario_id", userID, "motivo", "usuario no encontrado")
		return ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
		slog.Warn("cambio de correo rechazado", "usuario_id", userID, "motivo", "contrasena actual incorrecta")
		return ErrInvalidCurrentPassword
	}

	// Pedir el correo que ya se tiene no es un error: no hay nada que escribir.
	if nuevo == user.Username {
		return nil
	}

	var enUso int64
	if err := s.DB.Model(&schema.User{}).
		Where("LOWER(username) = ? AND id <> ?", nuevo, userID).
		Count(&enUso).Error; err != nil {
		return err
	}
	if enUso > 0 {
		slog.Warn("cambio de correo rechazado", "usuario_id", userID, "motivo", "correo en uso")
		return ErrUsernameTaken
	}

	if err := s.DB.Model(&user).Update("username", nuevo).Error; err != nil {
		slog.Error("no se pudo guardar el correo nuevo", "usuario_id", userID, "error", err.Error())
		return err
	}

	slog.Info("correo de acceso cambiado", "usuario_id", userID, "usuario", nuevo)
	return nil
}

func (s *UserService) ChangePassword(userID uint, oldPassword, newPassword string) error {
	var user schema.User
	if err := s.DB.First(&user, userID).Error; err != nil {
		slog.Warn("cambio de contrasena rechazado", "usuario_id", userID, "motivo", "usuario no encontrado")
		return errors.New("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		slog.Warn("cambio de contrasena rechazado", "usuario_id", userID, "motivo", "contrasena actual incorrecta")
		return errors.New("current password is incorrect")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if err := s.DB.Model(&user).Update("password", string(hashedPassword)).Error; err != nil {
		slog.Error("no se pudo guardar la contrasena nueva", "usuario_id", userID, "error", err.Error())
		return err
	}

	slog.Info("contrasena cambiada", "usuario", user.Username, "usuario_id", userID)
	return nil
}
