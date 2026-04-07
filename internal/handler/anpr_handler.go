package handler

import (
	"encoding/xml"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"sez-checkpoint-backend/internal/models"
	"sez-checkpoint-backend/internal/repository"
	"sez-checkpoint-backend/internal/websocket"
)

// ANPRData структура для данных внутри тега ANPR
type ANPRData struct {
	Country         string `xml:"country"`
	LicensePlate    string `xml:"licensePlate"`
	Line            int    `xml:"line"`
	Direction       string `xml:"direction"`
	ConfidenceLevel int    `xml:"confidenceLevel"`
	VehicleType     string `xml:"vehicleType"`
	PlateColor      string `xml:"plateColor"`
	OriginalLicense string `xml:"originalLicensePlate"`
}

// HikvisionEventNotification структура для парсинга XML от камеры
type HikvisionEventNotification struct {
	XMLName     xml.Name `xml:"EventNotificationAlert"`
	IPAddress   string   `xml:"ipAddress"`
	PortNo      int      `xml:"portNo"`
	Protocol    string   `xml:"protocol"`
	EventType   string   `xml:"eventType"`
	DateTime    string   `xml:"dateTime"`
	ANPR        ANPRData `xml:"ANPR"`
	ChannelID   int      `xml:"channelID"`
	ChannelName string   `xml:"channelName"`
}

// ANPRHandler хендлер для ANPR событий от камеры
type ANPRHandler struct {
	accessLogRepo     *repository.AccessLogRepository
	approvedPlateRepo *repository.ApprovedPlateRepository
	websocketHub      *websocket.Hub
}

// NewANPRHandler создает новый ANPRHandler
func NewANPRHandler(
	accessLogRepo *repository.AccessLogRepository,
	approvedPlateRepo *repository.ApprovedPlateRepository,
	websocketHub *websocket.Hub,
) *ANPRHandler {
	return &ANPRHandler{
		accessLogRepo:     accessLogRepo,
		approvedPlateRepo: approvedPlateRepo,
		websocketHub:      websocketHub,
	}
}

