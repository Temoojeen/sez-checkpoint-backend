package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"sez-checkpoint-backend/internal/repository"
)

type ParticipantHandler struct {
	appRepo      *repository.ApplicationRepository
	approvedRepo *repository.ApprovedPlateRepository
	userRepo     *repository.UserRepository
}

func NewParticipantHandler(
	appRepo *repository.ApplicationRepository,
	approvedRepo *repository.ApprovedPlateRepository,
	userRepo *repository.UserRepository,
) *ParticipantHandler {
	return &ParticipantHandler{
		appRepo:      appRepo,
		approvedRepo: approvedRepo,
		userRepo:     userRepo,
	}
}

// GetDashboard - получает дашборд участника
func (h *ParticipantHandler) GetDashboard(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
		return
	}

	// Получаем заявки участника
	applications, err := h.appRepo.GetByApplicant(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении заявок"})
		return
	}

	// Получаем доступные списки
	lists, err := h.userRepo.GetUserListPermissions(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении списков"})
		return
	}

	// Получаем утвержденные номера участника
	user, err := h.userRepo.GetByID(userID.(string))
	if err == nil && user.OrganizationID != nil {
		plates, _ := h.approvedRepo.GetByOrganization(*user.OrganizationID)

		c.JSON(http.StatusOK, gin.H{
			"applications":    applications,
			"available_lists": lists,
			"approved_plates": plates,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"applications":    applications,
		"available_lists": lists,
	})
}
