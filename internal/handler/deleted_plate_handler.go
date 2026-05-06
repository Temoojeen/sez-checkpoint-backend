package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"sez-checkpoint-backend/internal/repository"
)

type DeletedPlateHandler struct {
	deletedPlateRepo *repository.DeletedPlateRepository
}

func NewDeletedPlateHandler(deletedPlateRepo *repository.DeletedPlateRepository) *DeletedPlateHandler {
	return &DeletedPlateHandler{
		deletedPlateRepo: deletedPlateRepo,
	}
}

// GetAll - получение всех удаленных номеров (для админа)
func (h *DeletedPlateHandler) GetAll(c *gin.Context) {
	plates, err := h.deletedPlateRepo.GetAll()
	if err != nil {
		log.Printf("❌ Ошибка при получении удаленных номеров: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении данных"})
		return
	}

	c.JSON(http.StatusOK, plates)
}
