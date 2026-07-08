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

type SecurityHandler struct {
	accessLogRepo *repository.AccessLogRepository
	approvedRepo  *repository.ApprovedPlateRepository
}

func NewSecurityHandler(
	accessLogRepo *repository.AccessLogRepository,
	approvedRepo *repository.ApprovedPlateRepository,
) *SecurityHandler {
	return &SecurityHandler{
		accessLogRepo: accessLogRepo,
		approvedRepo:  approvedRepo,
	}
}

// GetRecentLogs - получает последние 5 проездов за сегодня
func (h *SecurityHandler) GetRecentLogs(c *gin.Context) {
	logs, err := h.accessLogRepo.GetRecentToday(5)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении истории"})
		return
	}

	if logs == nil {
		logs = []*models.AccessLog{}
	}

	c.JSON(http.StatusOK, logs)
}

// CheckPlate - проверяет номер в списке пропусков
func (h *SecurityHandler) CheckPlate(c *gin.Context) {
	plateNumber := c.Param("number")

	if plateNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Номер машины не указан",
		})
		return
	}

	log.Printf("🔍 Проверка номера: '%s'", plateNumber)

	plate, err := h.approvedRepo.GetByPlateNumberIncludeInactive(plateNumber)
	if err != nil {
		log.Printf("❌ Номер '%s' не найден: %v", plateNumber, err)
		c.JSON(http.StatusOK, models.CheckPlateResponse{
			Exists:  false,
			Message: "Номер не найден в списке пропусков",
		})
		return
	}

	log.Printf("✅ Номер найден: %+v", plate)
	log.Printf("   OrganizationName: '%s'", plate.OrganizationName)
	log.Printf("   ListName: '%s'", plate.ListName)

	// Проверяем, активен ли номер (с учётом только даты, без времени)
	isActive := plate.IsActive
	if plate.ValidUntil != nil {
		now := time.Now()
		validUntilLocal := plate.ValidUntil.In(now.Location())
		todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		log.Printf("⏰ Проверка даты: validUntil=%v, validUntilLocal=%v, todayStart=%v",
			plate.ValidUntil, validUntilLocal, todayStart)

		// Номер неактивен только если validUntil раньше начала сегодняшнего дня
		if validUntilLocal.Before(todayStart) {
			isActive = false
			log.Printf("❌ Номер неактивен: validUntilLocal < todayStart")
		} else {
			log.Printf("✅ Номер активен: validUntilLocal >= todayStart")
		}
	}

	go h.logAccessAttempt(plateNumber, isActive, plate.ListName)

	if !isActive {
		c.JSON(http.StatusOK, models.CheckPlateResponse{
			Exists:           true,
			PlateNumber:      plate.PlateNumber,
			OrganizationName: plate.OrganizationName,
			ListName:         plate.ListName,
			ListType:         plate.ListType,
			ListColor:        plate.ListColor,
			Message:          "Номер найден, но неактивен. Обратитесь к администратору",
			IsActive:         false,
		})
		return
	}

	response := models.CheckPlateResponse{
		Exists:           true,
		PlateNumber:      plate.PlateNumber,
		OrganizationName: plate.OrganizationName,
		ListName:         plate.ListName,
		ListType:         plate.ListType,
		ListColor:        plate.ListColor,
		Message:          "Номер найден, доступ разрешен",
		IsActive:         true,
	}

	if plate.ValidUntil != nil {
		response.ValidUntil = plate.ValidUntil.Format("2006-01-02")
	}

	c.JSON(http.StatusOK, response)
}

