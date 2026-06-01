package handler

import (
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"

	"sez-checkpoint-backend/internal/models"
	"sez-checkpoint-backend/internal/repository"
)

type ImportHandler struct {
	approvedPlateRepo *repository.ApprovedPlateRepository
	accessListRepo    *repository.AccessListRepository
}

func NewImportHandler(
	approvedPlateRepo *repository.ApprovedPlateRepository,
	accessListRepo *repository.AccessListRepository,
) *ImportHandler {
	return &ImportHandler{
		approvedPlateRepo: approvedPlateRepo,
		accessListRepo:    accessListRepo,
	}
}

// ImportPlatesFromExcel - импорт номеров из Excel файла
func (h *ImportHandler) ImportPlatesFromExcel(c *gin.Context) {
	// Логируем Content-Type
	contentType := c.GetHeader("Content-Type")
	log.Printf("📋 Content-Type: %s", contentType)

	// Парсим multipart форму
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		log.Printf("❌ Ошибка парсинга формы: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Ошибка обработки формы: %v", err)})
		return
	}

	// Получаем файл напрямую из MultipartForm
	var file multipart.File
	var header *multipart.FileHeader
	var err error

	if c.Request.MultipartForm != nil && c.Request.MultipartForm.File != nil {
		// Логируем все поля формы
		log.Printf("📋 Поля формы:")
		for key, values := range c.Request.MultipartForm.Value {
			log.Printf("   Поле: %s = %v", key, values)
		}
		for key, headers := range c.Request.MultipartForm.File {
			for _, h := range headers {
				log.Printf("   Файл: %s = %s (%d байт)", key, h.Filename, h.Size)
			}
		}

		// Пробуем получить файл с ключом "file"
		if fileHeaders, ok := c.Request.MultipartForm.File["file"]; ok && len(fileHeaders) > 0 {
			header = fileHeaders[0]
			file, err = fileHeaders[0].Open()
			if err != nil {
				log.Printf("❌ Ошибка открытия файла: %v", err)
			}
		} else {
			// Если ключ "file" не найден, берем первый попавшийся файл
			for key, fileHeaders := range c.Request.MultipartForm.File {
				if len(fileHeaders) > 0 {
					log.Printf("📎 Использую первый найденный файл с ключом '%s'", key)
					header = fileHeaders[0]
					file, err = fileHeaders[0].Open()
					if err != nil {
						log.Printf("❌ Ошибка открытия файла: %v", err)
					}
					break
				}
			}
		}
	}

	if file == nil {
		log.Printf("❌ Файл не найден в запросе")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Файл не найден. Используйте ключ 'file' и тип File"})
		return
	}
	defer file.Close()

	log.Printf("📄 Получен файл: %s, размер: %d байт", header.Filename, header.Size)

	// Читаем Excel файл
	f, err := excelize.OpenReader(file)
	if err != nil {
		log.Printf("❌ Ошибка открытия Excel файла: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Не удалось прочитать Excel файл"})
		return
	}
	defer f.Close()

	// Получаем первый лист
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Excel файл пустой"})
		return
	}

	sheetName := sheets[0]
	log.Printf("📊 Чтение листа: %s", sheetName)

	// Получаем все строки
	rows, err := f.GetRows(sheetName)
	if err != nil {
		log.Printf("❌ Ошибка чтения строк: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Не удалось прочитать данные из Excel"})
		return
	}

	if len(rows) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Excel файл не содержит данных (только заголовки)"})
		return
	}

	// Ищем или создаем список "Старая база КПП-1"
	listName := "Старая база КПП-1"
	list, err := h.accessListRepo.GetByName(listName)
	if err != nil {
		log.Printf("📝 Список '%s' не найден, создаем новый", listName)

		adminID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
			return
		}
		adminIDStr := adminID.(string)

		list = &models.AccessList{
			ID:          uuid.New().String(),
			Name:        listName,
			Description: "Импортирован из Excel",
			Color:       "#3b82f6",
			Priority:    5,
			IsActive:    true,
			CreatedBy:   &adminIDStr,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := h.accessListRepo.Create(list); err != nil {
			log.Printf("❌ Ошибка создания списка: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать список"})
			return
		}
		log.Printf("✅ Список '%s' создан с ID: %s", listName, list.ID)
	} else {
		log.Printf("✅ Найден существующий список '%s' с ID: %s", listName, list.ID)
	}

	// Статистика
	imported := 0
	skipped := 0
	errors := 0
	var skippedPlates []string
	var errorPlates []string

	// Обрабатываем строки (пропускаем заголовок)
	for i, row := range rows {
		if i == 0 {
			// Пропускаем заголовок
			continue
		}

		if len(row) == 0 {
			continue
		}

		// Получаем гос номер (первая колонка)
		plateNumber := strings.TrimSpace(row[0])
		if plateNumber == "" {
			continue
		}

		// Очищаем номер: убираем лишние пробелы, приводим к верхнему регистру
		plateNumber = strings.ToUpper(plateNumber)
		plateNumber = strings.Join(strings.Fields(plateNumber), "")

		log.Printf("🔍 Обработка номера: '%s' (строка %d)", plateNumber, i+1)

		// Проверяем, существует ли уже такой номер в этом списке
		exists, err := h.approvedPlateRepo.CheckExistsInList(plateNumber, list.ID)
		if err != nil {
			log.Printf("❌ Ошибка проверки номера %s: %v", plateNumber, err)
			errors++
			errorPlates = append(errorPlates, plateNumber)
			continue
		}

		if exists {
			log.Printf("⏭️ Номер %s уже существует в списке, пропускаем", plateNumber)
			skipped++
			skippedPlates = append(skippedPlates, plateNumber)
			continue
		}

		// Создаем запись в approved_plates
		plate := &models.ApprovedPlate{
			ID:          uuid.New().String(),
			PlateNumber: plateNumber,
			ListID:      list.ID,
			// OrganizationID оставляем nil
			IsActive:  true,
			Notes:     "Импортирован из Excel",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := h.approvedPlateRepo.Create(plate); err != nil {
			log.Printf("❌ Ошибка добавления номера %s: %v", plateNumber, err)
			errors++
			errorPlates = append(errorPlates, plateNumber)
			continue
		}

		log.Printf("✅ Номер %s добавлен в список '%s'", plateNumber, listName)
		imported++
	}

	// Формируем ответ
	response := gin.H{
		"message": fmt.Sprintf("Импорт завершен. Добавлено: %d, пропущено: %d, ошибок: %d",
			imported, skipped, errors),
		"imported": imported,
		"skipped":  skipped,
		"errors":   errors,
		"listId":   list.ID,
		"listName": list.Name,
	}

	if len(skippedPlates) > 0 {
		response["skippedPlates"] = skippedPlates
	}
	if len(errorPlates) > 0 {
		response["errorPlates"] = errorPlates
	}

	log.Printf("📊 Итоги импорта: добавлено=%d, пропущено=%d, ошибок=%d", imported, skipped, errors)
	c.JSON(http.StatusOK, response)
}