// HandleCameraEvent обрабатывает события от камеры Hikvision
func (h *ANPRHandler) HandleCameraEvent(c *gin.Context) {
	// Получаем Content-Type
	contentType := c.GetHeader("Content-Type")
	log.Printf("📸 Получен запрос от камеры, Content-Type: %s", contentType)

	// Читаем multipart form data
	err := c.Request.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		log.Printf("❌ Ошибка парсинга multipart form: %v", err)
		c.String(http.StatusBadRequest, "Bad Request")
		return
	}

	// Ищем XML часть
	var xmlData []byte

	for key, headers := range c.Request.MultipartForm.File {
		log.Printf("Файл в запросе: key=%s", key)

		for _, header := range headers {
			file, err := header.Open()
			if err != nil {
				log.Printf("❌ Ошибка открытия файла: %v", err)
				continue
			}
			defer file.Close()

			data, err := io.ReadAll(file)
			if err != nil {
				log.Printf("❌ Ошибка чтения файла: %v", err)
				continue
			}

			// Если это XML файл
			if header.Filename == "anpr.xml" {
				xmlData = data
				log.Printf("📄 Найден XML файл: %s, размер: %d байт", header.Filename, len(xmlData))
			} else {
				// Это изображение
				log.Printf("🖼️ Найдено изображение: %s, размер: %d байт", header.Filename, len(data))
			}
		}
	}

	if len(xmlData) == 0 {
		log.Printf("❌ Не найден XML в запросе")
		c.String(http.StatusBadRequest, "No XML data found")
		return
	}

	// Парсим XML
	var event HikvisionEventNotification
	if err := xml.Unmarshal(xmlData, &event); err != nil {
		log.Printf("❌ Ошибка парсинга XML: %v", err)
		log.Printf("XML данные: %s", string(xmlData))
		c.String(http.StatusBadRequest, "Invalid XML")
		return
	}

	// Проверяем тип события
	if event.EventType != "vehicleDetection" && event.EventType != "ANPR" {
		log.Printf("ℹ️ Игнорируем событие типа: %s", event.EventType)
		c.String(http.StatusOK, "OK")
		return
	}

	// Получаем номер из ANPR структуры
	plateNumber := event.ANPR.LicensePlate
	plateCountry := event.ANPR.Country
	direction := event.ANPR.Direction
	laneNo := event.ANPR.Line
	confidence := event.ANPR.ConfidenceLevel
	vehicleType := event.ANPR.VehicleType

	log.Printf("✅ Распознан номер: %s (%s)", plateNumber, plateCountry)
	log.Printf("   Направление: %s, Полоса: %d", direction, laneNo)
	log.Printf("   Уверенность: %d%%", confidence)
	log.Printf("   Тип ТС: %s, Цвет номера: %s", vehicleType, event.ANPR.PlateColor)

	// Проверяем номер в approved_plates
	var allowed bool
	var plateInfo *models.ApprovedPlate
	var message string

	if h.approvedPlateRepo != nil && plateNumber != "" {
		plate, err := h.approvedPlateRepo.GetByPlateNumberIncludeInactive(plateNumber)
		if err == nil && plate != nil {
			// Проверяем активность
			isActive := plate.IsActive
			if plate.ValidUntil != nil && plate.ValidUntil.Before(time.Now()) {
				isActive = false
			}

			if isActive {
				allowed = true
				message = "Доступ разрешен"
				log.Printf("   ✅ Номер найден в белом списке!")
			} else {
				allowed = false
				message = "Номер неактивен"
				log.Printf("   ⚠️ Номер найден, но неактивен")
			}

			plateInfo = plate

			if plate.OrganizationName != "" {
				log.Printf("      Организация: %s", plate.OrganizationName)
			}
			if plate.ListName != "" {
				log.Printf("      Список: %s", plate.ListName)
			}
		} else {
			allowed = false
			message = "Номер не найден в списке пропусков"
			log.Printf("   ❌ Номер НЕ найден в белом списке")
		}
	} else if plateNumber == "" {
		log.Printf("   ⚠️ Номер не распознан")
		message = "Номер не распознан"
	}

	// Отправляем WebSocket событие на фронтенд
	if h.websocketHub != nil {
		organizationName := ""
		listName := ""
		listColor := ""
		if plateInfo != nil {
			organizationName = plateInfo.OrganizationName
			listName = plateInfo.ListName
			listColor = plateInfo.ListColor
		}

		h.websocketHub.SendPlateEvent(
			plateNumber,
			allowed,
			organizationName,
			listName,
			listColor,
			message,
		)
	}

	// Сохраняем в лог доступа
	if h.accessLogRepo != nil {
		// Генерируем UUID для лога
		logID := uuid.New().String()

		accessLog := &models.AccessLog{
			ID:               logID,
			PlateNumber:      plateNumber,
			AccessGranted:    allowed,
			CameraID:         event.IPAddress,
			CameraLocation:   event.ChannelName,
			OrganizationName: "",
			ListName:         "",
			CreatedAt:        time.Now(),
		}

		if plateInfo != nil {
			accessLog.OrganizationName = plateInfo.OrganizationName
			accessLog.ListName = plateInfo.ListName
		}

		if err := h.accessLogRepo.Create(accessLog); err != nil {
			log.Printf("❌ Ошибка сохранения лога доступа: %v", err)
		} else {
			log.Printf("📝 Лог доступа сохранен: ID=%s, номер=%s, доступ=%v", logID, plateNumber, allowed)
		}
	}

	// Отправляем команду на шлагбаум (если нужно)
	if allowed && plateNumber != "" {
		log.Printf("🚪 РАЗРЕШАЕМ проезд для номера: %s", plateNumber)
		// Здесь можно добавить отправку команды на шлагбаум
	} else if plateNumber != "" {
		log.Printf("🚫 ЗАПРЕЩАЕМ проезд для номера: %s", plateNumber)
	}

	c.String(http.StatusOK, "OK")
}