// LogAccess - запись проезда (будет вызываться камерой)
func (h *SecurityHandler) LogAccess(c *gin.Context) {
	var req models.CreateAccessLogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неверный формат данных",
		})
		return
	}

	plate, err := h.approvedRepo.GetByPlateNumber(req.PlateNumber)
	accessGranted := err == nil

	log := &models.AccessLog{
		ID:               uuid.New().String(),
		PlateNumber:      req.PlateNumber,
		OrganizationName: req.OrganizationName,
		ListName:         req.ListName,
		ImagePath:        req.ImagePath,
		AccessGranted:    accessGranted,
		CameraID:         req.CameraID,
		CameraLocation:   req.CameraLocation,
		CreatedAt:        time.Now(),
	}

	if accessGranted {
		log.OrganizationName = plate.OrganizationName
		log.ListName = plate.ListName
	}

	if err := h.accessLogRepo.Create(log); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка при сохранении записи",
		})
		return
	}

	c.JSON(http.StatusOK, log)
}

// logAccessAttempt - вспомогательный метод для логирования
func (h *SecurityHandler) logAccessAttempt(plateNumber string, granted bool, listName string) {
	log := &models.AccessLog{
		ID:            uuid.New().String(),
		PlateNumber:   plateNumber,
		ListName:      listName,
		AccessGranted: granted,
		CreatedAt:     time.Now(),
	}

	go h.accessLogRepo.Create(log)
}

// GetStatistics - получает статистику проездов (только за сегодня)
func (h *SecurityHandler) GetStatistics(c *gin.Context) {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Second)

	logs, err := h.accessLogRepo.GetByDateRange(startOfDay, endOfDay)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении статистики"})
		return
	}

	total := len(logs)
	granted := 0
	denied := 0

	for _, log := range logs {
		if log.AccessGranted {
			granted++
		} else {
			denied++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"period": gin.H{
			"from": startOfDay.Format("2006-01-02"),
			"to":   endOfDay.Format("2006-01-02"),
		},
		"statistics": gin.H{
			"total":   total,
			"granted": granted,
			"denied":  denied,
		},
		"logs": logs,
	})
}

// GetLogsByPlate - получает историю проездов по номеру
func (h *SecurityHandler) GetLogsByPlate(c *gin.Context) {
	plateNumber := c.Param("number")

	if plateNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Номер не указан"})
		return
	}

	logs, err := h.accessLogRepo.GetByPlateNumber(plateNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении истории"})
		return
	}

	c.JSON(http.StatusOK, logs)
}

// GetAllLogs - получает все логи с фильтрацией по датам и номеру (для админа)
func (h *SecurityHandler) GetAllLogs(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")
	plateNumber := c.Query("plateNumber")

	var fromTime, toTime time.Time
	var err error

	if from == "" && to == "" {
		toTime = time.Now()
		fromTime = toTime.AddDate(0, 0, -1)
	} else {
		if from != "" {
			fromTime, err = time.Parse("2006-01-02", from)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат даты from"})
				return
			}
		} else {
			fromTime = time.Now().AddDate(0, 0, -7)
		}

		if to != "" {
			toTime, err = time.Parse("2006-01-02", to)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат даты to"})
				return
			}
			toTime = toTime.Add(24 * time.Hour).Add(-time.Second)
		} else {
			toTime = time.Now()
		}
	}

	logs, err := h.accessLogRepo.GetByDateRangeAndPlate(fromTime, toTime, plateNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении логов"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"total": len(logs),
		"period": gin.H{
			"from": fromTime.Format("2006-01-02"),
			"to":   toTime.Format("2006-01-02"),
		},
	})
}

// SearchPlates - поиск номеров по частичному совпадению
func (h *SecurityHandler) SearchPlates(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Не указан поисковый запрос"})
		return
	}

	if len(query) < 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Минимум 3 символа для поиска"})
		return
	}

	log.Printf("🔍 Поиск номеров по запросу: %s", query)

	plates, err := h.approvedRepo.SearchByPartialPlate(query)
	if err != nil {
		log.Printf("❌ Ошибка при поиске номеров: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при поиске номеров"})
		return
	}

	if plates == nil {
		plates = []map[string]interface{}{}
	}

	c.JSON(http.StatusOK, plates)
}
