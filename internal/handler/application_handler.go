package handler

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"sez-checkpoint-backend/internal/models"
	"sez-checkpoint-backend/internal/repository"
)

type ApplicationHandler struct {
	appRepo      *repository.ApplicationRepository
	contractRepo *repository.ContractRepository
	userRepo     *repository.UserRepository
	approvedRepo *repository.ApprovedPlateRepository
}

func NewApplicationHandler(
	appRepo *repository.ApplicationRepository,
	contractRepo *repository.ContractRepository,
	userRepo *repository.UserRepository,
	approvedRepo *repository.ApprovedPlateRepository,
) *ApplicationHandler {
	return &ApplicationHandler{
		appRepo:      appRepo,
		contractRepo: contractRepo,
		userRepo:     userRepo,
		approvedRepo: approvedRepo,
	}
}

// Create - создание новой заявки (участник)
func (h *ApplicationHandler) Create(c *gin.Context) {
	var req models.CreateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ Ошибка парсинга запроса: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Заполните все обязательные поля",
		})
		return
	}

	log.Printf("📝 Создание заявки: номер=%s, договор=%s, список=%s, smartParking=%v",
		req.PlateNumber, req.ContractNumber, req.ListID, req.SmartParking)

	// Получаем ID текущего пользователя
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
		return
	}
	log.Printf("👤 ID пользователя: %s", userID)

	// Получаем информацию о пользователе
	user, err := h.userRepo.GetByID(userID.(string))
	if err != nil {
		log.Printf("❌ Ошибка при получении информации о пользователе: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка при получении информации о пользователе",
		})
		return
	}
	log.Printf("👤 Информация о пользователе: username=%s, организация=%v",
		user.Username, user.OrganizationID)

	// Проверяем, что у пользователя есть организация
	if user.OrganizationID == nil {
		log.Printf("❌ У пользователя %s нет организации", userID)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "У вас нет организации. Обратитесь к администратору",
		})
		return
	}

	// Проверяем существует ли договор с таким номером
	contract, err := h.contractRepo.GetByNumber(req.ContractNumber)
	if err != nil {
		log.Printf("❌ Договор %s не найден: %v", req.ContractNumber, err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Договор с таким номером не найден",
		})
		return
	}
	log.Printf("📄 Информация о договоре: ID=%s, статус=%s, организация=%s",
		contract.ID, contract.Status, contract.OrganizationID)

	// Проверяем, что договор активен
	if contract.Status != "active" {
		log.Printf("❌ Договор %s не активен, статус: %s", req.ContractNumber, contract.Status)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Договор не активен. Текущий статус: %s", contract.Status),
		})
		return
	}

	// Проверяем, что договор принадлежит организации пользователя
	if contract.OrganizationID != *user.OrganizationID {
		log.Printf("❌ Несоответствие организации: пользователь=%s, договор=%s",
			*user.OrganizationID, contract.OrganizationID)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Этот договор не принадлежит вашей организации",
		})
		return
	}
	log.Printf("✅ Организации совпадают: %s", *user.OrganizationID)

	// Проверяем, имеет ли пользователь право подавать заявку в этот список
	hasPermission, err := h.userRepo.CheckListPermission(userID.(string), req.ListID)
	if err != nil || !hasPermission {
		log.Printf("❌ У пользователя %s нет прав на список %s: %v", userID, req.ListID, err)
		c.JSON(http.StatusForbidden, gin.H{
			"error": "У вас нет права подавать заявки в этот список",
		})
		return
	}
	log.Printf("✅ Права на список подтверждены")

	// Проверяем, существует ли уже активный номер в этом списке
	existingPlate, err := h.approvedRepo.GetByPlateNumberAndListIncludeInactive(req.PlateNumber, req.ListID)
	if err == nil && existingPlate != nil && existingPlate.IsActive {
		log.Printf("❌ Номер %s уже существует в списке %s", req.PlateNumber, req.ListID)
		c.JSON(http.StatusConflict, gin.H{
			"error": "Данный номер уже есть в списке пропусков",
		})
		return
	}

	// Парсим даты
	var validFrom, validUntil *time.Time
	if req.ValidFrom != "" {
		t, err := time.Parse("2006-01-02", req.ValidFrom)
		if err == nil {
			validFrom = &t
		}
	}
	if req.ValidUntil != "" {
		t, err := time.Parse("2006-01-02", req.ValidUntil)
		if err == nil {
			validUntil = &t
		}
	}

	// Создаем заявку на КПП 1
	applicationKPP := &models.Application{
		ID:             uuid.New().String(),
		PlateNumber:    req.PlateNumber,
		VehicleBrand:   req.VehicleBrand,
		VehicleModel:   req.VehicleModel,
		VehicleColor:   req.VehicleColor,
		ContractID:     &contract.ID,
		OrganizationID: user.OrganizationID,
		ListID:         req.ListID,
		ApplicantID:    userID.(string),
		Status:         "pending",
		Destination:    "kpp1",
		ValidFrom:      validFrom,
		ValidUntil:     validUntil,
		Notes:          req.Notes,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := h.appRepo.Create(applicationKPP); err != nil {
		log.Printf("❌ Ошибка при создании заявки на КПП 1: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Не удалось создать заявку",
		})
		return
	}

	log.Printf("✅ Заявка на КПП 1 создана: ID=%s, номер=%s", applicationKPP.ID, applicationKPP.PlateNumber)

	// Если отмечен SmartParking, создаем вторую заявку
	var applicationSP *models.Application
	if req.SmartParking {
		applicationSP = &models.Application{
			ID:             uuid.New().String(),
			PlateNumber:    req.PlateNumber,
			VehicleBrand:   req.VehicleBrand,
			VehicleModel:   req.VehicleModel,
			VehicleColor:   req.VehicleColor,
			ContractID:     &contract.ID,
			OrganizationID: user.OrganizationID,
			ListID:         req.ListID,
			ApplicantID:    userID.(string),
			Status:         "pending",
			Destination:    "smartparking",
			ValidFrom:      validFrom,
			ValidUntil:     validUntil,
			Notes:          req.Notes,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		if err := h.appRepo.Create(applicationSP); err != nil {
			log.Printf("❌ Ошибка при создании заявки на SmartParking: %v", err)
			// Не возвращаем ошибку, так как заявка на КПП уже создана
		} else {
			log.Printf("✅ Заявка на SmartParking создана: ID=%s, номер=%s", applicationSP.ID, applicationSP.PlateNumber)
		}
	}

	response := gin.H{
		"message":        "Заявка успешно создана",
		"applicationKPP": applicationKPP,
	}
	if applicationSP != nil {
		response["applicationSP"] = applicationSP
	}

	c.JSON(http.StatusCreated, response)
}

