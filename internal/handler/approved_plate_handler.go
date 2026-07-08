package handler

import (
	"log"
	"net/http"

	"sez-checkpoint-backend/internal/models"
	"sez-checkpoint-backend/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ApprovedPlateHandler struct {
	approvedPlateRepo *repository.ApprovedPlateRepository
	userRepo          *repository.UserRepository
	deletedPlateRepo  *repository.DeletedPlateRepository
}

func NewApprovedPlateHandler(
	approvedPlateRepo *repository.ApprovedPlateRepository,
	userRepo *repository.UserRepository,
	deletedPlateRepo *repository.DeletedPlateRepository,
) *ApprovedPlateHandler {
	return &ApprovedPlateHandler{
		approvedPlateRepo: approvedPlateRepo,
		userRepo:          userRepo,
		deletedPlateRepo:  deletedPlateRepo,
	}
}

// GetPlatesByList - получение номеров по списку
func (h *ApprovedPlateHandler) GetPlatesByList(c *gin.Context) {
	listID := c.Param("listId")
	if listID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID списка не указан"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
		return
	}

	userRole, exists := c.Get("roleID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
		return
	}

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

	// Для участников показываем все номера, включая неактивные
	plates, err := h.approvedPlateRepo.GetAll("", listID, false)
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

// DeleteByParticipant - удаление номера участником с указанием причины
func (h *ApprovedPlateHandler) DeleteByParticipant(c *gin.Context) {
	plateID := c.Param("id")

	var req struct {
		Reason string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.Reason = ""
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
		return
	}
	userIDStr := userID.(string)

	user, err := h.userRepo.GetByID(userIDStr)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	if user.OrganizationID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "У вас нет организации"})
		return
	}

	plate, err := h.approvedPlateRepo.GetByID(plateID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Номер не найден"})
		return
	}

	if plate.OrganizationID == nil || *plate.OrganizationID != *user.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Нет прав на удаление этого номера"})
		return
	}

	log.Printf("🗑️ Участник %s удаляет номер %s, причина: %s", userIDStr, plate.PlateNumber, req.Reason)

	deletedPlate := &models.DeletedPlate{
		ID:               uuid.New().String(),
		PlateNumber:      plate.PlateNumber,
		VehicleBrand:     plate.VehicleBrand,
		VehicleModel:     plate.VehicleModel,
		VehicleColor:     plate.VehicleColor,
		OrganizationID:   plate.OrganizationID,
		OrganizationName: plate.OrganizationName,
		ListID:           plate.ListID,
		ListName:         plate.ListName,
		DeletedBy:        &userIDStr,
		DeletedByName:    user.FullName,
		DeleteReason:     req.Reason,
		OriginalPlateID:  &plateID,
	}

	if err := h.deletedPlateRepo.Create(deletedPlate); err != nil {
		log.Printf("⚠️ Ошибка при сохранении в историю удалений: %v", err)
	}

	if err := h.approvedPlateRepo.HardDelete(plateID); err != nil {
		log.Printf("❌ Ошибка при удалении номера: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении номера"})
		return
	}

	log.Printf("✅ Номер %s полностью удален из базы", plate.PlateNumber)
	c.JSON(http.StatusOK, gin.H{"message": "Номер успешно удалён"})
}
