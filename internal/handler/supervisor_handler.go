package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"sez-checkpoint-backend/internal/repository"
)

type SupervisorHandler struct {
	appRepo      *repository.ApplicationRepository
	approvedRepo *repository.ApprovedPlateRepository
}

func NewSupervisorHandler(
	appRepo *repository.ApplicationRepository,
	approvedRepo *repository.ApprovedPlateRepository,
) *SupervisorHandler {
	return &SupervisorHandler{
		appRepo:      appRepo,
		approvedRepo: approvedRepo,
	}
}

// GetDashboard - получает дашборд руководителя
func (h *SupervisorHandler) GetDashboard(c *gin.Context) {
	pending, err := h.appRepo.GetPendingForSupervisor()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении заявок"})
		return
	}

	expiringSoon, err := h.approvedRepo.GetExpiringSoon(30)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении истекающих номеров"})
		return
	}

	stats, err := h.appRepo.GetStats("")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении статистики"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pending_applications": pending,
		"expiring_soon":        expiringSoon,
		"statistics":           stats,
	})
}
