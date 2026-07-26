package middlewares

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"uneg.edu.ve/servicio-sadu-back/config"
	"uneg.edu.ve/servicio-sadu-back/helpers"
)

var ignoreRoutesOfVerification = []string{
	"/users/id/:id",
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			helpers.SendError(c, http.StatusUnauthorized,"Error", "Token no proporcionado")
			c.Abort()
			return
		}
		if strings.HasPrefix(strings.ToLower(tokenString), "bearer ") {
			tokenString = tokenString[7:]
		}
		tokenString = strings.TrimSpace(tokenString)

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return config.SecretKey(), nil
		})

		if err != nil || !token.Valid {
			helpers.SendError(c, http.StatusUnauthorized,"Error", err.Error())
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			//fmt.Printf("CLAIMS EXTRAÍDOS: ID=%v, User=%v\n", claims["user_id"], claims["username"])
			userID := uint(claims["user_id"].(float64))
			username := claims["username"].(string)
			c.Set("userId", userID)
			c.Set("username", username)

			// Ignore verification for some routes
			for _, route := range ignoreRoutesOfVerification {
				if c.FullPath() == route {
					paramId := c.Param("id")
					userIdStr := strconv.Itoa(int(userID))

					if paramId == userIdStr {
						c.Next()
						return
					}
					helpers.SendError(c, http.StatusForbidden, "Error", "No tienes permiso para acceder a este perfil")
					c.Abort()
					return
				}
			}
		} else {
			helpers.SendError(c, http.StatusInternalServerError, "Error", "invalid token claims")
			c.Abort()
			return
		}

		c.Next()
	}
}
