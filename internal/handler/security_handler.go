package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"sez-checkpoint-backend/internal/models" // Этот импорт должен быть
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

// GetRecentLogs - получает последние 5 проездов
func (h *SecurityHandler) GetRecentLogs(c *gin.Context) {
	logs, err := h.accessLogRepo.GetRecent(5)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка при получении истории",
		})
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

	// Ищем номер в утвержденных (включая неактивные)
	plate, err := h.approvedRepo.GetByPlateNumberIncludeInactive(plateNumber)
	if err != nil {
		// Логируем попытку проезда с неразрешенным номером
		h.logAccessAttempt(plateNumber, false, "")

		c.JSON(http.StatusOK, models.CheckPlateResponse{
			Exists:  false,
			Message: "Номер не найден в списке пропусков",
		})
		return
	}

	// Проверяем, активен ли номер
	isActive := plate.IsActive
	if plate.ValidUntil != nil && plate.ValidUntil.Before(time.Now()) {
		isActive = false
	}

	// Логируем попытку проезда
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

	// Проверяем номер в списке пропусков
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

	// Если номер найден, используем данные из БД
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

	// Асинхронно сохраняем в БД
	go h.accessLogRepo.Create(log)
}

// GetStatistics - получает статистику проездов
func (h *SecurityHandler) GetStatistics(c *gin.Context) {
	// Получаем параметры дат из запроса
	from := c.Query("from")
	to := c.Query("to")

	var fromTime, toTime time.Time
	var err error

	if from != "" {
		fromTime, err = time.Parse("2006-01-02", from)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат даты from"})
			return
		}
	} else {
		fromTime = time.Now().AddDate(0, 0, -7) // по умолчанию за неделю
	}

	if to != "" {
		toTime, err = time.Parse("2006-01-02", to)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат даты to"})
			return
		}
		toTime = toTime.Add(24 * time.Hour).Add(-time.Second) // конец дня
	} else {
		toTime = time.Now()
	}

	// Получаем логи за период
	logs, err := h.accessLogRepo.GetByDateRange(fromTime, toTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении статистики"})
		return
	}

	// Считаем статистику
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
			"from": fromTime.Format("2006-01-02"),
			"to":   toTime.Format("2006-01-02"),
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
