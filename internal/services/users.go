package services

import (
	"errors"
	"log"
	"os"
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
		return "", errors.New("invalid credentials")
	}

	// Verifica el password usando bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
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
		return "", err
	}

	return tokenString, nil
}

func (s *UserService) EnsureAdminUser() error {
	username := os.Getenv("ADMIN_USER")
	password := os.Getenv("ADMIN_PASS")

	if username == "" || password == "" {
		return nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	var user schema.User
	err = s.DB.Where("username = ?", username).First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new admin
			user = schema.User{
				Username: username,
				Password: string(hashedPassword),
			}
			return s.DB.Create(&user).Error
		}
		return err
	}

	// El administrador ya existe. No se sobrescribe su contraseña en cada arranque:
	// hacerlo deshacia en silencio cualquier cambio hecho desde /users/change-password,
	// que quedaba revertido al siguiente reinicio del servidor.
	//
	// Para recuperar el acceso si se pierde la contraseña, arrancar una vez con
	// ADMIN_RESET_PASSWORD=true: eso restablece la contraseña al valor de ADMIN_PASS.
	if os.Getenv("ADMIN_RESET_PASSWORD") != "true" {
		return nil
	}

	log.Println("ADMIN_RESET_PASSWORD=true: restableciendo la contraseña del administrador a ADMIN_PASS")
	return s.DB.Model(&user).Update("password", string(hashedPassword)).Error
}

func (s *UserService) ChangePassword(userID uint, oldPassword, newPassword string) error {
	var user schema.User
	if err := s.DB.First(&user, userID).Error; err != nil {
		return errors.New("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return errors.New("current password is incorrect")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if err := s.DB.Model(&user).Update("password", string(hashedPassword)).Error; err != nil {
		return err
	}

	return nil
}
