package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5" // Изменено с jwt на jwt/v5
)

func AuthMiddleware(jwtKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
			c.Abort()
			return
		}

		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)

		// Парсим токен
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtKey), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Недействительный токен"})
			c.Abort()
			return
		}

		// Извлекаем claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Недействительный токен"})
			c.Abort()
			return
		}

		// Сохраняем информацию о пользователе в контексте
		c.Set("userID", claims["user_id"])
		c.Set("username", claims["username"])

		// Преобразуем role_id в int
		if roleID, ok := claims["role_id"].(float64); ok {
			c.Set("roleID", int(roleID))
		}

		c.Next()
	}
}

func RoleMiddleware(allowedRoles ...int) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("roleID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
			c.Abort()
			return
		}

		for _, role := range allowedRoles {
			if userRole == role {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Недостаточно прав"})
		c.Abort()
	}
}