// GetMyApplications - получает заявки текущего пользователя
func (h *ApplicationHandler) GetMyApplications(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
		return
	}

	applications, err := h.appRepo.GetByApplicant(userID.(string))
	if err != nil {
		log.Printf("❌ Ошибка при получении заявок пользователя %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении заявок"})
		return
	}

	c.JSON(http.StatusOK, applications)
}

// GetPendingForOperator - получает заявки для оператора КПП 1
func (h *ApplicationHandler) GetPendingForOperator(c *gin.Context) {
	applications, err := h.appRepo.GetPendingForOperator()
	if err != nil {
		log.Printf("❌ Ошибка при получении заявок для оператора: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении заявок"})
		return
	}

	c.JSON(http.StatusOK, applications)
}

// GetPendingForSmartParkingOperator - получает заявки для оператора SmartParking
func (h *ApplicationHandler) GetPendingForSmartParkingOperator(c *gin.Context) {
	applications, err := h.appRepo.GetPendingForSmartParkingOperator()
	if err != nil {
		log.Printf("❌ Ошибка при получении заявок для оператора SmartParking: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении заявок"})
		return
	}

	c.JSON(http.StatusOK, applications)
}

// GetPendingForSupervisor - получает заявки для руководителя
func (h *ApplicationHandler) GetPendingForSupervisor(c *gin.Context) {
	applications, err := h.appRepo.GetPendingForSupervisor()
	if err != nil {
		log.Printf("❌ Ошибка при получении заявок для руководителя: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении заявок"})
		return
	}

	c.JSON(http.StatusOK, applications)
}

