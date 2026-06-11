package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"sez-checkpoint-backend/internal/models"
	"sez-checkpoint-backend/internal/repository"
)

type PassManagerHandler struct {
	approvedRepo     *repository.ApprovedPlateRepository
	accessListRepo   *repository.AccessListRepository
	userRepo         *repository.UserRepository
	deletedPlateRepo *repository.DeletedPlateRepository
	organizationRepo *repository.OrganizationRepository
}

func NewPassManagerHandler(
	approvedRepo *repository.ApprovedPlateRepository,
	accessListRepo *repository.AccessListRepository,
	userRepo *repository.UserRepository,
	deletedPlateRepo *repository.DeletedPlateRepository,
	organizationRepo *repository.OrganizationRepository,
) *PassManagerHandler {
	return &PassManagerHandler{
		approvedRepo:     approvedRepo,
		accessListRepo:   accessListRepo,
		userRepo:         userRepo,
		deletedPlateRepo: deletedPlateRepo,
		organizationRepo: organizationRepo,
	}
}

// GetAllLists - получает ВСЕ активные списки
func (h *PassManagerHandler) GetAllLists(c *gin.Context) {
	lists, err := h.accessListRepo.GetAll(true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении списков"})
		return
	}
	if lists == nil {
		lists = []*models.AccessList{}
	}
	c.JSON(http.StatusOK, lists)
}

// GetPlatesByList - получение номеров по списку
func (h *PassManagerHandler) GetPlatesByList(c *gin.Context) {
	listID := c.Param("listId")
	plates, err := h.approvedRepo.GetAll("", listID, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении номеров"})
		return
	}
	c.JSON(http.StatusOK, plates)
}

// GetAllPlates - все номера во всех списках
func (h *PassManagerHandler) GetAllPlates(c *gin.Context) {
	plates, err := h.approvedRepo.GetAll("", "", false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении номеров"})
		return
	}
	c.JSON(http.StatusOK, plates)
}

// GetOrganizations - список организаций для выбора
func (h *PassManagerHandler) GetOrganizations(c *gin.Context) {
	orgs, err := h.organizationRepo.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении организаций"})
		return
	}
	c.JSON(http.StatusOK, orgs)
}

// AddPlate - добавление номера
func (h *PassManagerHandler) AddPlate(c *gin.Context) {
	var req struct {
		PlateNumber    string `json:"plateNumber" binding:"required"`
		VehicleBrand   string `json:"vehicleBrand"`
		VehicleModel   string `json:"vehicleModel"`
		VehicleColor   string `json:"vehicleColor"`
		ListID         string `json:"listId" binding:"required"`
		OrganizationID string `json:"organizationId"`
		WithoutOrg     bool   `json:"withoutOrg"`
		Notes          string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	// Проверяем, существует ли уже такой номер в этом списке
	existingPlate, err := h.approvedRepo.GetByPlateNumberAndListIncludeInactive(req.PlateNumber, req.ListID)
	if err == nil && existingPlate != nil {
		if !existingPlate.IsActive {
			existingPlate.IsActive = true
			existingPlate.UpdatedAt = time.Now()
			if req.VehicleBrand != "" {
				existingPlate.VehicleBrand = req.VehicleBrand
			}
			if req.VehicleModel != "" {
				existingPlate.VehicleModel = req.VehicleModel
			}
			if req.VehicleColor != "" {
				existingPlate.VehicleColor = req.VehicleColor
			}
			if req.Notes != "" {
				existingPlate.Notes = req.Notes
			}
			if err := h.approvedRepo.Update(existingPlate); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при реактивации номера"})
				return
			}
			log.Printf("✅ Номер %s реактивирован менеджером пропусков", req.PlateNumber)
			c.JSON(http.StatusOK, existingPlate)
			return
		}
		c.JSON(http.StatusConflict, gin.H{"error": "Такой номер уже есть в этом списке"})
		return
	}

	userID, _ := c.Get("userID")
	userIDStr := userID.(string)

	plate := &models.ApprovedPlate{
		ID:           uuid.New().String(),
		PlateNumber:  req.PlateNumber,
		VehicleBrand: req.VehicleBrand,
		VehicleModel: req.VehicleModel,
		VehicleColor: req.VehicleColor,
		ListID:       req.ListID,
		ApprovedBy:   &userIDStr,
		IsActive:     true,
		Notes:        req.Notes,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Если указана организация — используем её
	if req.OrganizationID != "" && !req.WithoutOrg {
		plate.OrganizationID = &req.OrganizationID
	} else if req.WithoutOrg {
		// Ищем организацию "Гость"
		guestOrg, err := h.organizationRepo.GetByBIN("GUEST")
		if err == nil && guestOrg != nil {
			plate.OrganizationID = &guestOrg.ID
		}
	}

	if err := h.approvedRepo.Create(plate); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при добавлении номера"})
		return
	}

	log.Printf("✅ Номер %s добавлен менеджером пропусков в список %s", req.PlateNumber, req.ListID)
	c.JSON(http.StatusCreated, plate)
}

