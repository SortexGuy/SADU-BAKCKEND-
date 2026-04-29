package services

import (
	"errors"
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

var jwtKey = []byte(os.Getenv("SECRET_KEY"))

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

	if len(jwtKey) == 0 {
		jwtKey = []byte(os.Getenv("SECRET_KEY"))
		if len(jwtKey) == 0 {
			jwtKey = []byte("tu_clave_secreta")
		}
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
	tokenString, err := token.SignedString(jwtKey)
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

	// Update existing admin password
	return s.DB.Model(&user).Update("password", string(hashedPassword)).Error
}