// OperatorApprove - одобрение оператором КПП 1
func (h *ApplicationHandler) OperatorApprove(c *gin.Context) {
	appID := c.Param("id")

	operatorID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
		return
	}

	// Проверяем, что заявка предназначена для КПП 1
	app, err := h.appRepo.GetByID(appID)
	if err != nil {
		log.Printf("❌ Заявка %s не найдена: %v", appID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}

	if app.Destination != "kpp1" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Эта заявка не для КПП 1"})
		return
	}

	now := time.Now()
	operatorIDStr := operatorID.(string)

	log.Printf("🔐 Одобрение заявки КПП 1 %s оператором %s", appID, operatorIDStr)

	err = h.appRepo.UpdateStatus(appID, "operator_approved", &operatorIDStr, nil, &now, nil, "")
	if err != nil {
		log.Printf("❌ Ошибка при одобрении заявки %s: %v", appID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при одобрении заявки"})
		return
	}

	log.Printf("✅ Заявка КПП 1 %s одобрена оператором %s", appID, operatorIDStr)
	c.JSON(http.StatusOK, gin.H{"message": "Заявка одобрена оператором"})
}

// SmartParkingOperatorApprove - одобрение оператором SmartParking
func (h *ApplicationHandler) SmartParkingOperatorApprove(c *gin.Context) {
	appID := c.Param("id")

	var req struct {
		ParkingID int64 `json:"parkingId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Не указан ID парковки"})
		return
	}

	operatorID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
		return
	}

	// Проверяем, что заявка предназначена для SmartParking
	app, err := h.appRepo.GetByID(appID)
	if err != nil {
		log.Printf("❌ Заявка %s не найдена: %v", appID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}

	if app.Destination != "smartparking" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Эта заявка не для SmartParking"})
		return
	}

	operatorIDStr := operatorID.(string)
	log.Printf("🔐 Одобрение заявки SmartParking %s оператором %s, parkingId=%d", appID, operatorIDStr, req.ParkingID)

	// Удаляем заявку
	if err := h.appRepo.HardDelete(appID); err != nil {
		log.Printf("❌ Ошибка при удалении заявки %s: %v", appID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении заявки"})
		return
	}

	log.Printf("✅ Заявка SmartParking %s одобрена, номер %s отправлен в Parqour, заявка удалена",
		appID, app.PlateNumber)

}

// SupervisorApprove - одобрение руководителем
func (h *ApplicationHandler) SupervisorApprove(c *gin.Context) {
	appID := c.Param("id")

	supervisorID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
		return
	}

	// Получаем информацию о заявке
	app, err := h.appRepo.GetByID(appID)
	if err != nil {
		log.Printf("❌ Заявка %s не найдена: %v", appID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}

	if app.Status != "operator_approved" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Заявка должна быть сначала одобрена оператором"})
		return
	}

	now := time.Now()
	supervisorIDStr := supervisorID.(string)

	log.Printf("🔐 Утверждение заявки %s руководителем %s", appID, supervisorIDStr)

	// Начинаем транзакцию
	tx, err := h.appRepo.BeginTx()
	if err != nil {
		log.Printf("❌ Ошибка начала транзакции: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при утверждении заявки"})
		return
	}
	defer tx.Rollback()

	// 1. Сначала добавляем номер в утвержденные
	approvedPlate := &models.ApprovedPlate{
		ID:             uuid.New().String(),
		PlateNumber:    app.PlateNumber,
		VehicleBrand:   app.VehicleBrand,
		VehicleModel:   app.VehicleModel,
		VehicleColor:   app.VehicleColor,
		ContractID:     app.ContractID,
		OrganizationID: app.OrganizationID,
		ListID:         app.ListID,
		ApplicationID:  &app.ID,
		ApprovedBy:     &supervisorIDStr,
		ValidFrom:      app.ValidFrom,
		ValidUntil:     app.ValidUntil,
		IsActive:       true,
		Notes:          app.Notes,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := h.approvedRepo.CreateTx(tx, approvedPlate); err != nil {
		log.Printf("❌ Ошибка при добавлении в approved_plates: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при добавлении номера в список"})
		return
	}

	// 2. Удаляем заявку
	if err := h.appRepo.DeleteTx(tx, appID); err != nil {
		log.Printf("❌ Ошибка при удалении заявки %s: %v", appID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении заявки"})
		return
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		log.Printf("❌ Ошибка при коммите транзакции: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при сохранении данных"})
		return
	}

	log.Printf("✅ Заявка %s утверждена руководителем %s, номер %s добавлен в список, заявка удалена",
		appID, supervisorIDStr, app.PlateNumber)
	c.JSON(http.StatusOK, gin.H{
		"message": "Заявка утверждена, номер добавлен в список пропусков, заявка удалена",
		"plate":   approvedPlate,
	})
}

// Reject - отклонение заявки оператором КПП 1
func (h *ApplicationHandler) Reject(c *gin.Context) {
	appID := c.Param("id")

	var req struct {
		Reason string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.Reason = ""
	}

	operatorID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
		return
	}

	operatorIDStr := operatorID.(string)

	log.Printf("🔐 Отклонение заявки %s оператором %s, причина: %s", appID, operatorIDStr, req.Reason)

	err := h.appRepo.UpdateStatus(appID, "rejected", &operatorIDStr, nil, nil, nil, req.Reason)
	if err != nil {
		log.Printf("❌ Ошибка при отклонении заявки %s: %v", appID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при отклонении заявки"})
		return
	}

	log.Printf("✅ Заявка %s отклонена оператором %s", appID, operatorIDStr)
	c.JSON(http.StatusOK, gin.H{"message": "Заявка отклонена"})
}

// GetAllApplications - получение всех заявок (только для админа)
func (h *ApplicationHandler) GetAllApplications(c *gin.Context) {
	// Проверяем, что текущий пользователь - администратор
	currentUserRole, exists := c.Get("roleID")
	if !exists || currentUserRole != 1 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Недостаточно прав"})
		return
	}

	// Получаем параметры фильтрации из запроса
	status := c.Query("status")
	organizationID := c.Query("organizationId")
	listID := c.Query("listId")
	fromDate := c.Query("from")
	toDate := c.Query("to")

	log.Printf("📊 Получение всех заявок с фильтрами: status=%s, org=%s, list=%s",
		status, organizationID, listID)

	applications, err := h.appRepo.GetAllFiltered(status, organizationID, listID, fromDate, toDate)
	if err != nil {
		log.Printf("❌ Ошибка при получении всех заявок: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при получении заявок"})
		return
	}

	log.Printf("✅ Найдено %d заявок", len(applications))
	c.JSON(http.StatusOK, applications)
}

// AdminApproveAsOperator - одобрение заявки от имени оператора (только для админа)
func (h *ApplicationHandler) AdminApproveAsOperator(c *gin.Context) {
	appID := c.Param("id")

	// Проверяем права администратора
	currentUserRole, exists := c.Get("roleID")
	if !exists || currentUserRole != 1 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Недостаточно прав"})
		return
	}

	adminID, _ := c.Get("userID")
	adminIDStr := adminID.(string)
	now := time.Now()

	log.Printf("🔐 Администратор %s одобряет заявку %s от имени оператора", adminIDStr, appID)

	err := h.appRepo.UpdateStatus(appID, "operator_approved", &adminIDStr, nil, &now, nil, "")
	if err != nil {
		log.Printf("❌ Ошибка при одобрении заявки администратором: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при одобрении заявки"})
		return
	}

	log.Printf("✅ Администратор %s одобрил заявку %s от имени оператора", adminIDStr, appID)
	c.JSON(http.StatusOK, gin.H{"message": "Заявка одобрена от имени оператора"})
}

// AdminApproveAsSupervisor - утверждение заявки от имени руководителя (только для админа)
func (h *ApplicationHandler) AdminApproveAsSupervisor(c *gin.Context) {
	appID := c.Param("id")

	// Проверяем права администратора
	currentUserRole, exists := c.Get("roleID")
	if !exists || currentUserRole != 1 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Недостаточно прав"})
		return
	}

	// Получаем информацию о заявке
	app, err := h.appRepo.GetByID(appID)
	if err != nil {
		log.Printf("❌ Заявка %s не найдена: %v", appID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}

	// Проверяем статус заявки
	if app.Status != "operator_approved" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Заявка должна быть сначала одобрена оператором"})
		return
	}

	adminID, _ := c.Get("userID")
	adminIDStr := adminID.(string)
	now := time.Now()

	log.Printf("🔐 Администратор %s утверждает заявку %s от имени руководителя", adminIDStr, appID)

	// Начинаем транзакцию
	tx, err := h.appRepo.BeginTx()
	if err != nil {
		log.Printf("❌ Ошибка начала транзакции: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при утверждении заявки"})
		return
	}
	defer tx.Rollback()

	// 1. Сначала добавляем номер в утвержденные
	approvedPlate := &models.ApprovedPlate{
		ID:             uuid.New().String(),
		PlateNumber:    app.PlateNumber,
		VehicleBrand:   app.VehicleBrand,
		VehicleModel:   app.VehicleModel,
		VehicleColor:   app.VehicleColor,
		ContractID:     app.ContractID,
		OrganizationID: app.OrganizationID,
		ListID:         app.ListID,
		ApplicationID:  &app.ID,
		ApprovedBy:     &adminIDStr,
		ValidFrom:      app.ValidFrom,
		ValidUntil:     app.ValidUntil,
		IsActive:       true,
		Notes:          app.Notes,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := h.approvedRepo.CreateTx(tx, approvedPlate); err != nil {
		log.Printf("❌ Ошибка при добавлении в approved_plates: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при добавлении номера в список"})
		return
	}

	// 2. Удаляем заявку
	if err := h.appRepo.DeleteTx(tx, appID); err != nil {
		log.Printf("❌ Ошибка при удалении заявки %s: %v", appID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении заявки"})
		return
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		log.Printf("❌ Ошибка при коммите транзакции: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при сохранении данных"})
		return
	}

	log.Printf("✅ Администратор %s утвердил заявку %s от имени руководителя. Номер %s добавлен в список %s, заявка удалена",
		adminIDStr, appID, app.PlateNumber, app.ListID)
	c.JSON(http.StatusOK, gin.H{
		"message": "Заявка утверждена, номер добавлен в список пропусков, заявка удалена",
		"plate":   approvedPlate,
	})
}

// AdminReject - отклонение заявки (только для админа)
func (h *ApplicationHandler) AdminReject(c *gin.Context) {
	appID := c.Param("id")

	// Проверяем права администратора
	currentUserRole, exists := c.Get("roleID")
	if !exists || currentUserRole != 1 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Недостаточно прав"})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Reason = ""
	}

	adminID, _ := c.Get("userID")
	adminIDStr := adminID.(string)
	now := time.Now()

	log.Printf("🔐 Администратор %s отклоняет заявку %s, причина: %s", adminIDStr, appID, req.Reason)

	err := h.appRepo.UpdateStatus(appID, "rejected", &adminIDStr, nil, nil, &now, req.Reason)
	if err != nil {
		log.Printf("❌ Ошибка при отклонении заявки администратором: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при отклонении заявки"})
		return
	}

	log.Printf("✅ Администратор %s отклонил заявку %s", adminIDStr, appID)
	c.JSON(http.StatusOK, gin.H{"message": "Заявка отклонена"})
}

// GetByID - получение заявки по ID (доступно для всех авторизованных)
func (h *ApplicationHandler) GetByID(c *gin.Context) {
	appID := c.Param("id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID заявки не указан"})
		return
	}

	// Получаем информацию о пользователе для проверки прав
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
		return
	}

	userRole, _ := c.Get("roleID")

	log.Printf("🔍 Получение заявки %s пользователем %s", appID, userID)

	application, err := h.appRepo.GetByID(appID)
	if err != nil {
		log.Printf("❌ Ошибка при получении заявки %s: %v", appID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}

	// Проверка прав доступа:
	// - Администратор (role 1) может смотреть любые заявки
	// - Оператор (role 2) и руководитель (role 3) могут смотреть все заявки (для работы)
	// - Участник (role 4) может смотреть только свои заявки
	if userRole == 4 && application.ApplicantID != userID {
		log.Printf("❌ Пользователь %s не имеет доступа к заявке %s", userID, appID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Нет доступа к этой заявке"})
		return
	}

	log.Printf("✅ Заявка %s успешно получена", appID)
	c.JSON(http.StatusOK, application)
}

// AdminDeleteApplication - удаление заявки (только для админа)
func (h *ApplicationHandler) AdminDeleteApplication(c *gin.Context) {
	appID := c.Param("id")

	// Проверяем права администратора
	currentUserRole, exists := c.Get("roleID")
	if !exists || currentUserRole != 1 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Недостаточно прав"})
		return
	}

	adminID, _ := c.Get("userID")
	adminIDStr := adminID.(string)

	log.Printf("🗑️ Администратор %s удаляет заявку %s", adminIDStr, appID)

	// Получаем информацию о заявке перед удалением
	app, err := h.appRepo.GetByID(appID)
	if err != nil {
		log.Printf("❌ Заявка %s не найдена: %v", appID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}

	// Удаляем заявку
	if err := h.appRepo.HardDelete(appID); err != nil {
		log.Printf("❌ Ошибка при удалении заявки %s: %v", appID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении заявки"})
		return
	}

	log.Printf("✅ Администратор %s удалил заявку %s (номер %s)", adminIDStr, appID, app.PlateNumber)
	c.JSON(http.StatusOK, gin.H{"message": "Заявка успешно удалена"})
}

// SupervisorReject - отклонение заявки руководителем (для заявок в статусе operator_approved)
func (h *ApplicationHandler) SupervisorReject(c *gin.Context) {
	appID := c.Param("id")

	var req struct {
		Reason string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.Reason = "Отклонено руководителем"
	}

	supervisorID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
		return
	}

	supervisorIDStr := supervisorID.(string)

	log.Printf("🔐 Руководитель %s отклоняет заявку %s, причина: %s", supervisorIDStr, appID, req.Reason)

	// Получаем информацию о заявке
	app, err := h.appRepo.GetByID(appID)
	if err != nil {
		log.Printf("❌ Заявка %s не найдена: %v", appID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}

	// Проверяем, что заявка в статусе operator_approved
	if app.Status != "operator_approved" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Можно отклонить только заявки, одобренные оператором",
		})
		return
	}

	// Отклоняем заявку
	err = h.appRepo.UpdateStatus(appID, "rejected", nil, &supervisorIDStr, nil, nil, req.Reason)
	if err != nil {
		log.Printf("❌ Ошибка при отклонении заявки %s: %v", appID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при отклонении заявки"})
		return
	}

	log.Printf("✅ Заявка %s отклонена руководителем %s", appID, supervisorIDStr)
	c.JSON(http.StatusOK, gin.H{"message": "Заявка отклонена"})
}

// DeleteSmartParkingApplication - удаление заявки оператором SmartParking
func (h *ApplicationHandler) DeleteSmartParkingApplication(c *gin.Context) {
	appID := c.Param("id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID заявки не указан"})
		return
	}

	operatorID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
		return
	}

	log.Printf("🗑️ Оператор SmartParking %s удаляет заявку %s", operatorID, appID)

	// Получаем заявку
	app, err := h.appRepo.GetByID(appID)
	if err != nil {
		log.Printf("❌ Заявка %s не найдена: %v", appID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}

	// Проверяем, что заявка предназначена для SmartParking
	if app.Destination != "smartparking" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Эта заявка не для SmartParking"})
		return
	}

	// Удаляем заявку
	if err := h.appRepo.HardDelete(appID); err != nil {
		log.Printf("❌ Ошибка при удалении заявки %s: %v", appID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при удалении заявки"})
		return
	}

	log.Printf("✅ Оператор SmartParking %s удалил заявку %s (номер %s)", operatorID, appID, app.PlateNumber)
	c.JSON(http.StatusOK, gin.H{"message": "Заявка успешно удалена"})
}
