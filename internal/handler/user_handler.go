package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"sez-checkpoint-backend/internal/models"
	"sez-checkpoint-backend/internal/repository"
)

type UserHandler struct {
	userRepo         *repository.UserRepository
	organizationRepo *repository.OrganizationRepository
}

func NewUserHandler(
	userRepo *repository.UserRepository,
	organizationRepo *repository.OrganizationRepository,
) *UserHandler {
	return &UserHandler{
		userRepo:         userRepo,
		organizationRepo: organizationRepo,
	}
}

// GetProfile - получает профиль текущего пользователя
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
		return
	}

	user, err := h.userRepo.GetByID(userID.(string))
	if err != nil {
		log.Printf("❌ Ошибка при получении профиля пользователя %s: %v", userID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// GetAvailableLists - получает списки, доступные пользователю
func (h *UserHandler) GetAvailableLists(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
		return
	}

	log.Printf("🔍 Получение доступных списков для пользователя %s", userID)

	lists, err := h.userRepo.GetUserListPermissions(userID.(string))
	if err != nil {
		log.Printf("❌ Ошибка при получении списков пользователя %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении списков"})
		return
	}

	// Если списков нет, возвращаем пустой массив, а не null
	if lists == nil {
		lists = []*models.AccessList{}
	}

	log.Printf("✅ Найдено %d доступных списков для пользователя %s", len(lists), userID)
	c.JSON(http.StatusOK, lists)
}

// GetUserOrganization - получает организацию пользователя
func (h *UserHandler) GetUserOrganization(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
		return
	}

	user, err := h.userRepo.GetByID(userID.(string))
	if err != nil {
		log.Printf("❌ Ошибка при получении пользователя %s: %v", userID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	if user.OrganizationID == nil {
		c.JSON(http.StatusOK, gin.H{"organization": nil})
		return
	}

	org, err := h.organizationRepo.GetByID(*user.OrganizationID)
	if err != nil {
		log.Printf("❌ Ошибка при получении организации %s: %v", *user.OrganizationID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении организации"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"organization": org})
}

// UpdateProfile - обновляет профиль текущего пользователя
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
		return
	}

	var req struct {
		FullName string  `json:"fullName"`
		Email    *string `json:"email"`
		Phone    *string `json:"phone"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	user, err := h.userRepo.GetByID(userID.(string))
	if err != nil {
		log.Printf("❌ Ошибка при получении пользователя %s: %v", userID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.Email != nil {
		user.Email = req.Email
	}
	if req.Phone != nil {
		user.Phone = req.Phone
	}

	if err := h.userRepo.Update(user); err != nil {
		log.Printf("❌ Ошибка при обновлении пользователя %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении профиля"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Профиль успешно обновлен"})
}

// GetMyListPermissions - получает списки, доступные текущему пользователю
func (h *UserHandler) GetMyListPermissions(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
		return
	}

	log.Printf("🔍 Получение доступных списков для пользователя %s", userID)

	lists, err := h.userRepo.GetUserListPermissions(userID.(string))
	if err != nil {
		log.Printf("❌ Ошибка при получении списков пользователя: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении списков"})
		return
	}

	// Если списков нет, возвращаем пустой массив
	if lists == nil {
		lists = []*models.AccessList{}
	}

	log.Printf("✅ Найдено %d доступных списков для пользователя %s", len(lists), userID)
	c.JSON(http.StatusOK, lists)
}