// UpdatePlate - обновление номера
func (h *PassManagerHandler) UpdatePlate(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		PlateNumber    string `json:"plateNumber"`
		VehicleBrand   string `json:"vehicleBrand"`
		VehicleModel   string `json:"vehicleModel"`
		VehicleColor   string `json:"vehicleColor"`
		ListID         string `json:"listId"`
		OrganizationID string `json:"organizationId"`
		WithoutOrg     *bool  `json:"withoutOrg"`
		Notes          string `json:"notes"`
		IsActive       *bool  `json:"isActive"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	plate, err := h.approvedRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Номер не найден"})
		return
	}

	if req.PlateNumber != "" {
		plate.PlateNumber = req.PlateNumber
	}
	if req.VehicleBrand != "" {
		plate.VehicleBrand = req.VehicleBrand
	}
	if req.VehicleModel != "" {
		plate.VehicleModel = req.VehicleModel
	}
	if req.VehicleColor != "" {
		plate.VehicleColor = req.VehicleColor
	}
	if req.ListID != "" {
		plate.ListID = req.ListID
	}
	if req.Notes != "" {
		plate.Notes = req.Notes
	}
	if req.IsActive != nil {
		plate.IsActive = *req.IsActive
	}

	if req.OrganizationID != "" {
		plate.OrganizationID = &req.OrganizationID
	} else if req.WithoutOrg != nil && *req.WithoutOrg {
		guestOrg, err := h.organizationRepo.GetByBIN("GUEST")
		if err == nil && guestOrg != nil {
			plate.OrganizationID = &guestOrg.ID
		} else {
			plate.OrganizationID = nil
		}
	}

	plate.UpdatedAt = time.Now()

	if err := h.approvedRepo.Update(plate); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении номера"})
		return
	}

	c.JSON(http.StatusOK, plate)
}

// DeletePlate - удаление номера с обязательной причиной
func (h *PassManagerHandler) DeletePlate(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Укажите причину удаления"})
		return
	}

	plate, err := h.approvedRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Номер не найден"})
		return
	}

	userID, _ := c.Get("userID")
	userIDStr := userID.(string)

	// Получаем имя пользователя
	user, err := h.userRepo.GetByID(userIDStr)
	userName := ""
	if err == nil && user != nil {
		userName = user.FullName
	}

	// Сохраняем в историю удаленных
	deletedPlate := &models.DeletedPlate{
		ID:              uuid.New().String(),
		PlateNumber:     plate.PlateNumber,
		VehicleBrand:    plate.VehicleBrand,
		VehicleModel:    plate.VehicleModel,
		VehicleColor:    plate.VehicleColor,
		OrganizationID:  plate.OrganizationID,
		ListID:          plate.ListID,
		ListName:        plate.ListName,
		DeletedBy:       &userIDStr,
		DeletedByName:   userName,
		DeleteReason:    req.Reason,
		OriginalPlateID: &id,
		CreatedAt:       time.Now(),
		DeletedAt:       time.Now(),
	}
	_ = h.deletedPlateRepo.Create(deletedPlate)

	if err := h.approvedRepo.HardDelete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении номера"})
		return
	}

	log.Printf("🗑️ Номер %s удален менеджером пропусков %s. Причина: %s", plate.PlateNumber, userName, req.Reason)
	c.JSON(http.StatusOK, gin.H{"message": "Номер удален"})
}
