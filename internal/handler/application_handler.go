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

	log.Printf("📝 Создание заявки: номер=%s, договор=%s, список=%s",
		req.PlateNumber, req.ContractNumber, req.ListID)

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

	// Парсим даты
	var validFrom, validUntil *time.Time
	if req.ValidFrom != "" {
		t, err := time.Parse("2006-01-02", req.ValidFrom)
		if err == nil {
			validFrom = &t
		} else {
			log.Printf("⚠️ Ошибка парсинга даты validFrom: %v", err)
		}
	}
	if req.ValidUntil != "" {
		t, err := time.Parse("2006-01-02", req.ValidUntil)
		if err == nil {
			validUntil = &t
		} else {
			log.Printf("⚠️ Ошибка парсинга даты validUntil: %v", err)
		}
	}

	// Создаем заявку
	application := &models.Application{
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
		ValidFrom:      validFrom,
		ValidUntil:     validUntil,
		Notes:          req.Notes,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	log.Printf("📦 Создание объекта заявки: ID=%s, Plate=%s, Org=%s, List=%s",
		application.ID, application.PlateNumber, *application.OrganizationID, application.ListID)

	if err := h.appRepo.Create(application); err != nil {
		log.Printf("❌ Ошибка при создании заявки: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Не удалось создать заявку",
		})
		return
	}

	log.Printf("✅ Заявка успешно создана: ID=%s, номер=%s", application.ID, application.PlateNumber)
	c.JSON(http.StatusCreated, gin.H{
		"message":     "Заявка успешно создана",
		"application": application,
	})
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

// GetPendingForOperator - получает заявки для оператора
func (h *ApplicationHandler) GetPendingForOperator(c *gin.Context) {
	applications, err := h.appRepo.GetPendingForOperator()
	if err != nil {
		log.Printf("❌ Ошибка при получении заявок для оператора: %v", err)
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

// OperatorApprove - одобрение оператором
func (h *ApplicationHandler) OperatorApprove(c *gin.Context) {
	appID := c.Param("id")

	operatorID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
		return
	}

	now := time.Now()
	operatorIDStr := operatorID.(string)

	log.Printf("🔐 Одобрение заявки %s оператором %s", appID, operatorIDStr)

	err := h.appRepo.UpdateStatus(appID, "operator_approved", &operatorIDStr, nil, &now, nil, "")
	if err != nil {
		log.Printf("❌ Ошибка при одобрении заявки %s: %v", appID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при одобрении заявки"})
		return
	}

	log.Printf("✅ Заявка %s одобрена оператором %s", appID, operatorIDStr)
	c.JSON(http.StatusOK, gin.H{"message": "Заявка одобрена оператором"})
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

// Reject - отклонение заявки
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
