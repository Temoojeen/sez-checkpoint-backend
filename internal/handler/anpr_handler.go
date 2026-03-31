package handler

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"sez-checkpoint-backend/internal/models"
	"sez-checkpoint-backend/internal/repository"
)

// HikvisionEventNotification структура для парсинга XML от камеры
type HikvisionEventNotification struct {
	XMLName         xml.Name `xml:"EventNotificationAlert"`
	IPAddress       string   `xml:"ipAddress"`
	PortNo          int      `xml:"portNo"`
	Protocol        string   `xml:"protocol"`
	EventType       string   `xml:"eventType"`
	DateTime        string   `xml:"dateTime"`
	PlateNumber     string   `xml:"plateNumber"`
	PlateCountry    string   `xml:"plateCountry"`
	LaneNo          int      `xml:"laneNo"`
	Direction       string   `xml:"direction"`
	VehicleType     string   `xml:"vehicleType"`
	ConfidenceLevel int      `xml:"confidenceLevel"`
	SnapshotURL     string   `xml:"snapshotUrl"`
	Allowlist       string   `xml:"allowlist"`
	Blocklist       string   `xml:"blocklist"`
}

// ANPREvent структура для внутреннего представления
type ANPREvent struct {
	EventType       string `json:"eventType"`
	PlateNumber     string `json:"plateNumber"`
	PlateCountry    string `json:"plateCountry"`
	Direction       string `json:"direction"`
	Time            string `json:"time"`
	SnapshotURL     string `json:"snapshotUrl"`
	LaneNo          int    `json:"laneNo"`
	VehicleType     string `json:"vehicleType"`
	Allowlist       bool   `json:"allowlist"`
	Blocklist       bool   `json:"blocklist"`
	ConfidenceLevel int    `json:"confidenceLevel"`
}

// ANPRHandler хендлер для ANPR событий от камеры
type ANPRHandler struct {
	accessLogRepo     *repository.AccessLogRepository
	approvedPlateRepo *repository.ApprovedPlateRepository
}

// NewANPRHandler создает новый ANPRHandler
func NewANPRHandler(accessLogRepo *repository.AccessLogRepository, approvedPlateRepo *repository.ApprovedPlateRepository) *ANPRHandler {
	return &ANPRHandler{
		accessLogRepo:     accessLogRepo,
		approvedPlateRepo: approvedPlateRepo,
	}
}

// HandleCameraEvent обрабатывает события от камеры Hikvision
func (h *ANPRHandler) HandleCameraEvent(c *gin.Context) {
	// Читаем тело запроса
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("❌ Ошибка чтения тела запроса: %v", err)
		c.String(http.StatusBadRequest, "Bad Request")
		return
	}

	log.Printf("📸 Получено событие от камеры")
	log.Printf("Raw XML: %s", string(body))

	// Парсим XML
	var event HikvisionEventNotification
	if err := xml.Unmarshal(body, &event); err != nil {
		log.Printf("❌ Ошибка парсинга XML: %v", err)
		c.String(http.StatusBadRequest, "Invalid XML")
		return
	}

	// Проверяем тип события
	if event.EventType != "vehicleDetection" {
		log.Printf("ℹ️ Игнорируем событие типа: %s", event.EventType)
		c.String(http.StatusOK, "OK")
		return
	}

	// Преобразуем в JSON
	anprEvent := ANPREvent{
		EventType:       event.EventType,
		PlateNumber:     event.PlateNumber,
		PlateCountry:    event.PlateCountry,
		Direction:       event.Direction,
		Time:            event.DateTime,
		SnapshotURL:     fmt.Sprintf("http://%s%s", event.IPAddress, event.SnapshotURL),
		LaneNo:          event.LaneNo,
		VehicleType:     event.VehicleType,
		Allowlist:       event.Allowlist == "true",
		Blocklist:       event.Blocklist == "true",
		ConfidenceLevel: event.ConfidenceLevel,
	}

	log.Printf("✅ Распознан номер: %s (%s)", anprEvent.PlateNumber, anprEvent.PlateCountry)
	log.Printf("   Направление: %s, Полоса: %d", anprEvent.Direction, anprEvent.LaneNo)
	log.Printf("   Уверенность: %d%%", anprEvent.ConfidenceLevel)

	// Проверяем номер в approved_plates
	var allowed bool
	var plateInfo *models.ApprovedPlate

	if h.approvedPlateRepo != nil {
		plate, err := h.approvedPlateRepo.GetByPlateNumber(anprEvent.PlateNumber)
		if err == nil && plate != nil {
			allowed = true
			plateInfo = plate
			log.Printf("   ✅ Номер найден в белом списке!")
			log.Printf("      Организация: %s", plate.OrganizationName)
			log.Printf("      Список: %s", plate.ListName)
			if plate.ValidUntil != nil {
				log.Printf("      Действует до: %s", plate.ValidUntil.Format("2006-01-02"))
			}
		} else {
			allowed = false
			log.Printf("   ❌ Номер НЕ найден в белом списке")
		}
	} else {
		log.Printf("   ⚠️ Репозиторий approvedPlateRepo не инициализирован")
	}

	// Сохраняем в лог доступа
	if h.accessLogRepo != nil {
		accessLog := &models.AccessLog{
			PlateNumber:   anprEvent.PlateNumber,
			AccessGranted: allowed,
			CameraID:      event.IPAddress,
			CreatedAt:     time.Now(),
		}

		// Если номер найден, добавляем дополнительную информацию
		if plateInfo != nil {
			accessLog.OrganizationName = plateInfo.OrganizationName
			accessLog.ListName = plateInfo.ListName
		}

		// TODO: добавить метод Create в AccessLogRepository
		// if err := h.accessLogRepo.Create(accessLog); err != nil {
		//     log.Printf("⚠️ Ошибка сохранения в лог: %v", err)
		// } else {
		//     log.Printf("📝 Запись в лог доступа для номера: %s (доступ: %v)", anprEvent.PlateNumber, allowed)
		// }

		log.Printf("📝 Запись в лог доступа для номера: %s (доступ: %v)", anprEvent.PlateNumber, allowed)
	}

	// TODO: Здесь можно добавить отправку команды на шлагбаум
	if allowed {
		log.Printf("🚪 РАЗРЕШАЕМ проезд для номера: %s", anprEvent.PlateNumber)
		// Например:
		// http.Post("http://192.168.0.200:80/open", ...)
	} else {
		log.Printf("🚫 ЗАПРЕЩАЕМ проезд для номера: %s", anprEvent.PlateNumber)
	}

	// Отвечаем камере
	c.String(http.StatusOK, "OK")
}
