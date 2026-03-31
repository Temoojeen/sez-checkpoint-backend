package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"sez-checkpoint-backend/internal/repository"
)

type OperatorHandler struct {
	appRepo *repository.ApplicationRepository
}

func NewOperatorHandler(appRepo *repository.ApplicationRepository) *OperatorHandler {
	return &OperatorHandler{
		appRepo: appRepo,
	}
}

// GetDashboard - получает дашборд оператора
func (h *OperatorHandler) GetDashboard(c *gin.Context) {
	pending, err := h.appRepo.GetPendingForOperator()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении заявок"})
		return
	}

	stats, err := h.appRepo.GetStats("")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении статистики"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pending_applications": pending,
		"statistics":           stats,
	})
}
