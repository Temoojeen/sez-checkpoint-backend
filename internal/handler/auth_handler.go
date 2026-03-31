package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"sez-checkpoint-backend/internal/models"
	"sez-checkpoint-backend/internal/repository"
)

type AuthHandler struct {
	userRepo *repository.UserRepository
	jwtKey   string
}

func NewAuthHandler(userRepo *repository.UserRepository, jwtKey string) *AuthHandler {
	return &AuthHandler{
		userRepo: userRepo,
		jwtKey:   jwtKey,
	}
}

// Login - авторизация пользователя
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ Ошибка парсинга запроса: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неверный формат запроса",
		})
		return
	}

	log.Printf("🔐 Попытка входа: username=%s", req.Username)

	// Получаем пользователя из БД
	user, err := h.userRepo.GetByUsername(req.Username)
	if err != nil {
		log.Printf("❌ Пользователь не найден: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Неверный логин или пароль",
		})
		return
	}

	// Проверяем пароль
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		log.Printf("❌ Неверный пароль для пользователя %s", req.Username)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Неверный логин или пароль",
		})
		return
	}

	log.Printf("✅ Пароль верный, генерируем токен")

	// Обновляем время последнего входа
	err = h.userRepo.UpdateLastLogin(user.ID)
	if err != nil {
		log.Printf("⚠️ Ошибка при обновлении времени входа: %v", err)
		// Не возвращаем ошибку, так как это не критично
	}

	// Генерируем JWT токен
	token, err := h.generateToken(user)
	if err != nil {
		log.Printf("❌ Ошибка генерации токена: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка при генерации токена",
		})
		return
	}

	log.Printf("✅ Токен сгенерирован, возвращаем ответ")

	// Формируем ответ
	responseUser := models.User{
		ID:       user.ID,
		Username: user.Username,
		FullName: user.FullName,
		RoleID:   user.RoleID,
		IsActive: user.IsActive,
	}

	// Добавляем опциональные поля если они есть
	if user.Email != nil {
		responseUser.Email = user.Email
	}
	if user.Phone != nil {
		responseUser.Phone = user.Phone
	}
	if user.OrganizationID != nil {
		responseUser.OrganizationID = user.OrganizationID
	}

	// Возвращаем ответ
	c.JSON(http.StatusOK, models.LoginResponse{
		Token: token,
		User:  responseUser,
	})
}

// generateToken - создает JWT токен
func (h *AuthHandler) generateToken(user *models.User) (string, error) {
	// Устанавливаем время жизни токена - 24 часа
	expirationTime := time.Now().Add(24 * time.Hour)

	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role_id":  user.RoleID,
		"exp":      expirationTime.Unix(),
		"iat":      time.Now().Unix(),
		"iss":      "sez-checkpoint-backend",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtKey))
}

// RefreshToken - обновление токена (опционально)
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	// Получаем токен из заголовка
	tokenString := c.GetHeader("Authorization")
	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Токен не предоставлен",
		})
		return
	}

	// Убираем "Bearer " из начала
	if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]
	}

	// Парсим токен
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.jwtKey), nil
	})

	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Недействительный токен",
		})
		return
	}

	// Извлекаем claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Недействительный токен",
		})
		return
	}

	// Получаем user_id из claims
	userID, ok := claims["user_id"].(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Недействительный токен",
		})
		return
	}

	// Получаем пользователя из БД
	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Пользователь не найден",
		})
		return
	}

	// Генерируем новый токен
	newToken, err := h.generateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка при генерации токена",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": newToken,
	})
}

// Logout - выход из системы (опционально - для инвалидации токена на клиенте)
func (h *AuthHandler) Logout(c *gin.Context) {
	// В реальном приложении здесь можно добавить токен в черный список
	// Но для простоты просто возвращаем успех
	c.JSON(http.StatusOK, gin.H{
		"message": "Выход выполнен успешно",
	})
}

// GetMe - получение информации о текущем пользователе
func (h *AuthHandler) GetMe(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Не авторизован",
		})
		return
	}

	user, err := h.userRepo.GetByID(userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Пользователь не найден",
		})
		return
	}

	// Формируем ответ
	responseUser := models.User{
		ID:       user.ID,
		Username: user.Username,
		FullName: user.FullName,
		RoleID:   user.RoleID,
		IsActive: user.IsActive,
	}

	if user.Email != nil {
		responseUser.Email = user.Email
	}
	if user.Phone != nil {
		responseUser.Phone = user.Phone
	}
	if user.OrganizationID != nil {
		responseUser.OrganizationID = user.OrganizationID
	}
	if user.OrganizationName != "" {
		responseUser.OrganizationName = user.OrganizationName
	}
	if user.RoleName != "" {
		responseUser.RoleName = user.RoleName
	}

	c.JSON(http.StatusOK, responseUser)
}

// ChangePassword - смена пароля
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неверный формат запроса",
		})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Не авторизован",
		})
		return
	}

	// Получаем пользователя
	user, err := h.userRepo.GetByID(userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Пользователь не найден",
		})
		return
	}

	// Проверяем старый пароль
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Неверный старый пароль",
		})
		return
	}

	// Хешируем новый пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка при хешировании пароля",
		})
		return
	}

	// Обновляем пароль
	err = h.userRepo.UpdatePassword(userID.(string), string(hashedPassword))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка при обновлении пароля",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Пароль успешно изменен",
	})
}
