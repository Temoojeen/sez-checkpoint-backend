package handler

import (
	"log"
	"net/http"

	"sez-checkpoint-backend/internal/repository"

	"github.com/gin-gonic/gin"
)

type ApprovedPlateHandler struct {
	approvedPlateRepo *repository.ApprovedPlateRepository
	userRepo          *repository.UserRepository
}

func NewApprovedPlateHandler(
	approvedPlateRepo *repository.ApprovedPlateRepository,
	userRepo *repository.UserRepository,
) *ApprovedPlateHandler {
	return &ApprovedPlateHandler{
		approvedPlateRepo: approvedPlateRepo,
		userRepo:          userRepo,
	}
}

// GetPlatesByList - получение номеров по списку
func (h *ApprovedPlateHandler) GetPlatesByList(c *gin.Context) {
	listID := c.Param("listId")
	if listID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID списка не указан"})
		return
	}

	// Получаем ID пользователя из контекста (устанавливается middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
		return
	}

	// Получаем роль пользователя
	userRole, exists := c.Get("roleID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
		return
	}

	// Администратор (role 1) имеет доступ ко всем спискам
	if userRole == 1 {
		plates, err := h.approvedPlateRepo.GetByList(listID)
		if err != nil {
			log.Printf("❌ Ошибка при получении номеров по списку %s: %v", listID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении номеров"})
			return
		}
		c.JSON(http.StatusOK, plates)
		return
	}

	// Для остальных пользователей проверяем права
	hasPermission, err := h.userRepo.CheckListPermission(userID.(string), listID)
	if err != nil {
		log.Printf("❌ Ошибка при проверке прав пользователя %s на список %s: %v", userID, listID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при проверке прав"})
		return
	}

	if !hasPermission {
		log.Printf("❌ Пользователь %s не имеет прав на просмотр списка %s", userID, listID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Недостаточно прав для просмотра этого списка"})
		return
	}

	// Получаем номера по списку
	plates, err := h.approvedPlateRepo.GetByList(listID)
	if err != nil {
		log.Printf("❌ Ошибка при получении номеров по списку %s: %v", listID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении номеров"})
		return
	}

	c.JSON(http.StatusOK, plates)
}

// GetPlatesByListAdmin - для администратора (без проверки прав)
func (h *ApprovedPlateHandler) GetPlatesByListAdmin(c *gin.Context) {
	listID := c.Param("listId")
	if listID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID списка не указан"})
		return
	}

	plates, err := h.approvedPlateRepo.GetByList(listID)
	if err != nil {
		log.Printf("❌ Ошибка при получении номеров по списку %s: %v", listID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении номеров"})
		return
	}

	c.JSON(http.StatusOK, plates)
}
